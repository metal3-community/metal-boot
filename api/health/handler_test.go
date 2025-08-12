package health

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	gitRev := "test-revision"
	startTime := time.Now()

	handler := New(logger, gitRev, startTime)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
}

func TestHandler_ServeHTTP(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	gitRev := "test-revision"
	startTime := time.Now()

	handler := New(logger, gitRev, startTime)

	req := httptest.NewRequest(http.MethodGet, "/healthcheck", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	if response["git_rev"] != gitRev {
		t.Errorf("Expected git_rev '%s', got %v", gitRev, response["git_rev"])
	}
}
