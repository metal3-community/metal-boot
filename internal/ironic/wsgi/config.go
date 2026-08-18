// Package wsgi hosts a Python WSGI application (by default the OpenStack
// Ironic API) inside an embedded CPython interpreter, bridging Go's net/http
// directly to the WSGI callable without an intermediate process or socket.
//
// The design follows github.com/jonny5532/wsgo: a single interpreter is
// initialized on a locked OS thread, a fixed pool of worker goroutines (each
// pinned to its own OS thread with a persistent PyThreadState) executes WSGI
// calls, and the GIL is released around all request-body reads and
// response-body writes so Go I/O never blocks Python execution.
//
// The cgo implementation is only compiled when the "ironic_wsgi" build tag is
// set (it requires the python3-embed pkg-config package). Without the tag,
// New returns ErrNotBuilt and callers should fall back to supervising an
// external ironic process.
package wsgi

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrNotBuilt is returned by New when the binary was compiled without the
// ironic_wsgi build tag.
var ErrNotBuilt = errors.New(
	"embedded WSGI support not built (rebuild with -tags ironic_wsgi)",
)

// Config controls the embedded WSGI server.
type Config struct {
	// ConfigFile is passed to the application via sys.argv as
	// "--config-file". For Ironic this is the ironic.conf path.
	ConfigFile string

	// Module is the Python module that provides the WSGI application. The
	// module must expose either a module-level "application" callable or an
	// "initialize_wsgi_app(argv)" factory (as ironic.api.wsgi does).
	// Defaults to "ironic.api.wsgi".
	Module string

	// Bootstrap optionally replaces the generated bootstrap code entirely.
	// It is executed in __main__ and must define a module-level
	// "application" callable.
	Bootstrap string

	// PythonPath entries are prepended to sys.path before the application
	// module is imported.
	PythonPath []string

	// Workers is the number of Python worker threads. Because of the GIL
	// these mostly interleave, but they allow requests to overlap during
	// blocking I/O inside the application. Defaults to 4.
	Workers int

	// QueueSize is the number of requests that may be queued waiting for a
	// worker before new requests are rejected with 503. Defaults to 64.
	QueueSize int

	// RequestTimeout bounds a single WSGI call. On expiry a TimeoutError is
	// raised asynchronously inside the stuck Python thread and the client
	// receives a 504. Defaults to 60s.
	RequestTimeout time.Duration

	// RequestBufferLen is how much of a POST/PUT/PATCH body to read into
	// memory before occupying a Python worker. Defaults to 1 MiB.
	RequestBufferLen int

	// ServerName and ServerPort populate the WSGI SERVER_NAME/SERVER_PORT
	// environ keys.
	ServerName string
	ServerPort string
}

func (c Config) withDefaults() Config {
	if c.Module == "" {
		c.Module = "ironic.api.wsgi"
	}
	if c.Workers <= 0 {
		c.Workers = 4
	}
	if c.QueueSize <= 0 {
		c.QueueSize = 64
	}
	if c.RequestTimeout <= 0 {
		c.RequestTimeout = 60 * time.Second
	}
	if c.RequestBufferLen <= 0 {
		c.RequestBufferLen = 1 << 20
	}
	if c.ServerName == "" {
		c.ServerName = "metal-boot"
	}
	if c.ServerPort == "" {
		c.ServerPort = "80"
	}
	return c
}

// pyLiteral renders a Go string as a Python string literal. JSON string
// escaping is a valid subset of Python string literal syntax.
func pyLiteral(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// bootstrapCode builds the Python code executed in __main__ at interpreter
// startup. It must leave a module-level "application" WSGI callable behind.
func (c Config) bootstrapCode() string {
	if c.Bootstrap != "" {
		return c.Bootstrap
	}

	var sb strings.Builder
	sb.WriteString("import sys\n")
	sb.WriteString("import faulthandler\n")
	sb.WriteString("faulthandler.enable()\n")

	argv := []string{"ironic-api"}
	if c.ConfigFile != "" {
		argv = append(argv, "--config-file", c.ConfigFile)
	}
	elems := make([]string, len(argv))
	for i, a := range argv {
		elems[i] = pyLiteral(a)
	}
	fmt.Fprintf(&sb, "sys.argv = [%s]\n", strings.Join(elems, ", "))

	for i := len(c.PythonPath) - 1; i >= 0; i-- {
		fmt.Fprintf(&sb, "sys.path.insert(0, %s)\n", pyLiteral(c.PythonPath[i]))
	}

	fmt.Fprintf(&sb, `import importlib
_metal_boot_wsgi_module = importlib.import_module(%s)
application = getattr(_metal_boot_wsgi_module, "application", None)
if application is None:
    _metal_boot_wsgi_init = getattr(_metal_boot_wsgi_module, "initialize_wsgi_app", None)
    if _metal_boot_wsgi_init is None:
        raise RuntimeError(
            "module %s provides neither 'application' nor 'initialize_wsgi_app'")
    application = _metal_boot_wsgi_init(sys.argv)
`, pyLiteral(c.Module), pyLiteral(c.Module))

	return sb.String()
}
