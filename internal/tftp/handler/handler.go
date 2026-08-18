// Package handler defines the TFTP read-request handler interface and adapters.
// Ported from github.com/tinkerbell/tinkerbell/pkg/tftp/handler.
package handler

import (
	"errors"
	"io"
)

// ErrNotFound is returned when a requested TFTP file is not found.
var ErrNotFound = errors.New("file not found")

// Handler responds to a TFTP read request.
type Handler interface {
	ServeTFTP(filename string, rf io.ReaderFrom) error
}

// HandlerFunc is an adapter to allow the use of ordinary functions as TFTP handlers.
type HandlerFunc func(filename string, rf io.ReaderFrom) error //nolint:revive // method name should match naming convention for Go's http.HandlerFunc

// ServeTFTP calls f(filename, rf).
func (f HandlerFunc) ServeTFTP(filename string, rf io.ReaderFrom) error {
	return f(filename, rf)
}

// notFound is the shared handler returned by NotFoundHandler.
var notFound Handler = HandlerFunc(func(string, io.ReaderFrom) error {
	return ErrNotFound
})

// NotFoundHandler returns a handler that always returns ErrNotFound.
func NotFoundHandler() Handler {
	return notFound
}
