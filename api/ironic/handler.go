package ironic

import (
	"log/slog"
	"net/http"

	"github.com/metal3-community/metal-boot/internal/ironic"
)

// handler handles metrics requests.
type handler struct {
	logger      *slog.Logger
	socketProxy *ironic.SocketProxy
}

// New creates a new metrics handler.
func New(logger *slog.Logger, socketPath string) http.Handler {
	return &handler{
		logger:      logger,
		socketProxy: ironic.NewSocketProxy(logger, socketPath),
	}
}

// ServeHTTP processes ironic requests.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Handling ironic request", "path", r.URL.Path, "method", r.Method)

	h.socketProxy.ServeHTTP(w, r)
}
