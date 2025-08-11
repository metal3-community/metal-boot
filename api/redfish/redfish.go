package redfish

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

//go:generate go tool oapi-codegen -package redfish -o server.gen.go -generate std-http-server,models openapi.yaml
func (server *RedfishServer) ListenAndServe(
	ctx context.Context,
	handlers map[string]http.HandlerFunc,
) error {
	m := http.NewServeMux()

	options := StdHTTPServerOptions{
		BaseURL:    server.Config.Address,
		BaseRouter: m,
	}

	if options.ErrorHandlerFunc == nil {
		options.ErrorHandlerFunc = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}

	for path, handler := range handlers {
		m.HandleFunc(path, handler)
	}

	s := &http.Server{
		Handler: HandlerWithOptions(server, options),

		Addr: fmt.Sprintf("%s:%d", server.Config.Address, server.Config.Port),
	}

	go func() {
		<-ctx.Done()
		server.Log.Info("shutting down http server")
		_ = s.Shutdown(ctx)
	}()
	if err := s.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		server.Log.Error(err, "listen and serve http")
		return err
	}

	return nil
}

func (server *RedfishServer) Handler(w http.ResponseWriter, r *http.Request) {
	HandlerWithOptions(server, StdHTTPServerOptions{}).ServeHTTP(w, r)
}

func (server *RedfishServer) Register(router *http.ServeMux) {
	HandlerWithOptions(server, StdHTTPServerOptions{
		BaseRouter: router,
	})
}
