//go:build ironic_wsgi

package wsgi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// startTestServer boots one embedded interpreter with a small pure-Python
// WSGI app. CPython can only be initialized once per process, so a single
// server instance is shared by every test in this file.
var (
	testServerOnce sync.Once
	testServer     *Server
	testHTTP       *httptest.Server
	testCancel     context.CancelFunc
)

const testApp = `
import json

def application(environ, start_response):
    path = environ.get("PATH_INFO", "")
    if path == "/echo":
        body = environ["wsgi.input"].read(int(environ.get("CONTENT_LENGTH") or 0))
        start_response("200 OK", [("Content-Type", "application/octet-stream")])
        return [body]
    if path == "/headers":
        start_response("201 Created", [
            ("Content-Type", "text/plain"),
            ("X-Custom", "one"),
            ("X-Custom", "two"),
        ])
        return [b"created"]
    if path == "/env":
        keys = {k: str(v) for k, v in environ.items() if k.startswith(("REQUEST", "PATH", "QUERY", "HTTP_X"))}
        start_response("200 OK", [("Content-Type", "application/json")])
        return [json.dumps(keys).encode()]
    if path == "/chunks":
        start_response("200 OK", [("Content-Type", "text/plain")])
        return (c.encode() for c in ["alpha", "beta", "gamma"])
    start_response("404 Not Found", [("Content-Type", "text/plain")])
    return [b"not found"]
`

func startTestServer(t *testing.T) {
	t.Helper()
	testServerOnce.Do(func() {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		srv, err := New(logger, Config{
			Bootstrap:      testApp,
			Workers:        2,
			RequestTimeout: 10 * time.Second,
		})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		testCancel = cancel
		testServer = srv
		go func() {
			if err := srv.Run(ctx); err != nil {
				t.Errorf("Run: %v", err)
			}
		}()
		testHTTP = httptest.NewServer(srv.Handler())

		// Wait for the interpreter to come up.
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			resp, err := http.Get(testHTTP.URL + "/headers")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode != http.StatusServiceUnavailable {
					return
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatal("embedded WSGI app did not become ready")
	})
	if testServer == nil {
		t.Fatal("test server failed to start")
	}
}

func TestEmbeddedEcho(t *testing.T) {
	startTestServer(t)

	body := strings.Repeat("payload-", 1024)
	resp, err := http.Post(testHTTP.URL+"/echo", "application/octet-stream",
		strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if string(got) != body {
		t.Fatalf("echo mismatch: got %d bytes, want %d", len(got), len(body))
	}
}

func TestEmbeddedHeaders(t *testing.T) {
	startTestServer(t)

	resp, err := http.Get(testHTTP.URL + "/headers")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, want 201", resp.StatusCode)
	}
	if got := resp.Header.Values("X-Custom"); len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("X-Custom = %v", got)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "created" {
		t.Fatalf("body = %q", body)
	}
}

func TestEmbeddedEnviron(t *testing.T) {
	startTestServer(t)

	req, _ := http.NewRequest(http.MethodGet, testHTTP.URL+"/env?a=1&b=2", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	for _, want := range []string{
		`"PATH_INFO": "/env"`,
		`"QUERY_STRING": "a=1&b=2"`,
		`"REQUEST_METHOD": "GET"`,
		`"HTTP_X_FORWARDED_PROTO": "https"`,
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("environ missing %s in %s", want, body)
		}
	}
}

func TestEmbeddedGeneratorResponse(t *testing.T) {
	startTestServer(t)

	resp, err := http.Get(testHTTP.URL + "/chunks")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "alphabetagamma" {
		t.Fatalf("body = %q", body)
	}
}

func TestEmbeddedConcurrent(t *testing.T) {
	startTestServer(t)

	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Post(testHTTP.URL+"/echo", "text/plain",
				strings.NewReader("concurrent"))
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if string(body) != "concurrent" {
				errs <- io.ErrUnexpectedEOF
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
