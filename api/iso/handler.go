package iso

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"github.com/metal3-community/metal-boot/internal/backend"
	"github.com/metal3-community/metal-boot/internal/config"
	"github.com/metal3-community/metal-boot/internal/dhcp/data"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	defaultConsoles = "console=ttyAMA0 console=ttyS0 console=tty0 console=tty1 console=ttyS1"
)

// Context key type for patch data.
type patchCtxKeyType string

const isoPatchCtxKey patchCtxKeyType = "iso-patch"

// withPatch adds patch data to the context.
func withPatch(ctx context.Context, patch []byte) context.Context {
	return context.WithValue(ctx, isoPatchCtxKey, patch)
}

// getPatch retrieves patch data from the context.
func getPatch(ctx context.Context) []byte {
	patch, ok := ctx.Value(isoPatchCtxKey).([]byte)
	if !ok {
		return nil
	}
	return patch
}

// isoHandler is a struct that contains the necessary fields to patch an ISO file with
// relevant information for the Tink worker.
type isoHandler struct {
	Backend           backend.BackendReader
	ExtraKernelParams []string
	Logger            logr.Logger
	// MagicString is the string pattern that will be matched
	// in the source iso before patching. The field can be set
	// during build time by setting this field.
	// Ref: https://github.com/tinkerbell/hook/blob/main/linuxkit-templates/hook.template.yaml
	MagicString string
	// SourceISO is the source url where the unmodified iso lives.
	// It must be a valid url.URL{} object and must have a url.URL{}.Scheme of HTTP or HTTPS.
	SourceISO         string
	Syslog            string
	UseTLS            bool
	GRPCAddr          string
	StaticIPAMEnabled bool
	// parsedURL derives a url.URL from the SourceISO field.
	// It needed for validation of SourceISO and easier modification.
	parsedURL       *url.URL
	magicStrPadding []byte
}

func New(
	logger logr.Logger,
	cfg *config.Config,
	backend backend.BackendReader,
) http.Handler {
	return &isoHandler{
		Backend:           backend,
		ExtraKernelParams: cfg.IpxeHttpScript.ExtraKernelArgs,
		Logger:            logger,
		MagicString:       cfg.Iso.MagicString,
		SourceISO:         cfg.Iso.Url,
		Syslog:            cfg.Dhcp.SyslogIP,
		UseTLS:            cfg.IpxeHttpScript.UseTLS,
		StaticIPAMEnabled: cfg.Dhcp.StaticIPAMEnabled,
	}
}

// ServeHTTP processes metrics requests.
func (h *isoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Logger.V(1).Info("Handling metrics request", "path", r.URL.Path, "method", r.Method)

	target, err := url.Parse(h.SourceISO)
	if err != nil {
		h.Logger.Error(err, "failed to parse SourceISO", "sourceISO", h.SourceISO)
		return
	}
	h.parsedURL = target

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			// r.SetXForwarded()
			r.SetURL(target)
		},
	}

	proxy.Transport = h
	proxy.FlushInterval = -1

	h.magicStrPadding = bytes.Repeat([]byte{' '}, len(h.MagicString))

	proxy.ServeHTTP(w, r)
}

// Copy implements the internal.CopyBuffer interface.
// This implementation allows us to inspect and patch content on its way to the client without buffering the entire response
// in memory. This allows memory use to be constant regardless of the size of the response.
func (h *isoHandler) Copy(
	ctx context.Context,
	dst io.Writer,
	src io.Reader,
	buf []byte,
) (int64, error) {
	if len(buf) == 0 {
		buf = make([]byte, 32*1024)
	}
	var written int64
	for {
		nr, rerr := src.Read(buf)
		if rerr != nil && rerr != io.EOF &&
			rerr != context.Canceled { //nolint: errorlint // going to defer to the stdlib on this one.
			h.Logger.Info("httputil: ReverseProxy read error during body copy: %v", rerr)
		}
		if nr > 0 {
			// This is the patching check and handling.
			b := buf[:nr]
			i := bytes.Index(b, []byte(h.MagicString))
			if i != -1 {
				dup := make([]byte, len(b))
				copy(dup, b)
				copy(dup[i:], h.magicStrPadding)
				copy(dup[i:], getPatch(ctx))
				b = dup
			}
			nw, werr := dst.Write(b)
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, werr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				rerr = nil
			}
			return written, rerr
		}
	}
}

// RoundTrip is a method on the Handler struct that implements the http.RoundTripper interface.
// This method is called by the internal.NewSingleHostReverseProxy to handle the incoming request.
// The method is responsible for validating the incoming request and getting the source ISO.
func (h *isoHandler) RoundTrip(req *http.Request) (*http.Response, error) {
	log := h.Logger.WithValues(
		"method",
		req.Method,
		"inboundURI",
		req.RequestURI,
		"remoteAddr",
		req.RemoteAddr,
	)
	log.V(1).Info("starting the ISO patching HTTP handler")

	if filepath.Ext(req.URL.Path) != ".iso" {
		log.Info("extension not supported, only supported extension is '.iso'")
		return &http.Response{
			Status: fmt.Sprintf(
				"%d %s",
				http.StatusNotFound,
				http.StatusText(http.StatusNotFound),
			),
			StatusCode: http.StatusNotFound,
			Body:       http.NoBody,
			Request:    req,
		}, nil
	}

	// The incoming request url is expected to have the mac address present.
	// Fetch the mac and validate if there's a hardware object
	// associated with the mac.
	//
	// We serve the iso only if this validation passes.
	ha, err := getMAC(req.URL.Path)
	if err != nil {
		log.Info("unable to parse mac address in the URL path", "error", err)
		return &http.Response{
			Status: fmt.Sprintf(
				"%d %s",
				http.StatusBadRequest,
				http.StatusText(http.StatusBadRequest),
			),
			StatusCode: http.StatusBadRequest,
			Body:       http.NoBody,
			Request:    req,
		}, nil
	}

	fac, dhcpData, err := h.getFacility(req.Context(), ha, h.Backend)
	if err != nil {
		log.Info("unable to get the hardware object", "error", err, "mac", ha)
		if apierrors.IsNotFound(err) {
			return &http.Response{
				Status: fmt.Sprintf(
					"%d %s",
					http.StatusNotFound,
					http.StatusText(http.StatusNotFound),
				),
				StatusCode: http.StatusNotFound,
				Body:       http.NoBody,
				Request:    req,
			}, nil
		}
		return &http.Response{
			Status: fmt.Sprintf(
				"%d %s",
				http.StatusInternalServerError,
				http.StatusText(http.StatusInternalServerError),
			),
			StatusCode: http.StatusInternalServerError,
			Body:       http.NoBody,
			Request:    req,
		}, nil
	}
	// The hardware object doesn't contain a dedicated field for consoles right now and
	// historically the facility is used as a way to define consoles on a per Hardware basis.
	var consoles string
	switch {
	case fac != "" && strings.Contains(fac, "console="):
		consoles = fmt.Sprintf("facility=%s", fac)
	case fac != "":
		consoles = fmt.Sprintf("facility=%s %s", fac, defaultConsoles)
	default:
		consoles = defaultConsoles
	}
	// The patch is added to the request context so that it can be used in the Copy method.
	req = req.WithContext(
		withPatch(
			req.Context(),
			[]byte(h.constructPatch(consoles, ha.String(), dhcpData)),
		),
	)

	// The internal.NewSingleHostReverseProxy takes the incoming request url and adds the path to the target (h.SourceISO).
	// This function is more than a pass through proxy. The MAC address in the url path is required to do hardware lookups using the backend reader
	// and is not used when making http calls to the target (h.SourceISO). All valid requests are passed through to the target.
	req.URL.Path = h.parsedURL.Path
	log = log.WithValues("outboundURL", req.URL.String())

	// RoundTripper needs a Transport to execute a HTTP transaction
	// For our use case the default transport will suffice.
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Error(err, "issue getting the source ISO", "sourceIso", h.SourceISO)
		return nil, err
	}
	// by setting this header we are telling the logging middleware to not log its default log message.
	// we do this because there are a lot of partial content requests and it allow this handler to take care of logging.
	resp.Header.Set("X-Global-Logging", "false")

	if resp.StatusCode == http.StatusPartialContent {
		// 0.002% of the time we log a 206 request message.
		// In testing, it was observed that about 3000 HTTP 206 requests are made per ISO mount.
		// 0.002% gives us about 5 - 10, log messages per ISO mount.
		// We're optimizing for showing "enough" log messages so that progress can be observed.
		if p := randomPercentage(100000); p < 0.002 {
			log.Info("206 status code response", "sourceIso", h.SourceISO, "status", resp.Status)
		}
	} else {
		log.Info("response received", "sourceIso", h.SourceISO, "status", resp.Status)
	}

	log.V(1).Info("roundtrip complete")

	return resp, nil
}

func (h *isoHandler) constructPatch(console, mac string, d *data.DHCP) string {
	syslogHost := fmt.Sprintf("syslog_host=%s", h.Syslog)
	grpcAuthority := fmt.Sprintf("grpc_authority=%s", h.GRPCAddr)
	tinkerbellTLS := fmt.Sprintf("tinkerbell_tls=%v", h.UseTLS)
	workerID := fmt.Sprintf("worker_id=%s", mac)
	vlanID := func() string {
		if d != nil && d.VLANID != "" {
			return fmt.Sprintf("vlan_id=%s", d.VLANID)
		}
		return ""
	}()
	hwAddr := fmt.Sprintf("hw_addr=%s", mac)
	all := []string{
		strings.Join(h.ExtraKernelParams, " "),
		console,
		vlanID,
		hwAddr,
		syslogHost,
		grpcAuthority,
		tinkerbellTLS,
		workerID,
	}
	if h.StaticIPAMEnabled {
		all = append(all, parseIPAM(d))
	}

	return strings.Join(all, " ")
}

func getMAC(urlPath string) (net.HardwareAddr, error) {
	mac := path.Base(path.Dir(urlPath))
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse URL path: %s , the second to last element in the URL path must be a valid mac address, err: %w",
			urlPath,
			err,
		)
	}

	return hw, nil
}

func (h *isoHandler) getFacility(
	ctx context.Context,
	mac net.HardwareAddr,
	br backend.BackendReader,
) (string, *data.DHCP, error) {
	if br == nil {
		return "", nil, errors.New("backend is nil")
	}

	d, n, err := br.GetByMac(ctx, mac)
	if err != nil {
		return "", nil, err
	}

	return n.Facility, d, nil
}

func randomPercentage(precision int64) float64 {
	random, err := rand.Int(rand.Reader, big.NewInt(precision))
	if err != nil {
		return 0
	}

	return float64(random.Int64()) / float64(precision)
}

func parseIPAM(d *data.DHCP) string {
	if d == nil {
		return ""
	}
	// return format is ipam=<mac-address>:<vlan-id>:<ip-address>:<netmask>:<gateway>:<hostname>:<dns>:<search-domains>:<ntp>
	ipam := make([]string, 9)
	ipam[0] = func() string {
		m := d.MACAddress.String()

		return strings.ReplaceAll(m, ":", "-")
	}()
	ipam[1] = func() string {
		if d.VLANID != "" {
			return d.VLANID
		}
		return ""
	}()
	ipam[2] = func() string {
		if d.IPAddress.Compare(netip.Addr{}) != 0 {
			return d.IPAddress.String()
		}
		return ""
	}()
	ipam[3] = func() string {
		if d.SubnetMask != nil {
			return net.IP(d.SubnetMask).String()
		}
		return ""
	}()
	ipam[4] = func() string {
		if d.DefaultGateway.Compare(netip.Addr{}) != 0 {
			return d.DefaultGateway.String()
		}
		return ""
	}()
	ipam[5] = d.Hostname
	ipam[6] = func() string {
		var nameservers []string
		for _, e := range d.NameServers {
			nameservers = append(nameservers, e.String())
		}
		if len(nameservers) > 0 {
			return strings.Join(nameservers, ",")
		}

		return ""
	}()
	ipam[7] = func() string {
		if len(d.DomainSearch) > 0 {
			return strings.Join(d.DomainSearch, ",")
		}

		return ""
	}()
	ipam[8] = func() string {
		var ntp []string
		for _, e := range d.NTPServers {
			ntp = append(ntp, e.String())
		}
		if len(ntp) > 0 {
			return strings.Join(ntp, ",")
		}

		return ""
	}()

	return fmt.Sprintf("ipam=%s", strings.Join(ipam, ":"))
}
