package metrics

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// handler handles metrics requests.
type handler struct {
	logger *slog.Logger
}

// New creates a new metrics handler.
func New(logger *slog.Logger) http.Handler {
	return &handler{
		logger: logger,
	}
}

// ServeHTTP processes metrics requests.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Handling metrics request", "path", r.URL.Path, "method", r.Method)

	// Use the Prometheus handler
	promhttp.Handler().ServeHTTP(w, r)
}
