package script

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"path"

	"github.com/bmcpi/pibmc/internal/backend"
	"github.com/bmcpi/pibmc/internal/config"
)

// scriptHandler handles iPXE script requests.
type scriptHandler struct {
	logger  *slog.Logger
	config  *config.Config
	backend backend.BackendReader
}

// New creates a new iPXE script handler.
func New(logger *slog.Logger, cfg *config.Config, backend backend.BackendReader) http.Handler {
	return &scriptHandler{
		logger:  logger,
		config:  cfg,
		backend: backend,
	}
}

// ServeHTTP handles iPXE script requests.
// It is expected that the request path is /<mac address>/auto.ipxe.
func (h *scriptHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqLogger := h.logger.With("method", r.Method, "path", r.URL.Path)
	reqLogger.Debug("Handling iPXE script request")

	if path.Base(r.URL.Path) != "auto.ipxe" {
		reqLogger.Info("URL path not supported")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ctx := r.Context()

	// Should we serve a custom ipxe script?
	// This gates serving PXE file by
	// 1. the existence of a hardware record in tink server
	// AND
	// 2. the network.interfaces[].netboot.allow_pxe value, in the tink server hardware record, equal to true
	// This allows serving custom ipxe scripts, starting up into OSIE or other installation environments
	// without a tink workflow present.

	// Try to get the MAC address from the URL path, if not available get the source IP address.
	if ha, err := getMAC(r.URL.Path); err == nil {
		hw, err := h.getByMac(ctx, ha)
		if err != nil && h.config.IpxeHttpScript.StaticIPXEEnabled {
			reqLogger.Info("Serving static ipxe script", "mac", ha, "error", err)
			h.serveStaticIPXEScript(w)
			return
		}
		if err != nil || !hw.AllowNetboot {
			w.WriteHeader(http.StatusNotFound)
			reqLogger.Info(
				"The hardware data for this machine, or lack thereof, does not allow it to pxe",
				"client", ha,
				"error", err,
			)
			return
		}
		h.serveBootScript(ctx, w, path.Base(r.URL.Path), hw)
		return
	}

	if ip, err := getIP(r.RemoteAddr); err == nil {
		hw, err := h.getByIP(ctx, ip)
		if err != nil && h.config.IpxeHttpScript.StaticIPXEEnabled {
			reqLogger.Info("Serving static ipxe script", "client", r.RemoteAddr, "error", err)
			h.serveStaticIPXEScript(w)
			return
		}
		if err != nil || !hw.AllowNetboot {
			w.WriteHeader(http.StatusNotFound)
			reqLogger.Info(
				"The hardware data for this machine, or lack thereof, does not allow it to pxe",
				"client", r.RemoteAddr,
				"error", err,
			)
			return
		}
		h.serveBootScript(ctx, w, path.Base(r.URL.Path), hw)
		return
	}

	// If we get here, we were unable to get the MAC address from the URL path or the source IP address.
	w.WriteHeader(http.StatusNotFound)
	reqLogger.Info(
		"Unable to get the MAC address from the URL path or the source IP address",
		"client", r.RemoteAddr,
		"url_path", r.URL.Path,
	)
}

type data struct {
	AllowNetboot  bool // If true, the client will be provided netboot options in the DHCP offer/ack.
	Console       string
	MACAddress    net.HardwareAddr
	Arch          string
	VLANID        string
	WorkflowID    string
	Facility      string
	IPXEScript    string
	IPXEScriptURL *url.URL
	OSIE          OSIE
}

// OSIE or OS Installation Environment is the data about where the OSIE parts are located.
type OSIE struct {
	// BaseURL is the URL where the OSIE parts are located.
	BaseURL *url.URL
	// Kernel is the name of the kernel file.
	Kernel string
	// Initrd is the name of the initrd file.
	Initrd string
}

// getByMac uses the backend.BackendReader to get the (hardware) data and then
// translates it to the script.Data struct.
func (h *scriptHandler) getByMac(ctx context.Context, mac net.HardwareAddr) (data, error) {
	if h.backend == nil {
		return data{}, errors.New("backend is nil")
	}
	d, n, err := h.backend.GetByMac(ctx, mac)
	if err != nil {
		return data{}, err
	}

	return data{
		AllowNetboot:  n.AllowNetboot,
		Console:       "",
		MACAddress:    d.MACAddress,
		Arch:          d.Arch,
		VLANID:        d.VLANID,
		WorkflowID:    d.MACAddress.String(),
		Facility:      n.Facility,
		IPXEScript:    n.IPXEScript,
		IPXEScriptURL: n.IPXEScriptURL,
		OSIE:          OSIE(n.OSIE),
	}, nil
}

func (h *scriptHandler) getByIP(ctx context.Context, ip net.IP) (data, error) {
	if h.backend == nil {
		return data{}, errors.New("backend is nil")
	}
	d, n, err := h.backend.GetByIP(ctx, ip)
	if err != nil {
		return data{}, err
	}

	return data{
		AllowNetboot:  n.AllowNetboot,
		Console:       "",
		MACAddress:    d.MACAddress,
		Arch:          d.Arch,
		VLANID:        d.VLANID,
		WorkflowID:    d.MACAddress.String(),
		Facility:      n.Facility,
		IPXEScript:    n.IPXEScript,
		IPXEScriptURL: n.IPXEScriptURL,
		OSIE:          OSIE(n.OSIE),
	}, nil
}

func (h *scriptHandler) serveStaticIPXEScript(w http.ResponseWriter) {
	h.logger.Info("Serving static iPXE script")
	// TODO: Implement static script generation
	// For now, return a simple message
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("#!ipxe\necho Static iPXE script not implemented yet\nreboot\n"))
}

func (h *scriptHandler) serveBootScript(
	ctx context.Context,
	w http.ResponseWriter,
	name string,
	hw data,
) {
	reqLogger := h.logger.With("script_name", name)
	reqLogger.Debug("Serving boot script")

	var script []byte
	// check if the custom script should be used
	if hw.IPXEScriptURL != nil || hw.IPXEScript != "" {
		name = "custom.ipxe"
	}

	switch name {
	case "auto.ipxe":
		s, err := h.defaultScript(hw)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			reqLogger.Error("Error with default ipxe script", "error", err)
			return
		}
		script = []byte(s)
	case "custom.ipxe":
		cs, err := h.customScript(hw)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			reqLogger.Error("Error with custom ipxe script", "error", err)
			return
		}
		script = []byte(cs)
	default:
		w.WriteHeader(http.StatusNotFound)
		reqLogger.Error("Boot script not found", "script", name)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if _, err := w.Write(script); err != nil {
		reqLogger.Error("Unable to write boot script", "error", err)
		return
	}

	reqLogger.Info("Boot script served successfully", "script_length", len(script))
}

func (h *scriptHandler) defaultScript(hw data) (string, error) {
	// TODO: Implement default script generation based on the original logic
	// For now, return a placeholder
	return "#!ipxe\necho Default iPXE script not implemented yet\nreboot\n", nil
}

// customScript returns the custom script or chain URL if defined in the hardware data otherwise an error.
func (h *scriptHandler) customScript(hw data) (string, error) {
	if chain := hw.IPXEScriptURL; chain != nil && chain.String() != "" {
		if chain.Scheme != "http" && chain.Scheme != "https" {
			return "", fmt.Errorf("invalid URL scheme: %v", chain.Scheme)
		}
		return fmt.Sprintf("#!ipxe\nchain %s\n", chain.String()), nil
	}
	if script := hw.IPXEScript; script != "" {
		return script, nil
	}

	return "", errors.New("no custom script or chain defined in the hardware data")
}

func getIP(remoteAddr string) (net.IP, error) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return net.IP{}, fmt.Errorf("error parsing client address: %w: client: %v", err, remoteAddr)
	}
	ip := net.ParseIP(host)
	return ip, nil
}

func getMAC(urlPath string) (net.HardwareAddr, error) {
	mac := path.Base(path.Dir(urlPath))
	ha, err := net.ParseMAC(mac)
	if err != nil {
		return net.HardwareAddr{}, fmt.Errorf(
			"URL path not supported, the second to last element in the URL path must be a valid mac address, err: %w",
			err,
		)
	}
	return ha, nil
}
