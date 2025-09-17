package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"github.com/metal3-community/metal-boot/api"
	"github.com/metal3-community/metal-boot/api/health"
	"github.com/metal3-community/metal-boot/api/images/talos"
	"github.com/metal3-community/metal-boot/api/ipxe"
	"github.com/metal3-community/metal-boot/api/ironic"
	"github.com/metal3-community/metal-boot/api/iso"
	"github.com/metal3-community/metal-boot/api/metrics"
	"github.com/metal3-community/metal-boot/api/redfish"
	"github.com/metal3-community/metal-boot/internal/backend"
	"github.com/metal3-community/metal-boot/internal/backend/dnsmasq"
	"github.com/metal3-community/metal-boot/internal/backend/dnsmasq/lease"
	"github.com/metal3-community/metal-boot/internal/backend/unifi"
	"github.com/metal3-community/metal-boot/internal/config"
	"github.com/metal3-community/metal-boot/internal/dhcp/handler/proxy"
	"github.com/metal3-community/metal-boot/internal/dhcp/handler/reservation"
	dhcpServer "github.com/metal3-community/metal-boot/internal/dhcp/server"
	ironicManager "github.com/metal3-community/metal-boot/internal/ironic"
	"github.com/metal3-community/metal-boot/internal/tftp"
	"golang.org/x/sync/errgroup"
)

var (
	// GitRev is the git revision of the build. It is set by the Makefile.
	GitRev = "unknown (use make)"

	startTime = time.Now()
)

//go:generate go run ../../internal/ipxe/generate
func main() {
	// Load configuration
	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Create structured logger from config
	logger := cfg.Log
	logger.Info("Metal Boot starting", "version", GitRev, "start_time", startTime)

	// Create readerBackend
	readerBackend, err := createReaderBackend(context.Background(), logger, cfg)
	if err != nil {
		logger.Error(err, "failed to create reader backend")
		os.Exit(1)
	}

	// Create pwrBackend
	pwrBackend, err := createPowerBackend(context.Background(), logger, cfg)
	if err != nil {
		logger.Error(err, "failed to create backend")
		os.Exit(1)
	}

	// Set up graceful shutdown context
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
	)
	defer cancel()

	// Start all services
	if err := startServices(ctx, cfg, logger, readerBackend, pwrBackend); err != nil {
		logger.Error(err, "failed to start services")
		os.Exit(1)
	}

	logger.Info("Metal Boot shutdown complete")
}

// createPowerBackend initializes and starts the backend service.
func createPowerBackend(
	ctx context.Context,
	log logr.Logger,
	cfg *config.Config,
) (backend.BackendPower, error) {
	backend, err := unifi.NewRemote(ctx, log, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend: %w", err)
	}
	return backend, nil
}

func createReaderBackend(
	ctx context.Context,
	log logr.Logger,
	cfg *config.Config,
) (backend.BackendReader, error) {
	backend, err := dnsmasq.NewBackend(log, dnsmasq.Config{
		RootDir:    cfg.Dnsmasq.RootDirectory,
		TFTPServer: cfg.Dhcp.TftpAddress,
		HTTPServer: cfg.Dhcp.IpxeBinaryUrl.GetUrl().Host,

		AutoAssignEnabled: cfg.Dnsmasq.AutoAssignEnabled,
		IPPoolStart:       cfg.Dnsmasq.IPPoolStart,
		IPPoolEnd:         cfg.Dnsmasq.IPPoolEnd,
		DefaultLeaseTime:  cfg.Dnsmasq.DefaultLeaseTime,
		DefaultGateway:    cfg.Dnsmasq.DefaultGateway,
		DefaultSubnet:     cfg.Dnsmasq.DefaultSubnet,
		DefaultDNS:        cfg.Dnsmasq.DefaultDNS,
		DefaultDomain:     cfg.Dnsmasq.DefaultDomain,
	})
	if err != nil {
		log.Error(err, "failed to create dnsmasq backend")
		return nil, fmt.Errorf("failed to create dnsmasq backend: %w", err)
	}
	if err := backend.Sync(ctx); err != nil {
		log.Error(err, "failed to sync dnsmasq backend")
		return nil, fmt.Errorf("failed to sync dnsmasq backend: %w", err)
	}
	return backend, nil
}

// startServices initializes and starts all configured services.
func startServices(
	ctx context.Context,
	cfg *config.Config,
	logger logr.Logger,
	readerBackend backend.BackendReader,
	pwrBackend backend.BackendPower,
) error {
	g, ctx := errgroup.WithContext(ctx)

	// Start Ironic supervisor if enabled
	if cfg.Ironic.SupervisorEnabled {
		logger.Info("Ironic supervisor enabled", "socket_path", cfg.Ironic.Socket.Path)
		if err := startIronicSupervisor(ctx, g, cfg, logger); err != nil {
			return fmt.Errorf("failed to start Ironic supervisor: %w", err)
		}
	}

	if cfg.Ironic.Rpc.Enabled {
		if err := startJsonRpcServer(ctx, g, cfg); err != nil {
			return fmt.Errorf("failed to start JSON-RPC server: %w", err)
		}
	}

	// Start HTTP API server
	if err := startHTTPServer(ctx, g, cfg, logger, readerBackend, pwrBackend); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	// Start TFTP server if enabled
	if cfg.Tftp.Enabled {
		logger.Info("TFTP server enabled", "root_directory", cfg.Tftp.RootDirectory)
		startTFTPServer(ctx, g, cfg, logger, readerBackend)
	}

	// Start DHCP server if enabled
	if cfg.Dhcp.Enabled {
		logger.Info(
			"DHCP server enabled",
			"interface",
			cfg.Dhcp.Interface,
			"address",
			cfg.Dhcp.Address,
		)
		if err := startDHCPServer(ctx, g, cfg, logger, readerBackend); err != nil {
			return fmt.Errorf("failed to start DHCP server: %w", err)
		}
	}

	// Wait for all services or shutdown signal
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("service error: %w", err)
	}

	return nil
}

func startJsonRpcServer(
	ctx context.Context,
	g *errgroup.Group,
	cfg *config.Config,
) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	httpHandler := ironic.New(logger, cfg.Ironic.Rpc.Socket.Path)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Ironic.Rpc.Port),
		Handler: httpHandler,
		// Add reasonable timeouts
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	g.Go(func() error {
		return httpServer.ListenAndServe()
	})

	g.Go(func() error {
		<-ctx.Done()
		logger.Info("shutting down HTTP server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- httpServer.Shutdown(ctx)
		}()

		select {
		case err := <-done:
			if err != nil {
				logger.ErrorContext(
					ctx,
					"error during HTTP server shutdown",
					slog.Any("error", err),
				)
			}
			return err
		case <-shutdownCtx.Done():
			logger.ErrorContext(ctx,
				"HTTP server shutdown timeout forced shutdown after 30 seconds",
			)
			return errors.New("HTTP server shutdown timeout")
		}
	})

	return nil
}

// startHTTPServer configures and starts the HTTP API server.
func startHTTPServer(
	ctx context.Context,
	g *errgroup.Group,
	cfg *config.Config,
	logger logr.Logger,
	readerBackend backend.BackendReader,
	pwrBackend backend.BackendPower,
) error {
	// Create structured logger for HTTP server
	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create API instance
	apiServer := api.New(cfg, slogger)

	// Configure API handlers
	configureAPIHandlers(apiServer, cfg, logger, readerBackend, pwrBackend, slogger)

	// Start the server in a goroutine
	bindAddr := fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)
	logger.Info("starting HTTP server", "addr", bindAddr)

	g.Go(func() error {
		return apiServer.Start()
	})

	// Handle graceful shutdown
	g.Go(func() error {
		<-ctx.Done()
		logger.Info("shutting down HTTP server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- apiServer.Shutdown()
		}()

		select {
		case err := <-done:
			if err != nil {
				logger.Error(err, "error during HTTP server shutdown")
			}
			return err
		case <-shutdownCtx.Done():
			logger.Error(
				errors.New("HTTP server shutdown timeout"),
				"forced shutdown after 30 seconds",
			)
			return errors.New("HTTP server shutdown timeout")
		}
	})

	return nil
}

// configureAPIHandlers sets up all HTTP API route handlers.
func configureAPIHandlers(
	apiServer *api.Api,
	cfg *config.Config,
	logger logr.Logger,
	readerBackend backend.BackendReader,
	pwrBackend backend.BackendPower,
	slogger *slog.Logger,
) {
	// Add health check handler
	apiServer.AddHandler("/healthcheck", health.New(slogger, GitRev, startTime))
	logger.V(1).Info("registered health check handler", "path", "/healthcheck")

	// Add metrics handler
	apiServer.AddHandler("/metrics", metrics.New(slogger))
	logger.V(1).Info("registered metrics handler", "path", "/metrics")

	// Add Redfish handler
	apiServer.AddHandler("/redfish/v1/", redfish.New(slogger, cfg, readerBackend, pwrBackend))
	logger.V(1).Info("registered Redfish handler", "path", "/redfish/v1/")

	apiServer.AddHandler("/v1/", ironic.New(slogger, cfg.Ironic.Socket.Path))
	logger.V(1).Info("registered Ironic handler", "path", "/v1/")

	// Add iPXE handlers if enabled
	if cfg.IpxeHttpScript.Enabled {
		apiServer.AddHandler("/", ipxe.New(slogger, cfg, readerBackend))
		logger.Info("iPXE HTTP script handler enabled", "path", "/")
	}

	// Add ISO handler if enabled
	if cfg.Iso.Enabled {
		apiServer.AddHandler("/iso/", iso.New(logger, cfg, readerBackend))
		logger.Info("ISO handler enabled", "path", "/iso/")
	}

	// Add Talos image handler if enabled
	if cfg.Talos.Enabled {
		apiServer.AddHandler("/images/talos/", talos.New(slogger, &cfg.Talos))
		logger.Info("Talos image handler enabled", "path", "/images/talos/")
	}
}

// startTFTPServer configures and starts the TFTP server.
func startTFTPServer(
	ctx context.Context,
	g *errgroup.Group,
	cfg *config.Config,
	logger logr.Logger,
	backend backend.BackendReader,
) {
	ts := &tftp.Server{
		Logger:        logger.WithName("tftp"),
		RootDirectory: cfg.Tftp.RootDirectory,
		Patch:         cfg.Tftp.IpxePatch,
	}

	logger.Info("starting TFTP server", "addr", cfg.Address)
	g.Go(func() error {
		return ts.ListenAndServe(
			ctx,
			netip.AddrPortFrom(netip.MustParseAddr(cfg.Address), 69),
			backend,
		)
	})
}

// startDHCPServer configures and starts the DHCP server.
func startDHCPServer(
	ctx context.Context,
	g *errgroup.Group,
	cfg *config.Config,
	logger logr.Logger,
	backend backend.BackendReader,
) error {
	dh, err := createDHCPHandler(cfg, logger, backend)
	if err != nil {
		return fmt.Errorf("failed to create DHCP handler: %w", err)
	}

	logger.Info("starting DHCP server", "bind_addr", cfg.Dhcp.Address)
	g.Go(func() error {
		return serveDHCP(ctx, cfg, logger, dh)
	})

	// Start lease cleanup routine if using reservation handler with lease management
	// if !cfg.Dhcp.ProxyEnabled && (cfg.Dhcp.LeaseFile != "" || cfg.Dhcp.ConfigFile != "") {
	// 	g.Go(func() error {
	// 		return runLeaseCleanup(ctx, logger, dh)
	// 	})
	// }

	return nil
}

// serveDHCP runs the DHCP server with proper connection handling.
func serveDHCP(
	ctx context.Context,
	cfg *config.Config,
	logger logr.Logger,
	dh dhcpServer.Handler,
) error {
	dhcpAddr, err := netip.ParseAddrPort(
		fmt.Sprintf("%s:%d", cfg.Dhcp.Address, cfg.Dhcp.Port),
	)
	if err != nil {
		return fmt.Errorf("invalid DHCP bind address: %w", err)
	}

	conn, err := server4.NewIPv4UDPConn(
		cfg.Dhcp.Interface,
		net.UDPAddrFromAddrPort(dhcpAddr),
	)
	if err != nil {
		return fmt.Errorf("failed to create DHCP connection: %w", err)
	}
	defer conn.Close()

	ds := &dhcpServer.DHCP{
		Logger:   logger,
		Conn:     conn,
		Handlers: []dhcpServer.Handler{dh},
	}

	// Handle shutdown gracefully
	go func() {
		<-ctx.Done()
		logger.Info("shutting down DHCP server")
		conn.Close()
		ds.Close()
	}()

	return ds.Serve(ctx)
}

// createDHCPHandler creates a DHCP handler with proper configuration.
func createDHCPHandler(
	cfg *config.Config,
	logger logr.Logger,
	backend backend.BackendReader,
) (dhcpServer.Handler, error) {
	return dhcpHandler(cfg, context.Background(), logger, backend)
}

// dhcpHandler configures a DHCP proxy handler with network boot capabilities.
func dhcpHandler(
	c *config.Config,
	_ context.Context,
	log logr.Logger,
	backend backend.BackendReader,
) (dhcpServer.Handler, error) {
	pktIP, err := netip.ParseAddr(c.Dhcp.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid bind address: %w", err)
	}
	tftpIP, err := netip.ParseAddrPort(fmt.Sprintf("%s:%d", c.Dhcp.TftpAddress, c.Dhcp.TftpPort))
	if err != nil {
		return nil, fmt.Errorf("invalid tftp address for DHCP server: %w", err)
	}
	c.Dhcp.IpxeBinaryUrl.GetUrl()
	httpBinaryURL := c.Dhcp.IpxeBinaryUrl.GetUrl()
	if _, err := url.Parse(httpBinaryURL.String()); err != nil {
		return nil, fmt.Errorf("invalid http ipxe binary url: %w", err)
	}

	ipxeScript := func(d *dhcpv4.DHCPv4) *url.URL {
		return c.Dhcp.IpxeBinaryUrl.GetUrl("/boot.ipxe")
	}

	var dh dhcpServer.Handler

	if c.Dhcp.ProxyEnabled {
		dh = &proxy.Handler{
			Backend: backend,
			IPAddr:  pktIP,
			Log:     log,
			Netboot: proxy.Netboot{
				IPXEBinServerTFTP: tftpIP,
				IPXEBinServerHTTP: httpBinaryURL,
				IPXEScriptURL:     ipxeScript,
				Enabled:           true,
			},
			OTELEnabled:      false, // Disabled since we removed OpenTelemetry
			AutoProxyEnabled: true,
		}
	} else {
		leaseBackend, err := lease.NewLeaseManager(
			log,
			filepath.Join(c.Dnsmasq.RootDirectory, "dnsmasq.leases"),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create lease manager: %w", err)
		}
		// Use reservation handler with lease management
		reservationHandler := &reservation.Handler{
			Backend:      backend,
			LeaseBackend: leaseBackend,
			IPAddr:       pktIP,
			Log:          log,
			Netboot: reservation.Netboot{
				IPXEBinServerTFTP: tftpIP,
				IPXEBinServerHTTP: httpBinaryURL,
				IPXEScriptURL:     ipxeScript,
				Enabled:           true,
			},
			OTELEnabled: false, // Disabled since we removed OpenTelemetry
		}

		dh = reservationHandler
	}
	return dh, nil
}

// startIronicSupervisor configures and starts the Ironic process supervisor.
func startIronicSupervisor(
	ctx context.Context,
	g *errgroup.Group,
	cfg *config.Config,
	logger logr.Logger,
) error {
	// Create structured logger for Ironic supervisor
	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create Ironic configuration, setting only values from external configuration
	ironicConfig := &ironicManager.Config{
		// ProcessManager configuration (not part of TOML config)
		SocketPath: cfg.Ironic.Socket.Path,
		ConfigPath: cfg.Ironic.ConfigPath,
		SkipDBSync: cfg.Ironic.SkipDBSync,

		// API section - set configured values
		API: ironicManager.APIConfig{
			UnixSocket:     cfg.Ironic.Socket.Path,
			UnixSocketMode: cfg.Ironic.Socket.Mode,
			Port:           cfg.Port,
			PublicEndpoint: cfg.Ironic.PublicEndpoint,
		},

		// Database section - set configured values
		Database: ironicManager.DatabaseConfig{
			Connection: cfg.Ironic.DatabaseConnection,
		},

		// Deploy section - set configured values
		Deploy: ironicManager.DeployConfig{
			HTTPURL:         cfg.Ironic.Url,
			ExternalHTTPURL: cfg.Ironic.PublicEndpoint,
		},

		// Conductor section - set configured values
		Conductor: ironicManager.ConductorConfig{
			APIURL: cfg.Ironic.Url,
		},

		// JSONRPC section - set configured values
		JSONRPC: ironicManager.JSONRPCConfig{
			Enabled:        cfg.Ironic.Rpc.Enabled,
			UnixSocket:     cfg.Ironic.Rpc.Socket.Path,
			UnixSocketMode: cfg.Ironic.Rpc.Socket.Mode,
			Port:           cfg.Ironic.Rpc.Port,
		},

		// ServiceCatalog section - set configured values
		ServiceCatalog: ironicManager.ServiceCatalogConfig{
			EndpointOverride: cfg.Ironic.PublicEndpoint,
		},
	}

	// Create and start the process manager
	processManager := ironicManager.NewProcessManager(ctx, slogger, ironicConfig)

	// Start the supervisor in a goroutine
	g.Go(func() error {
		logger.Info("starting Ironic supervisor")
		return processManager.Start()
	})

	// Handle graceful shutdown
	g.Go(func() error {
		<-ctx.Done()
		logger.Info("shutting down Ironic supervisor")
		processManager.Shutdown()
		return nil
	})

	return nil
}
