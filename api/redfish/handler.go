package redfish

import (
	"log/slog"
	"net/http"

	"github.com/metal3-community/metal-boot/internal/backend"
	"github.com/metal3-community/metal-boot/internal/config"
)

//go:generate go tool oapi-codegen -package redfish -o server.gen.go -generate std-http-server,models openapi.yaml
func New(
	logger *slog.Logger,
	cfg *config.Config,
	reader backend.BackendReader,
	pwrBackend backend.BackendPower,
) http.Handler {
	mux := http.NewServeMux()

	server := &RedfishServer{
		Config:       cfg,
		Log:          cfg.Log.WithName("redfish-server"),
		reader:       reader,
		firmwarePath: cfg.FirmwarePath,
		power:        pwrBackend,
	}

	options := StdHTTPServerOptions{
		BaseURL:    "",
		BaseRouter: mux,
	}

	if options.ErrorHandlerFunc == nil {
		options.ErrorHandlerFunc = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}

	server.Log.Info("starting redfish server",
		"address", cfg.Address,
		"port", cfg.Port,
		"firmware", cfg.FirmwarePath)

	// server.refreshSystems(context.Background())

	return HandlerWithOptions(server, options)
}
