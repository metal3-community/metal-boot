package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/bmcpi/pibmc/api/redfish"
	"github.com/bmcpi/pibmc/internal/backend/remote"
	"github.com/bmcpi/pibmc/internal/config"
	"github.com/bmcpi/pibmc/internal/dhcp/handler"
	"github.com/bmcpi/pibmc/internal/dhcp/handler/proxy"
	dhcpServer "github.com/bmcpi/pibmc/internal/dhcp/server"
	"github.com/bmcpi/pibmc/internal/ipxe/http"
	"github.com/bmcpi/pibmc/internal/ipxe/ihttp"
	"github.com/bmcpi/pibmc/internal/ipxe/script"
	"github.com/bmcpi/pibmc/internal/ipxe/static"
	"github.com/bmcpi/pibmc/internal/iso"
	"github.com/bmcpi/pibmc/internal/metric"
	"github.com/bmcpi/pibmc/internal/otel"
	itftp "github.com/bmcpi/pibmc/internal/tftp"
	"github.com/go-logr/logr"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"golang.org/x/sync/errgroup"
)

var (
	// GitRev is the git revision of the build. It is set by the Makefile.
	GitRev = "unknown (use make)"

	startTime = time.Now()
)

//go:generate go tool oapi-codegen -package redfish -o api/redfish/server.gen.go -generate std-http-server,models openapi.yaml
func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		panic(err)
	}

	log := cfg.Log

	backend, err := defaultBackend(context.Background(), log, cfg)
	if err != nil {
		log.Error(err, "failed to create backend")
		panic(fmt.Errorf("failed to create backend: %w", err))
	}

	ctx, done := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
	)
	defer done()

	oCfg := otel.Config{
		Servicename: "pibmc",
		Endpoint:    cfg.Otel.Endpoint,
		Insecure:    cfg.Otel.Insecure,
		Logger:      log,
	}
	ctx, otelShutdown, err := otel.Init(ctx, oCfg)
	if err != nil {
		log.Error(err, "failed to initialize OpenTelemetry")
		panic(err)
	}
	defer otelShutdown()
	metric.Init()

	g, ctx := errgroup.WithContext(ctx)

	handlers := make(http.HandlerMapping)
	registrations := make([]http.RegistrationFunc, 0)

	// create the http server before our merged handlers
	tp := parseTrustedProxies(cfg.TrustedProxies)
	httpServer := &http.Config{
		GitRev:         GitRev,
		StartTime:      startTime,
		Logger:         log,
		TrustedProxies: tp,
	}

	if cfg.IpxeHttpScript.StaticFilesEnabled {
		httpServer.IHttpHandler = ihttp.HandlerFunc(cfg)
		httpServer.StaticHandler = static.HandlerFunc(cfg)
	}

	// http ipxe script
	if cfg.IpxeHttpScript.Enabled {
		if len(cfg.Images.ImageURLs) > 0 {
			g.Go(func() error {
				return static.DownloadImages(cfg)
			})
		}

		osieUrl, err := cfg.GetOsieUrl()
		if err != nil {
			log.Error(err, "failed to get osie url")
		}
		jh := script.Handler{
			Logger:                log,
			Backend:               backend,
			OSIEURL:               osieUrl.String(),
			ExtraKernelParams:     cfg.IpxeHttpScript.ExtraKernelArgs,
			PublicSyslogFQDN:      cfg.Dhcp.SyslogIP,
			TinkServerTLS:         cfg.IpxeHttpScript.TinkServerUseTLS,
			TinkServerInsecureTLS: cfg.IpxeHttpScript.TinkServerInsecureTLS,
			TinkServerGRPCAddr:    cfg.IpxeHttpScript.TinkServer,
			IPXEScriptRetries:     cfg.IpxeHttpScript.Retries,
			IPXEScriptRetryDelay:  cfg.IpxeHttpScript.RetryDelay,
			StaticIPXEEnabled:     cfg.IpxeHttpScript.StaticIPXEEnabled,
		}

		httpServer.ScriptHandler = jh.HandlerFunc()

		handlers["/"] = httpServer.HandlerFunc()
	}

	if cfg.Iso.Enabled {
		ih := iso.Handler{
			Logger:             log,
			Backend:            backend,
			SourceISO:          cfg.Iso.Url,
			ExtraKernelParams:  cfg.IpxeHttpScript.ExtraKernelArgs,
			Syslog:             cfg.Dhcp.SyslogIP,
			TinkServerTLS:      false,
			TinkServerGRPCAddr: "",
			StaticIPAMEnabled:  false,
			MagicString: func() string {
				return cfg.Iso.MagicString
			}(),
		}
		isoHandler, err := ih.HandlerFunc()
		if err != nil {
			panic(fmt.Errorf("failed to create iso handler: %w", err))
		}
		handlers["/iso/"] = isoHandler
	}

	redfishServer := redfish.NewRedfishServer(cfg, backend)
	registrations = append(registrations, redfishServer.Register)

	if len(handlers) > 0 {
		// start the http server for ipxe binaries and scripts
		bindAddr := fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)
		log.Info("serving http", "addr", bindAddr, "trusted_proxies", tp)
		g.Go(func() error {
			return httpServer.ServeHTTP(ctx, bindAddr, handlers, registrations...)
		})
	}

	if cfg.Tftp.Enabled {
		ts := &itftp.Server{
			Logger:        log.WithName("tftp"),
			RootDirectory: cfg.Tftp.RootDirectory,
			Patch:         cfg.Tftp.IpxePatch,
		}

		g.Go(func() error {
			return ts.ListenAndServe(
				ctx,
				netip.AddrPortFrom(netip.MustParseAddr(cfg.Address), 69),
				backend,
			)
		})
	}

	if cfg.Dhcp.Enabled {
		dh, err := dhcpHandler(cfg, ctx, log, backend)
		if err != nil {
			log.Error(err, "failed to create dhcp listener")
			panic(fmt.Errorf("failed to create dhcp listener: %w", err))
		}
		log.Info("starting dhcp server", "bind_addr", cfg.Dhcp.Address)
		g.Go(func() error {
			dhcpIp, err := netip.ParseAddrPort(
				fmt.Sprintf("%s:%d", cfg.Dhcp.Address, cfg.Dhcp.Port),
			)
			if err != nil {
				return fmt.Errorf("invalid bind address: %w", err)
			}

			bindAddr, err := netip.ParseAddrPort(dhcpIp.String())
			if err != nil {
				panic(fmt.Errorf("invalid tftp address for DHCP server: %w", err))
			}
			conn, err := server4.NewIPv4UDPConn(
				cfg.Dhcp.Interface,
				net.UDPAddrFromAddrPort(bindAddr),
			)
			if err != nil {
				panic(err)
			}
			defer conn.Close()
			ds := &dhcpServer.DHCP{Logger: log, Conn: conn, Handlers: []dhcpServer.Handler{dh}}

			go func() {
				<-ctx.Done()
				conn.Close()
				ds.Conn.Close()
				ds.Close()
			}()
			return ds.Serve(ctx)
		})
	}

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Error(err, "failed running all services")
		panic(err)
	}
	log.Info("shutting down")
}

func defaultBackend(
	ctx context.Context,
	log logr.Logger,
	config *config.Config,
) (handler.BackendStore, error) {
	f, err := remote.NewRemote(log, config)
	// f, err := persist.NewPersist(log, config)
	if err != nil {
		return nil, err
	}

	go f.Start(ctx)

	return f, nil
}

func parseTrustedProxies(trustedProxies string) (result []string) {
	for cidr := range strings.SplitSeq(trustedProxies, ",") {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		_, _, err := net.ParseCIDR(cidr)
		if err != nil {
			// Its not a cidr, but maybe its an IP
			if ip := net.ParseIP(cidr); ip != nil {
				if ip.To4() != nil {
					cidr += "/32"
				} else {
					cidr += "/128"
				}
			} else {
				// not an IP, panic
				panic("invalid ip cidr in TRUSTED_PROXIES cidr=" + cidr)
			}
		}
		result = append(result, cidr)
	}

	return result
}

func dhcpHandler(
	c *config.Config,
	ctx context.Context,
	log logr.Logger,
	backend handler.BackendReader,
) (dhcpServer.Handler, error) {
	// 1. create the handler
	// 2. create the backend
	// 3. add the backend to the handler
	pktIP, err := netip.ParseAddr(c.Dhcp.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid bind address: %w", err)
	}
	tftpIP, err := netip.ParseAddrPort(fmt.Sprintf("%s:%d", c.Dhcp.TftpAddress, c.Dhcp.TftpPort))
	if err != nil {
		return nil, fmt.Errorf("invalid tftp address for DHCP server: %w", err)
	}
	httpBinaryURL := &url.URL{
		Scheme: c.Dhcp.IpxeBinaryUrl.Scheme,
		Host:   fmt.Sprintf("%s:%d", c.Dhcp.IpxeBinaryUrl.Address, c.Dhcp.IpxeBinaryUrl.Port),
		Path:   c.Dhcp.IpxeBinaryUrl.Path,
	}
	if _, err := url.Parse(httpBinaryURL.String()); err != nil {
		return nil, fmt.Errorf("invalid http ipxe binary url: %w", err)
	}

	httpScriptURL, err := c.GetIpxeHttpUrl()
	if err != nil {
		return nil, fmt.Errorf("failed to get ipxe http url: %w", err)
	}

	if _, err := url.Parse(httpScriptURL.String()); err != nil {
		return nil, fmt.Errorf("invalid http ipxe script url: %w", err)
	}

	ipxeScript := func(*dhcpv4.DHCPv4) *url.URL {
		return httpScriptURL
	}

	ipxeScript = func(d *dhcpv4.DHCPv4) *url.URL {
		u := *httpScriptURL
		p := path.Base(u.Path)
		u.Path = path.Join(path.Dir(u.Path), d.ClientHWAddr.String(), p)
		return &u
	}

	// backend, err := c.backend(ctx, log)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create backend: %w", err)
	// }

	dh := &proxy.Handler{
		Backend: backend,
		IPAddr:  pktIP,
		Log:     log,
		Netboot: proxy.Netboot{
			IPXEBinServerTFTP: tftpIP,
			IPXEBinServerHTTP: httpBinaryURL,
			IPXEScriptURL:     ipxeScript,
			Enabled:           true,
		},
		OTELEnabled:      true,
		AutoProxyEnabled: true,
	}
	return dh, nil
}
