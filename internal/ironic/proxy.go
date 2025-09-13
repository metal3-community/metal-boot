package ironic

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// SocketProxy creates a reverse proxy for the Ironic Unix socket.
type SocketProxy struct {
	proxy      *httputil.ReverseProxy
	dialer     *net.Dialer
	logger     *slog.Logger
	socketPath string
}

// NewSocketProxy creates a new reverse proxy for a Unix socket.
func NewSocketProxy(logger *slog.Logger, socketPath string) *SocketProxy {
	dialer := &net.Dialer{LocalAddr: nil}

	// Create a custom transport for Unix socket
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			raddr, err := net.ResolveUnixAddr("unix", socketPath)
			if err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, raddr.Network(), raddr.String())
		},
	}

	// Create reverse proxy
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			// Set target URL (the actual URL doesn't matter for Unix socket)
			r.SetURL(&url.URL{
				Scheme: "http",
				Host:   "unix",
			})

			// Remove the /v1 prefix from the path since Ironic expects it
			if strings.HasPrefix(r.In.URL.Path, "/v1") {
				r.Out.URL.Path = r.In.URL.Path
			}

			// Set proxy headers as recommended in Ironic NGINX config
			// These headers are crucial for Ironic to properly handle requests
			r.Out.Header.Set("Host", r.In.Host)
			
			// Get real IP address, checking for forwarded headers first
			realIP := r.In.RemoteAddr
			if host, _, err := net.SplitHostPort(realIP); err == nil {
				realIP = host
			}
			if forwardedFor := r.In.Header.Get("X-Forwarded-For"); forwardedFor != "" {
				realIP = strings.Split(forwardedFor, ",")[0]
				realIP = strings.TrimSpace(realIP)
			}
			r.Out.Header.Set("X-Real-IP", realIP)
			
			// Handle X-Forwarded-For header (append to existing or create new)
			if existingForwarded := r.In.Header.Get("X-Forwarded-For"); existingForwarded != "" {
				r.Out.Header.Set("X-Forwarded-For", existingForwarded+", "+realIP)
			} else {
				r.Out.Header.Set("X-Forwarded-For", realIP)
			}
			
			// Set forwarded protocol
			scheme := "http"
			if r.In.TLS != nil {
				scheme = "https"
			}
			// Check for existing forwarded proto header
			if forwardedProto := r.In.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
				scheme = forwardedProto
			}
			r.Out.Header.Set("X-Forwarded-Proto", scheme)
		},
		Transport: transport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Info("Proxy error", "method", r.Method, "path", r.URL.Path, "error", err)
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		},
	}

	return &SocketProxy{
		proxy:      proxy,
		logger:     logger,
		dialer:     dialer,
		socketPath: socketPath,
	}
}

// ServeHTTP implements http.Handler.
func (ip *SocketProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip.logger.Info("Proxying request", 
		"method", r.Method, 
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"host", r.Host,
		"x_forwarded_for", r.Header.Get("X-Forwarded-For"),
		"x_real_ip", r.Header.Get("X-Real-IP"),
		"x_forwarded_proto", r.Header.Get("X-Forwarded-Proto"))
	ip.proxy.ServeHTTP(w, r)
}
