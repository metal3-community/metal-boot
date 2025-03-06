package redfish

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

func (server *RedfishServer) ListenAndServe(ctx context.Context, handlers map[string]http.HandlerFunc) error {

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

func (server *RedfishServer) GetHandlers() map[string]http.HandlerFunc {
	wrapper := ServerInterfaceWrapper{
		Handler: server,
	}
	return map[string]http.HandlerFunc{
		"GET /redfish/v1":          wrapper.GetRoot,
		"GET /redfish/v1/Managers": wrapper.ListManagers,
		"POST /redfish/v1/Managers/iDRAC.Embedded.1/Actions/Manager.Reset":                                     wrapper.ResetIdrac,
		"GET /redfish/v1/Managers/{managerId}":                                                                 wrapper.GetManager,
		"GET /redfish/v1/Managers/{managerId}/VirtualMedia":                                                    wrapper.ListManagerVirtualMedia,
		"GET /redfish/v1/Managers/{managerId}/VirtualMedia/{virtualMediaId}":                                   wrapper.GetManagerVirtualMedia,
		"POST /redfish/v1/Managers/{managerId}/VirtualMedia/{virtualMediaId}/Actions/VirtualMedia.EjectMedia":  wrapper.EjectVirtualMedia,
		"POST /redfish/v1/Managers/{managerId}/VirtualMedia/{virtualMediaId}/Actions/VirtualMedia.InsertMedia": wrapper.InsertVirtualMedia,
		"GET /redfish/v1/Systems": wrapper.ListSystems,
		"POST /redfish/v1/Systems/{ComputerSystemId}/Actions/ComputerSystem.Reset":          wrapper.ResetSystem,
		"DELETE /redfish/v1/Systems/{ComputerSystemId}/Storage/Volumes/{StorageId}":         wrapper.DeleteVirtualdisk,
		"GET /redfish/v1/Systems/{ComputerSystemId}/Storage/{StorageControllerId}/Volumes":  wrapper.GetVolumes,
		"POST /redfish/v1/Systems/{ComputerSystemId}/Storage/{StorageControllerId}/Volumes": wrapper.CreateVirtualDisk,
		"GET /redfish/v1/Systems/{ComputerSystemId}":                                        wrapper.GetSystem,
		"PATCH /redfish/v1/Systems/{ComputerSystemId}":                                      wrapper.SetSystem,
		"GET /redfish/v1/TaskService/Tasks":                                                 wrapper.GetTaskList,
		"GET /redfish/v1/TaskService/Tasks/{taskId}":                                        wrapper.GetTask,
		"GET /redfish/v1/UpdateService":                                                     wrapper.UpdateService,
		"POST /redfish/v1/UpdateService/Actions/UpdateService.SimpleUpdate":                 wrapper.UpdateServiceSimpleUpdate,
		"GET /redfish/v1/UpdateService/FirmwareInventory":                                   wrapper.FirmwareInventory,
		"POST /redfish/v1/UpdateService/FirmwareInventory":                                  wrapper.FirmwareInventoryDownloadImage,
		"GET /redfish/v1/UpdateService/FirmwareInventory/{softwareId}":                      wrapper.GetSoftwareInventory,
	}
}
