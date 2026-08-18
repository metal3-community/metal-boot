package server

import (
	"io"
	"regexp"
	"sync"

	"github.com/go-logr/logr"
	"github.com/metal3-community/metal-boot/internal/tftp/handler"
)

// Route represents a TFTP route with its regex pattern, description, and handler.
type Route struct {
	Pattern     string          `json:"pattern"`
	Description string          `json:"description"`
	Handler     handler.Handler `json:"-"`
}

// Routes is a collection of Route objects that can be registered with a TFTP server.
type Routes []Route

// Register adds a new route to the Routes collection for later registration with a ServeMux.
func (rs *Routes) Register(pattern string, h handler.Handler, desc string) {
	if desc == "" {
		desc = "No description provided"
	}
	*rs = append(*rs, Route{
		Pattern:     pattern,
		Description: desc,
		Handler:     h,
	})
}

// Mux builds a ServeMux from the registered routes.
func (rs *Routes) Mux(log logr.Logger) *ServeMux {
	mux := NewServeMux()
	mux.SetLogger(log)
	for _, route := range *rs {
		mux.Handle(route.Pattern, route.Handler)
	}
	return mux
}

// patternHandler holds a compiled regex pattern and its associated handler.
type patternHandler struct {
	pattern *regexp.Regexp
	handler handler.Handler
}

// ServeMux is a TFTP request multiplexer that matches filenames against
// registered regex patterns and routes them to the appropriate handler.
type ServeMux struct {
	mu             sync.RWMutex
	patterns       []patternHandler
	defaultHandler handler.Handler
	log            logr.Logger
}

// NewServeMux allocates and returns a new ServeMux.
func NewServeMux() *ServeMux {
	return &ServeMux{log: logr.Discard()}
}

// SetLogger sets the logger for the ServeMux.
func (mux *ServeMux) SetLogger(log logr.Logger) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.log = log
}

// Handle registers the handler for the given regex pattern.
// If the pattern is malformed or the handler is nil, Handle panics.
func (mux *ServeMux) Handle(pattern string, h handler.Handler) {
	if h == nil {
		panic("tftp: nil handler for pattern " + pattern)
	}
	regex, err := regexp.Compile(pattern)
	if err != nil {
		panic("tftp: invalid pattern " + pattern + ": " + err.Error())
	}

	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.patterns = append(mux.patterns, patternHandler{
		pattern: regex,
		handler: h,
	})
}

// HandleFunc registers the handler function for the given regex pattern.
func (mux *ServeMux) HandleFunc(pattern string, h func(filename string, rf io.ReaderFrom) error) {
	mux.Handle(pattern, handler.HandlerFunc(h))
}

// SetDefaultHandler sets the handler used when no pattern matches.
func (mux *ServeMux) SetDefaultHandler(h handler.Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.defaultHandler = h
}

// match returns the handler for the first pattern matching filename, the
// source of the pattern that matched (empty when falling back to the default
// handler), and a snapshot of the logger. The handler is nil when no pattern
// matches and no default handler is set. Everything is read under a single
// read-lock so concurrent Handle/SetDefaultHandler/SetLogger calls are safe.
func (mux *ServeMux) match(filename string) (handler.Handler, string, logr.Logger) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	for i := range mux.patterns {
		if ph := &mux.patterns[i]; ph.pattern.MatchString(filename) {
			return ph.handler, ph.pattern.String(), mux.log
		}
	}
	return mux.defaultHandler, "", mux.log
}

// ServeTFTP dispatches the request to the handler whose pattern matches
// the request filename. If no pattern matches and a default handler is set,
// it is used. Otherwise ErrNotFound is returned.
func (mux *ServeMux) ServeTFTP(filename string, rf io.ReaderFrom) error {
	h, pattern, log := mux.match(filename)
	switch {
	case h == nil:
		log.Info("no tftp handler found for filename", "filename", filename)
		return handler.ErrNotFound
	case pattern == "":
		log.V(2).Info("using default tftp handler for filename", "filename", filename)
	default:
		log.V(2).Info("tftp request matched pattern", "filename", filename, "pattern", pattern)
	}
	return h.ServeTFTP(filename, rf)
}
