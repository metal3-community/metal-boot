// Package ipxe implements HTTP handlers for iPXE services.
package ipxe

import (
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/metal3-community/metal-boot/api/ipxe/binary"
	"github.com/metal3-community/metal-boot/api/ipxe/script"
	"github.com/metal3-community/metal-boot/api/ipxe/static"
	"github.com/metal3-community/metal-boot/internal/backend"
	"github.com/metal3-community/metal-boot/internal/config"
)

// handler routes iPXE requests to the appropriate sub-handlers.
type handler struct {
	logger        *slog.Logger
	config        *config.Config
	binaryHandler http.Handler
	scriptHandler http.Handler
	staticHandler http.Handler
}

// New creates a new iPXE router handler.
func New(logger *slog.Logger, cfg *config.Config, backend backend.BackendReader) http.Handler {
	return &handler{
		logger:        logger,
		config:        cfg,
		binaryHandler: binary.New(logger.With("component", "binary"), cfg),
		scriptHandler: script.New(logger.With("component", "script"), cfg, backend),
		staticHandler: static.New(logger.With("component", "static"), cfg),
	}
}

// ServeHTTP routes requests to the appropriate handler based on the requested file.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqLogger := h.logger.With("method", r.Method, "path", r.URL.Path)
	reqLogger.Debug("Routing iPXE request")

	basePath := filepath.Base(r.URL.Path)

	// Check if it's a known iPXE binary file
	if binary.Match(basePath) {
		reqLogger.Debug("Routing to binary handler")
		h.binaryHandler.ServeHTTP(w, r)
		return
	}

	// Check if it's an iPXE script request
	if basePath == "auto.ipxe" || basePath == "boot.ipxe" {
		reqLogger.Debug("Routing to script handler")
		h.scriptHandler.ServeHTTP(w, r)
		return
	}

	// Default to static file handler
	reqLogger.Debug("Routing to static handler")
	h.staticHandler.ServeHTTP(w, r)
}
