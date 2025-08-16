package binary

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/metal3-community/metal-boot/internal/config"
	"github.com/tinkerbell/ipxedust/binary"
)

// binaryHandler handles requests for iPXE binary files.
type binaryHandler struct {
	logger *slog.Logger
	config *config.Config
}

// New creates a new iPXE binary handler.
func New(logger *slog.Logger, cfg *config.Config) http.Handler {
	return &binaryHandler{
		logger: logger,
		config: cfg,
	}
}

func Match(filename string) bool {
	// Check if the requested filename matches any known iPXE binary files.
	_, found := binary.Files[filename]
	return found
}

// ServeHTTP handles GET and HEAD requests for iPXE binaries.
func (h *binaryHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	reqLogger := h.logger.With("method", req.Method, "path", req.URL.Path)
	reqLogger.Debug("Handling iPXE binary request")

	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		reqLogger.Warn("Method not allowed", "method", req.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	host, port, _ := net.SplitHostPort(req.RemoteAddr)
	reqLogger = reqLogger.With("host", host, "port", port)

	// If a mac address is provided (/0a:00:27:00:00:02/snp.efi), parse and log it.
	// Mac address is optional.
	optionalMac, _ := net.ParseMAC(strings.TrimPrefix(path.Dir(req.URL.Path), "/"))
	reqLogger = reqLogger.With("mac_from_uri", optionalMac.String())

	filename := filepath.Base(req.URL.Path)
	reqLogger = reqLogger.With("filename", filename)

	// clients can send traceparent over HTTP by appending the traceparent string
	// to the end of the filename they really want
	longfile := filename // hang onto this for logging
	ctx, shortfile, err := extractTraceparentFromFilename(req.Context(), filename)
	if err != nil {
		reqLogger.Warn("Failed to extract traceparent from filename", "error", err)
	}
	if shortfile != filename {
		reqLogger = reqLogger.With("short_file", shortfile)
		reqLogger.Info("Traceparent found in filename", "filename_with_traceparent", longfile)
		filename = shortfile
	}

	// Update context for downstream processing
	req = req.WithContext(ctx)
	reqLogger = reqLogger.With("final_filename", filename)

	file, found := binary.Files[filename]
	if !found {
		reqLogger.Info("Requested file not found")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Apply iPXE patch if configured
	if h.config.Tftp.IpxePatch != "" {
		file, err = binary.Patch(file, []byte(h.config.Tftp.IpxePatch))
		if err != nil {
			reqLogger.Error("Error patching file", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	http.ServeContent(w, req, filename, time.Now(), bytes.NewReader(file))

	switch req.Method {
	case http.MethodGet:
		reqLogger.Info("File served", "file_size", len(file))
	case http.MethodHead:
		reqLogger.Info("HEAD method requested", "file_size", len(file))
	}
}

// extractTraceparentFromFilename takes a context and filename and checks the filename for
// a traceparent tacked onto the end of it. If there is a match, the traceparent is extracted
// and used to create tracing context (though we're not using OpenTelemetry anymore, we keep
// this for compatibility). The filename is shortened to just the original filename.
func extractTraceparentFromFilename(
	ctx context.Context,
	filename string,
) (context.Context, string, error) {
	// traceparentRe captures 4 items, the original filename, the trace id, span id, and trace flags
	traceparentRe := regexp.MustCompile(
		"^(.*)-[[:xdigit:]]{2}-([[:xdigit:]]{32})-([[:xdigit:]]{16})-([[:xdigit:]]{2})",
	)
	parts := traceparentRe.FindStringSubmatch(filename)
	if len(parts) == 5 {
		// For now, we just return the shortened filename
		// Later we could add tracing context if needed
		return ctx, parts[1], nil
	}
	// no traceparent found, return everything as it was
	return ctx, filename, nil
}
