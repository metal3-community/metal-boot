// Package images implements an HTTP server for iPXE binaries.
package static

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/bmcpi/pibmc/internal/config"
	"github.com/bmcpi/pibmc/internal/util"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func DownloadImages(c *config.Config) error {
	root, err := os.OpenRoot(c.Images.RootDirectory)
	if err != nil {
		c.Log.Error(err, "failed to open root directory")
		return err
	}

	defer root.Close()

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,

			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	for _, image := range c.Images.ImageURLs {

		if util.ExistsInRoot(root, image.Path) {
			c.Log.Info("file already exists", "path", image.Path)
			continue
		}

		out, err := root.Create(image.Path)
		if err != nil {
			return err
		}
		defer out.Close()

		resp, err := httpClient.Get(image.URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Check server response
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("bad status: %s", resp.Status)
		}

		// Writer the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return err
		}
	}

	return nil
}

// Handle handles GET and HEAD responses to HTTP requests.
// Serves embedded iPXE binaries.
func HandlerFunc(c *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		c.Log.V(1).Info("handling request", "method", req.Method, "path", req.URL.Path)
		if req.Method != http.MethodGet && req.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		host, port, _ := net.SplitHostPort(req.RemoteAddr)
		log := c.Log.WithValues("host", host, "port", port)

		filename := filepath.Base(req.URL.Path)
		log = log.WithValues("filename", filename)

		// clients can send traceparent over HTTP by appending the traceparent string
		// to the end of the filename they really want
		longfile := filename // hang onto this to report in traces
		ctx, shortfile, err := extractTraceparentFromFilename(context.Background(), filename)
		if err != nil {
			log.Error(err, "failed to extract traceparent from filename")
		}
		if shortfile != filename {
			log = log.WithValues("shortfile", shortfile)
			log.Info("traceparent found in filename", "filenameWithTraceparent", longfile)
			filename = shortfile
		}

		tracer := otel.Tracer("HTTP")
		_, span := tracer.Start(ctx, fmt.Sprintf("HTTP %v", req.Method),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attribute.String("filename", filename)),
			trace.WithAttributes(attribute.String("requested-filename", longfile)),
			trace.WithAttributes(attribute.String("ip", host)),
		)
		defer span.End()

		path := req.URL.Path
		rootDirectory := c.Images.RootDirectory
		pathParts := filepath.SplitList(path)
		rootUrl := pathParts[0]
		if rootUrl == "tftp" {
			rootDirectory = c.Tftp.RootDirectory
			path = filepath.Join(pathParts[1:]...)
		}

		root, err := os.OpenRoot(rootDirectory)
		if err != nil {
			log.Error(err, "failed to open root directory")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer root.Close()

		if util.ExistsInRoot(root, path) {
			file, err := root.OpenFile(path, os.O_RDONLY, 0o755)
			if err != nil {
				log.Error(err, "file not found", "path", path)
				http.NotFound(w, req)
				return
			}
			defer file.Close()

			http.ServeContent(w, req, filename, time.Now(), file)
			if req.Method == http.MethodGet {
				log.Info("file served", "name", filename)
			} else if req.Method == http.MethodHead {
				log.Info("HEAD method requested")
			}

			span.SetStatus(codes.Ok, filename)
		} else {
			log.Info("file not found", "path", path)
			http.NotFound(w, req)
		}
	}
}

// extractTraceparentFromFilename takes a context and filename and checks the filename for
// a traceparent tacked onto the end of it. If there is a match, the traceparent is extracted
// and a new SpanContext is constructed and added to the context.Context that is returned.
// The filename is shortened to just the original filename so the rest of boots HTTP can
// carry on as usual.
func extractTraceparentFromFilename(ctx context.Context, filename string) (context.Context, string, error) {
	// traceparentRe captures 4 items, the original filename, the trace id, span id, and trace flags
	traceparentRe := regexp.MustCompile("^(.*)-[[:xdigit:]]{2}-([[:xdigit:]]{32})-([[:xdigit:]]{16})-([[:xdigit:]]{2})")
	parts := traceparentRe.FindStringSubmatch(filename)
	if len(parts) == 5 {
		traceID, err := trace.TraceIDFromHex(parts[2])
		if err != nil {
			return ctx, filename, fmt.Errorf("parsing OpenTelemetry trace id %q failed: %w", parts[2], err)
		}

		spanID, err := trace.SpanIDFromHex(parts[3])
		if err != nil {
			return ctx, filename, fmt.Errorf("parsing OpenTelemetry span id %q failed: %w", parts[3], err)
		}

		// create a span context with the parent trace id & span id
		spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			Remote:     true,
			TraceFlags: trace.FlagsSampled, // TODO: use the parts[4] value instead
		})

		// inject it into the context.Context and return it along with the original filename
		return trace.ContextWithSpanContext(ctx, spanCtx), parts[1], nil
	}
	// no traceparent found, return everything as it was
	return ctx, filename, nil
}
