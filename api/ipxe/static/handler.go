package static

import (
	"log/slog"
	"net/http"

	"github.com/bmcpi/pibmc/internal/config"
)

// handler handles static file requests.
type handler struct {
	logger *slog.Logger
	config *config.Config
}

// New creates a new static files handler.
func New(logger *slog.Logger, cfg *config.Config) http.Handler {
	return &handler{
		logger: logger,
		config: cfg,
	}
}

// ServeHTTP handles static file requests.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqLogger := h.logger.With("method", r.Method, "path", r.URL.Path)
	reqLogger.Debug("Handling static file request")

	// Use the built-in file server for the configured static directory
	fileServer := http.FileServer(http.Dir(h.config.Static.RootDirectory))
	fileServer.ServeHTTP(w, r)

	reqLogger.Info("Static file served")
}
