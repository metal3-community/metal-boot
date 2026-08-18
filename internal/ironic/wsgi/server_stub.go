//go:build !ironic_wsgi

package wsgi

import (
	"context"
	"log/slog"
	"net/http"
)

// Server is a placeholder for builds without the ironic_wsgi tag.
type Server struct{}

// New always fails when the binary was built without embedded WSGI support.
func New(_ *slog.Logger, _ Config) (*Server, error) {
	return nil, ErrNotBuilt
}

// Run is unreachable in stub builds; New never returns a Server.
func (s *Server) Run(_ context.Context) error { return ErrNotBuilt }

// Handler is unreachable in stub builds; New never returns a Server.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, ErrNotBuilt.Error(), http.StatusNotImplemented)
	})
}
