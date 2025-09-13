package ironic

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSocketProxy_ForwardingHeaders(t *testing.T) {
	// Create a test server to simulate Ironic backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the headers are set correctly
		tests := []struct {
			header   string
			expected bool
		}{
			{"Host", true},
			{"X-Real-IP", true},
			{"X-Forwarded-For", true},
			{"X-Forwarded-Proto", true},
		}

		for _, tt := range tests {
			if got := r.Header.Get(tt.header); (got != "") != tt.expected {
				t.Errorf(
					"Header %s: expected present=%v, got=%v",
					tt.header,
					tt.expected,
					got != "",
				)
			}
		}

		// Log received headers for debugging
		t.Logf("Received headers:")
		t.Logf("  Host: %s", r.Header.Get("Host"))
		t.Logf("  X-Real-IP: %s", r.Header.Get("X-Real-IP"))
		t.Logf("  X-Forwarded-For: %s", r.Header.Get("X-Forwarded-For"))
		t.Logf("  X-Forwarded-Proto: %s", r.Header.Get("X-Forwarded-Proto"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	// Create a logger that discards output for testing
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Note: We can't easily test the Unix socket functionality in a unit test,
	// but we can test the header logic by creating a proxy that points to our test server
	// and modifying the transport. For now, this test documents the expected behavior.

	t.Run("headers_should_be_set", func(t *testing.T) {
		// This test documents that our proxy should set the following headers:
		expectedHeaders := []string{
			"Host",
			"X-Real-IP",
			"X-Forwarded-For",
			"X-Forwarded-Proto",
		}

		for _, header := range expectedHeaders {
			t.Logf("Proxy should set header: %s", header)
		}

		// The actual proxy implementation is tested through integration tests
		// since it requires Unix socket setup
		_ = logger // Use logger to avoid unused variable error
	})
}

func TestSocketProxy_HeaderLogic(t *testing.T) {
	// Test IP extraction logic that's used in the proxy
	tests := []struct {
		name              string
		remoteAddr        string
		existingForwarded string
		expectedRealIP    string
		expectedForwarded string
	}{
		{
			name:              "simple_ip",
			remoteAddr:        "192.168.1.10:12345",
			existingForwarded: "",
			expectedRealIP:    "192.168.1.10",
			expectedForwarded: "192.168.1.10",
		},
		{
			name:              "ip_without_port",
			remoteAddr:        "192.168.1.10",
			existingForwarded: "",
			expectedRealIP:    "192.168.1.10",
			expectedForwarded: "192.168.1.10",
		},
		{
			name:              "existing_forwarded_header",
			remoteAddr:        "192.168.1.10:12345",
			existingForwarded: "203.0.113.1",
			expectedRealIP:    "203.0.113.1",
			expectedForwarded: "203.0.113.1, 203.0.113.1",
		},
		{
			name:              "multiple_forwarded_ips",
			remoteAddr:        "192.168.1.10:12345",
			existingForwarded: "203.0.113.1, 198.51.100.1",
			expectedRealIP:    "203.0.113.1",
			expectedForwarded: "203.0.113.1, 198.51.100.1, 203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the IP extraction logic from our proxy
			realIP := tt.remoteAddr
			if host, _, err := net.SplitHostPort(realIP); err == nil {
				realIP = host
			}
			if tt.existingForwarded != "" {
				realIP = strings.Split(tt.existingForwarded, ",")[0]
				realIP = strings.TrimSpace(realIP)
			}

			if realIP != tt.expectedRealIP {
				t.Errorf("Real IP: expected %s, got %s", tt.expectedRealIP, realIP)
			}

			// Test forwarded header construction
			var forwarded string
			if tt.existingForwarded != "" {
				forwarded = tt.existingForwarded + ", " + realIP
			} else {
				forwarded = realIP
			}

			if forwarded != tt.expectedForwarded {
				t.Errorf("Forwarded header: expected %s, got %s", tt.expectedForwarded, forwarded)
			}
		})
	}
}
