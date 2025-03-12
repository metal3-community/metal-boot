package redfish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/bmcpi/pibmc/internal/config"
	"github.com/bmcpi/pibmc/internal/dhcp/data"
	"github.com/bmcpi/pibmc/internal/dhcp/handler"
	"github.com/bmcpi/pibmc/internal/firmware/varstore"
	"github.com/bmcpi/pibmc/internal/util"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
)

func (s *PowerState) GetPoeMode() string {
	if s == nil {
		return ""
	}
	switch *s {
	case On:
		return "auto"
	case Off:
		return "off"
	case PoweringOff:
		return "off"
	case PoweringOn:
		return "auto"
	default:
		return "off"
	}
}

const tracerName = "github.com/bmcpi/pibmc/api/redfish"

type RedfishServerConfig struct {
	Insecure      bool
	UnifiUser     string
	UnifiPass     string
	UnifiEndpoint string
	UnifiSite     string
	UnifiDevice   string
	Logger        logr.Logger
	TftpRoot      string
}

type RedfishSystem struct {
	MacAddress       string `yaml:"mac"`
	IpAddress        string `yaml:"ip"`
	UnifiPort        int    `yaml:"port"`
	SiteID           string `yaml:"site"`
	DeviceMac        string `yaml:"device_mac"`
	PoeMode          string `yaml:"poe_mode"`
	EfiVariableStore *varstore.EfiVariableStore
}

func (r *RedfishSystem) GetPowerState() *PowerState {
	state := Off
	switch r.PoeMode {
	case "auto":
		state = On
	case "off":
		state = Off
	default:
		state = Off
	}
	return &state
}

func redfishError(err error) *RedfishError {
	return &RedfishError{
		Error: RedfishErrorError{
			Message: util.Ptr(err.Error()),
			Code:    util.Ptr("Base.1.0.GeneralError"),
		},
	}
}

type RedfishServer struct {
	Config *config.Config

	Log logr.Logger

	backend handler.BackendStore
}

func NewRedfishServer(cfg *config.Config, backend handler.BackendStore) *RedfishServer {
	server := &RedfishServer{
		Config:  cfg,
		Log:     cfg.Log.WithName("redfish-server"),
		backend: backend,
	}

	server.Log.Info("starting redfish server", "address", cfg.Address, "port", cfg.Port)

	server.refreshSystems(context.Background())

	return server
}

func (s *RedfishServer) refreshSystems(ctx context.Context) (err error) {

	s.Log.Info("refreshing systems", "backend", s.backend)

	s.backend.Sync(ctx)

	return
}

// CreateVirtualDisk implements ServerInterface.
func (s *RedfishServer) CreateVirtualDisk(w http.ResponseWriter, r *http.Request, systemId string, storageControllerId string) {

	req := CreateVirtualDiskRequestBody{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusNotFound)
		s.Log.Error(err, "error decoding request")
		return
	}

	panic("unimplemented")
}

// DeleteVirtualdisk implements ServerInterface.
func (s *RedfishServer) DeleteVirtualdisk(w http.ResponseWriter, r *http.Request, systemId string, storageId string) {
	panic("unimplemented")
}

// EjectVirtualMedia implements ServerInterface.
func (s *RedfishServer) EjectVirtualMedia(w http.ResponseWriter, r *http.Request, managerId string, virtualMediaId string) {
	panic("unimplemented")
}

// FirmwareInventory implements ServerInterface.
func (s *RedfishServer) FirmwareInventory(w http.ResponseWriter, r *http.Request) {

	panic("unimplemented")
}

// FirmwareInventoryDownloadImage implements ServerInterface.
func (s *RedfishServer) FirmwareInventoryDownloadImage(w http.ResponseWriter, r *http.Request) {
	panic("unimplemented")
}

// GetManager implements ServerInterface.
func (s *RedfishServer) GetManager(w http.ResponseWriter, r *http.Request, managerId string) {

	manager := Manager{
		Id:        &managerId,
		OdataId:   util.Ptr(fmt.Sprintf("/redfish/v1/Managers/%s", managerId)),
		OdataType: util.Ptr("#Manager.v1_11_0.Manager"),
		Name:      util.Ptr("Manager"),
		Status: &Status{
			State: util.Ptr(StateEnabled),
		},
	}
	if err := json.NewEncoder(w).Encode(manager); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error encoding response")
	}
}

// GetManagerVirtualMedia implements ServerInterface.
func (s *RedfishServer) GetManagerVirtualMedia(w http.ResponseWriter, r *http.Request, managerId string, virtualMediaId string) {
	panic("unimplemented")
}

// GetRoot implements ServerInterface.
func (s *RedfishServer) GetRoot(w http.ResponseWriter, r *http.Request) {

	root := Root{
		OdataId:        util.Ptr("/redfish/v1"),
		OdataType:      util.Ptr("#ServiceRoot.v1_11_0.ServiceRoot"),
		Id:             util.Ptr("RootService"),
		Name:           util.Ptr("Root Service"),
		RedfishVersion: util.Ptr("1.11.0"),
		Systems: &IdRef{
			OdataId: util.Ptr("/redfish/v1/Systems"),
		},
	}

	err := json.NewEncoder(w).Encode(root)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error encoding response")
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// GetSoftwareInventory implements ServerInterface.
func (s *RedfishServer) GetSoftwareInventory(w http.ResponseWriter, r *http.Request, softwareId string) {
	panic("unimplemented")
}

// GetSystem implements ServerInterface.
func (s *RedfishServer) GetSystem(w http.ResponseWriter, r *http.Request, systemId string) {

	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.GetSystem")
	defer span.End()

	err := s.refreshSystems(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error refreshing systems", "system", systemId)
		return
	}

	s.Log.Info("getting system", "system", systemId)

	systemIdAddr, err := net.ParseMAC(systemId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error parsing system id", "system", systemId)
		return
	}

	dhcp, _, pwr, err := s.backend.GetByMac(ctx, systemIdAddr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error getting system by mac", "system", systemId)
		return
	}

	defaultName := fmt.Sprintf("System %s", systemId)

	pwrState := Off
	switch pwr.State {
	case "auto":
		pwrState = On
	case "off":
		pwrState = Off
	default:
		pwrState = Off
	}

	if dhcp != nil {
		if dhcp.Hostname != "" {
			defaultName = dhcp.Hostname
		}
	}
	resp := ComputerSystem{
		Id:         &systemId,
		PowerState: (*PowerState)(&pwrState),
		Links: &SystemLinks{
			Chassis:   &[]IdRef{{OdataId: util.Ptr("/redfish/v1/Chassis/1")}},
			ManagedBy: &[]IdRef{{OdataId: util.Ptr("/redfish/v1/Managers/1")}},
		},
		Boot: &Boot{
			BootSourceOverrideEnabled: util.Ptr(BootSourceOverrideEnabledContinuous),
			BootSourceOverrideTarget:  util.Ptr(None),
			BootSourceOverrideTargetRedfishAllowableValues: &[]BootSource{
				Pxe,
				Hdd,
				None,
			},
		},
		Actions: &ComputerSystemActions{
			HashComputerSystemReset: &ComputerSystemReset{
				ResetTypeRedfishAllowableValues: &[]ResetType{
					ResetTypeOn,
					ResetTypeForceOn,
					ResetTypeForceOff,
					ResetTypePowerCycle,
				},
				Target: util.Ptr(fmt.Sprintf("/redfish/v1/Systems/%s/Actions/ComputerSystem.Reset", systemId)),
			},
		},
		OdataId:   util.Ptr(fmt.Sprintf("/redfish/v1/Systems/%s", systemId)),
		OdataType: util.Ptr("#ComputerSystem.v1_11_0.ComputerSystem"),
		Name:      util.Ptr(defaultName),
		Status: &Status{
			State: util.Ptr(StateEnabled),
		},
		UUID: util.Ptr(systemIdAddr.String()),
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error marshalling response", "system", systemId)
		return
	}
}

// GetTask implements ServerInterface.
func (s *RedfishServer) GetTask(w http.ResponseWriter, r *http.Request, taskId string) {
	panic("unimplemented")
}

// GetTaskList implements ServerInterface.
func (s *RedfishServer) GetTaskList(w http.ResponseWriter, r *http.Request) {
	panic("unimplemented")
}

// GetVolumes implements ServerInterface.
func (s *RedfishServer) GetVolumes(w http.ResponseWriter, r *http.Request, systemId string, storageControllerId string) {
	panic("unimplemented")
}

// InsertVirtualMedia implements ServerInterface.
func (s *RedfishServer) InsertVirtualMedia(w http.ResponseWriter, r *http.Request, managerId string, virtualMediaId string) {

	req := InsertMediaRequestBody{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error decoding request", "manager", managerId, "virtualMedia", virtualMediaId)
		return
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// ListManagerVirtualMedia implements ServerInterface.
func (s *RedfishServer) ListManagerVirtualMedia(w http.ResponseWriter, r *http.Request, managerId string) {
	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.ListManagerVirtualMedia")
	defer span.End()

	ids := make([]IdRef, 0)

	odataId := "/redfish/v1/Managers/1/VirtualMedia/1"
	ids = append(ids, IdRef{
		OdataId: &odataId,
	})

	response := Collection{
		Members:           &ids,
		OdataContext:      util.Ptr("/redfish/v1/$metadata#VirtualMediaCollection.VirtualMediaCollection"),
		OdataType:         "#VirtualMediaCollection.VirtualMediaCollection",
		Name:              util.Ptr("Virtual Media Collection"),
		OdataId:           "/redfish/v1/Managers/1/VirtualMedia",
		MembersOdataCount: util.Ptr(len(ids)),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error encoding response", "manager", managerId)
	}
}

// ListManagers implements ServerInterface.
func (s *RedfishServer) ListManagers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.ListManagers")
	defer span.End()

	s.Log.Info("listing managers", "url", r.URL)

	ids := make([]IdRef, 0)

	odataId := "/redfish/v1/Managers/1"
	ids = append(ids, IdRef{
		OdataId: &odataId,
	})

	response := Collection{
		Members:           &ids,
		OdataContext:      util.Ptr("/redfish/v1/$metadata#ManagerCollection.ManagerCollection"),
		OdataType:         "#ManagerCollection.ManagerCollection",
		Name:              util.Ptr("Manager Collection"),
		OdataId:           "/redfish/v1/Managers",
		MembersOdataCount: util.Ptr(len(ids)),
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(fmt.Appendf(nil, "error encoding response: %s", err))
	}
}

// ListSystems implements ServerInterface.
func (s *RedfishServer) ListSystems(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.ListSystems")
	defer span.End()

	s.Log.Info("listing systems", "url", r.URL)

	ids := make([]IdRef, 0)

	keys, err := s.backend.GetKeys(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error getting keys")
		return
	}

	for _, m := range keys {
		odataId := fmt.Sprintf("/redfish/v1/Systems/%s", m)
		ids = append(ids, IdRef{
			OdataId: &odataId,
		})
	}

	response := Collection{
		Members:           &ids,
		OdataContext:      util.Ptr("/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"),
		OdataType:         "#ComputerSystemCollection.ComputerSystemCollection",
		Name:              util.Ptr("Computer System Collection"),
		OdataId:           "/redfish/v1/Systems",
		MembersOdataCount: util.Ptr(len(ids)),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error encoding response")
	}
}

// ResetIdrac implements ServerInterface.
func (s *RedfishServer) ResetIdrac(w http.ResponseWriter, r *http.Request) {
	panic("unimplemented")
}

// ResetSystem implements ServerInterface.
func (s *RedfishServer) ResetSystem(w http.ResponseWriter, r *http.Request, systemId string) {

	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.ListManagerVirtualMedia")
	defer span.End()

	req := ResetSystemJSONRequestBody{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error decoding request")
		return
	}

	s.Log.Info("resetting system", "system", systemId, "resetType", req.ResetType)

	systemIdAddr, err := net.ParseMAC(systemId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error parsing system id")
		return
	}

	_, _, pwr, err := s.backend.GetByMac(ctx, systemIdAddr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error getting system by mac")
		return
	}

	if pwr == nil {
		w.WriteHeader(http.StatusNotFound)
		s.Log.Error(errors.New("power not found"), "system not found", "system", systemId)
		return
	}

	resetType := ResetTypePowerCycle
	if req.ResetType != nil {
		resetType = *req.ResetType
	}

	if resetType == ResetTypePowerCycle {
		err := s.backend.PowerCycle(ctx, systemIdAddr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.Log.Error(err, "error power cycling system", "system", systemId)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	state := "off"
	if pwr.State == "off" {
		state = "auto"
	}

	if resetType == ResetTypeForceOff {
		state = "off"
	}

	err = s.backend.Put(ctx, systemIdAddr, nil, nil, &data.Power{
		State:    state,
		Port:     pwr.Port,
		DeviceId: pwr.DeviceId,
		SiteId:   pwr.SiteId,
	})
	if err != nil {
		s.Log.Error(err, "error setting power state", "system", systemId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if state == "off" && resetType != ResetTypeForceOff {
		defer func() {
			time.Sleep(time.Duration(s.Config.ResetDelaySec) * time.Second)
			err = s.backend.Put(ctx, systemIdAddr, nil, nil, &data.Power{
				State:    "auto",
				Port:     pwr.Port,
				DeviceId: pwr.DeviceId,
				SiteId:   pwr.SiteId,
			})
			if err != nil {
				s.Log.Error(err, "error setting power state", "system", systemId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}()
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetSystem implements ServerInterface.
func (s *RedfishServer) SetSystem(w http.ResponseWriter, r *http.Request, systemId string) {

	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.SetSystem")
	defer span.End()

	req := SetSystemJSONRequestBody{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error decoding request")
		return
	}

	s.Log.Info("setting system", "system", systemId, "systemInfo", req)

	systemIdAddr, err := net.ParseMAC(systemId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error parsing system id")
		return
	}

	_, _, pwr, err := s.backend.GetByMac(ctx, systemIdAddr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error getting system by mac")
		return
	}

	powerState := On
	if req.PowerState != nil {
		powerState = *req.PowerState
	}

	poeState := "auto"
	if powerState == Off || powerState == PoweringOff {
		poeState = "off"
	}

	if pwr.State != poeState {
		err := s.backend.Put(ctx, systemIdAddr, nil, nil, &data.Power{
			State:    poeState,
			Port:     pwr.Port,
			DeviceId: pwr.DeviceId,
			SiteId:   pwr.SiteId,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.Log.Error(err, "error setting power state", "system", systemId)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
	s.Log.Info("system updated", "system", systemId)
}

// UpdateService implements ServerInterface.
func (s *RedfishServer) UpdateService(w http.ResponseWriter, r *http.Request) {
	panic("unimplemented")
}

// UpdateServiceSimpleUpdate implements ServerInterface.
func (s *RedfishServer) UpdateServiceSimpleUpdate(w http.ResponseWriter, r *http.Request) {
	panic("unimplemented")
}
