package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/metal3-community/metal-boot/internal/config"
	sloghttp "github.com/samber/slog-http"
	"github.com/sebest/xff"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HandlerMapping is a map of routes to http.HandlerFuncs.
type HandlerMapping map[string]http.Handler

type RegistrationFunc = func(router *http.ServeMux)

// Api represents the HTTP API server with all its dependencies.
type Api struct {
	config     *config.Config
	logger     *slog.Logger
	httpServer *http.Server
	handlers   HandlerMapping
}

// New creates a new Api instance with the given configuration.
func New(cfg *config.Config, logger *slog.Logger) *Api {
	return &Api{
		config:   cfg,
		logger:   logger,
		handlers: make(HandlerMapping),
	}
}

func (a *Api) AddHandler(path string, handler http.Handler) {
	if handler != nil {
		a.handlers[path] = otelhttp.WithRouteTag(path, handler)
	} else {
		a.logger.Warn("Attempted to add nil handler", "path", path)
	}
}

// Start initializes all dependencies and starts the HTTP server.
func (a *Api) Start(registrations ...RegistrationFunc) error {
	// Setup HTTP routes
	mux := http.NewServeMux()

	for path, handler := range a.handlers {
		mux.Handle(path, handler)
	}

	// wrap the mux with an OpenTelemetry interceptor
	httpHandler := otelhttp.NewHandler(mux, "ironic-http")

	trustedProxies := strings.Split(a.config.TrustedProxies, ",")
	if len(trustedProxies) > 0 && trustedProxies[0] != "" {
		xffmw, _ := xff.New(xff.Options{
			AllowedSubnets: trustedProxies,
		})
		httpHandler = xffmw.Handler(httpHandler)
	}

	config := sloghttp.Config{
		// Basic logging
		WithRequestID:      true,
		WithUserAgent:      true,
		WithRequestBody:    false, // Too verbose for iPXE binaries
		WithResponseBody:   false,
		WithRequestHeader:  false,
		WithResponseHeader: false,

		// Filter health checks and other noise
		Filters: []sloghttp.Filter{
			sloghttp.IgnorePathContains("/healthcheck"),
			sloghttp.IgnorePathContains("/metrics"),
		},
	}

	// Apply recovery middleware first
	httpHandler = sloghttp.Recovery(httpHandler)

	// Apply logging middleware
	httpHandler = sloghttp.NewWithConfig(a.logger, config)(httpHandler)

	// Create and configure HTTP server
	a.httpServer = &http.Server{
		Addr:    a.getAddress(),
		Handler: httpHandler,
		// Add reasonable timeouts
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	a.logger.Info("Starting HTTP server", "address", a.httpServer.Addr)

	// Start server - this blocks
	err := a.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		a.logger.Error("HTTP server failed to start", "error", err)
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (a *Api) Shutdown() error {
	if a.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		a.logger.Info("Shutting down HTTP server...")
		err := a.httpServer.Shutdown(ctx)
		if err != nil {
			a.logger.Error("Failed to shutdown HTTP server gracefully", "error", err)
			return err
		}

		a.httpServer = nil
		a.logger.Info("HTTP server shutdown complete")
	}

	return nil
}

// getAddress returns the server address from config.
func (a *Api) getAddress() string {
	return fmt.Sprintf("%s:%d", a.config.Address, a.config.Port)
}
