package ipxe

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bmcpi/pibmc/internal/config"
)

func TestNew(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{}

	handler := New(logger, cfg, nil)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
}

func TestHandler_ServeHTTP_RoutingLogic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		Static: config.StaticConfig{
			RootDirectory: "/tmp", // Use /tmp for testing
		},
	}

	handler := New(logger, cfg, nil)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "script request",
			path:           "/auto.ipxe",
			expectedStatus: http.StatusNotFound, // No backend configured, so script handler returns 404
		},
		{
			name:           "static file request",
			path:           "/some-file.txt",
			expectedStatus: http.StatusNotFound, // File doesn't exist in /tmp
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
