package health

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"time"
)

// handler handles health check requests.
type handler struct {
	logger    *slog.Logger
	gitRev    string
	startTime time.Time
}

// New creates a new health handler.
func New(logger *slog.Logger, gitRev string, startTime time.Time) http.Handler {
	return &handler{
		logger:    logger,
		gitRev:    gitRev,
		startTime: startTime,
	}
}

// ServeHTTP processes health check requests.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Handling health check", "path", r.URL.Path, "method", r.Method)

	response := map[string]any{
		"status":     "healthy",
		"git_rev":    h.gitRev,
		"uptime":     time.Since(h.startTime).Seconds(),
		"goroutines": runtime.NumGoroutine(),
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode health response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
