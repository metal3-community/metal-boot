package redfish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmcpi/pibmc/internal/config"
	"github.com/bmcpi/pibmc/internal/dhcp/data"
	"github.com/bmcpi/pibmc/internal/dhcp/handler"
	"github.com/bmcpi/pibmc/internal/firmware"
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
	FirmwarePath  string
}

type RedfishSystem struct {
	MacAddress       string `yaml:"mac"`
	IpAddress        string `yaml:"ip"`
	UnifiPort        int    `yaml:"port"`
	SiteID           string `yaml:"site"`
	DeviceMac        string `yaml:"device_mac"`
	PoeMode          string `yaml:"poe_mode"`
	EfiVariableStore *varstore.Edk2VarStore
	FirmwareManager  firmware.FirmwareManager
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

func decodeBody[T any](r *http.Request) (*T, error) {
	v := new(T)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return nil, err
	}
	return v, nil
}

type RedfishServer struct {
	Config *config.Config

	Log logr.Logger

	backend handler.BackendStore

	firmwarePath string
}

func NewRedfishServer(cfg *config.Config, backend handler.BackendStore) *RedfishServer {
	server := &RedfishServer{
		Config:       cfg,
		Log:          cfg.Log.WithName("redfish-server"),
		backend:      backend,
		firmwarePath: cfg.FirmwarePath,
	}

	server.Log.Info("starting redfish server",
		"address", cfg.Address,
		"port", cfg.Port,
		"firmware", cfg.FirmwarePath)

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

	if req, err := decodeBody[CreateVirtualDiskRequestBody](r); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error decoding request")
		return
	} else {
		s.Log.Info("creating virtual disk", "system", systemId, "storageController", storageControllerId, "request", req)
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
	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.FirmwareInventory")
	defer span.End()

	s.Log.Info("getting firmware inventory")

	// Check if firmware file exists
	if s.firmwarePath == "" {
		err := errors.New("firmware path not configured")
		s.Log.Error(err, "firmware path not set")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Create firmware inventory response
	firmwareName := filepath.Base(s.firmwarePath)
	inventory := Collection{
		OdataId:   "/redfish/v1/UpdateService/FirmwareInventory",
		OdataType: "#FirmwareInventory.SoftwareInventoryCollection",
		Name:      util.Ptr("Firmware Inventory Collection"),
		Members: &[]IdRef{
			{
				OdataId: util.Ptr(fmt.Sprintf("/redfish/v1/UpdateService/FirmwareInventory/%s", firmwareName)),
			},
		},
		MembersOdataCount: util.Ptr(1),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inventory)
}

// FirmwareInventoryDownloadImage implements ServerInterface.
func (s *RedfishServer) FirmwareInventoryDownloadImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.FirmwareInventoryDownloadImage")
	defer span.End()

	s.Log.Info("downloading firmware image")

	// Check if firmware file exists
	if s.firmwarePath == "" {
		err := errors.New("firmware path not configured")
		s.Log.Error(err, "firmware path not set")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	err := r.ParseMultipartForm(32 << 20) // 32MB limit
	if err != nil {
		s.Log.Error(err, "error parsing multipart form")
		w.WriteHeader(http.StatusBadRequest)
	}

	if fi, ok := r.MultipartForm.File["softwareImage"]; ok && len(fi) > 0 {
		file, err := fi[0].Open()
		if err != nil {
			s.Log.Error(err, "error opening firmware file")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(redfishError(err))
			return
		}
		defer file.Close()

		_, err = os.Stat(s.firmwarePath) // Check if firmware file exists
		if err != nil && !os.IsNotExist(err) {
			s.Log.Error(err, "error checking firmware file", "path", s.firmwarePath)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(redfishError(err))
			return
		}

		targetFile, err := os.Create(s.firmwarePath) // Create or overwrite the firmware file
		if err != nil {
			s.Log.Error(err, "error creating firmware file", "path", s.firmwarePath)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(redfishError(err))
			return
		}

		defer targetFile.Close()
		if _, err := io.Copy(targetFile, file); err != nil {
			s.Log.Error(err, "error copying firmware file", "path", s.firmwarePath)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(redfishError(err))
			return
		}
		// Create a temporary file to store the firmware image
	}
}

// GetManager implements ServerInterface.
func (s *RedfishServer) GetManager(w http.ResponseWriter, r *http.Request, managerId string) {
	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.GetManager")
	defer span.End()

	s.Log.Info("getting manager", "manager", managerId)

	manager := Manager{
		Id:        &managerId,
		OdataId:   util.Ptr(fmt.Sprintf("/redfish/v1/Managers/%s", managerId)),
		OdataType: util.Ptr("#Manager.v1_11_0.Manager"),
		Name:      util.Ptr("Manager"),
		Status: &Status{
			State: util.Ptr(StateEnabled),
		},
		ManagerType:     util.Ptr(ManagerTypeBMC),
		Model:           util.Ptr("Raspberry Pi BMC"),
		FirmwareVersion: util.Ptr("1.0.0"),
		// Add virtual media reference
		VirtualMedia: &IdRef{
			OdataId: util.Ptr(fmt.Sprintf("/redfish/v1/Managers/%s/VirtualMedia", managerId)),
		},
		// Add links to firmware inventory
		Links: &ManagerLinks{
			ManagerForServers: &[]IdRef{
				{
					OdataId: util.Ptr("/redfish/v1/Systems"),
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manager)
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
		Managers: &IdRef{
			OdataId: util.Ptr("/redfish/v1/Managers"),
		},
		UpdateService: &IdRef{
			OdataId: util.Ptr("/redfish/v1/UpdateService"),
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
	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.GetSoftwareInventory")
	defer span.End()

	s.Log.Info("getting software inventory", "id", softwareId)

	// Check if firmware file exists
	if s.firmwarePath == "" {
		err := errors.New("firmware path not configured")
		s.Log.Error(err, "firmware path not set")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Create firmware manager for the system
	firmwareMgr, err := firmware.NewEDK2Manager(s.firmwarePath, s.Log)
	if err != nil {
		s.Log.Error(err, "failed to create firmware manager")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Get firmware version and system info
	version, err := firmwareMgr.GetFirmwareVersion()
	if err != nil {
		s.Log.Error(err, "failed to get firmware version")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	sysInfo, err := firmwareMgr.GetSystemInfo()
	if err != nil {
		s.Log.Info("failed to get system info", "error", err)
		// Continue anyway
	}

	// Create software inventory response
	firmwareName := filepath.Base(s.firmwarePath)

	// If the requested ID doesn't match our firmware name, return 404
	if softwareId != firmwareName {
		err := fmt.Errorf("software inventory %s not found", softwareId)
		s.Log.Error(err, "software inventory not found")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Create description from system info
	description := fmt.Sprintf("UEFI Firmware %s", version)
	for k, v := range sysInfo {
		description += fmt.Sprintf(" - %s: %s", k, v)
	}

	inventory := SoftwareInventory{
		OdataId:     util.Ptr(fmt.Sprintf("/redfish/v1/UpdateService/FirmwareInventory/%s", softwareId)),
		OdataType:   util.Ptr("#SoftwareInventory.v1_5_0.SoftwareInventory"),
		Id:          &softwareId,
		Name:        util.Ptr("UEFI Firmware"),
		Description: util.Ptr(description),
		Version:     util.Ptr(version),
		Status: &Status{
			State:  util.Ptr(StateEnabled),
			Health: util.Ptr(HealthOK),
		},
		Updateable: util.Ptr(true),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inventory)
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
		Bios: &IdRef{
			OdataId: util.Ptr(fmt.Sprintf("/redfish/v1/Systems/%s/BIOS", systemId)),
		},
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Log.Error(err, "error marshalling response", "system", systemId)
		return
	}
}

// Add a new handler for BIOS settings
// func (s *RedfishServer) GetBIOS(w http.ResponseWriter, r *http.Request, systemId string) {
// 	ctx := r.Context()
// 	tracer := otel.Tracer(tracerName)
// 	_, span := tracer.Start(ctx, "redfish.RedfishServer.GetBIOS")
// 	defer span.End()

// 	s.Log.Info("getting BIOS settings", "system", systemId)

// 	// Check if firmware file exists
// 	if s.firmwarePath == "" {
// 		err := errors.New("firmware path not configured")
// 		s.Log.Error(err, "firmware path not set")
// 		w.WriteHeader(http.StatusNotFound)
// 		json.NewEncoder(w).Encode(redfishError(err))
// 		return
// 	}

// 	// Create firmware manager for the system
// 	firmwareMgr, err := firmware.NewEDK2Manager(s.firmwarePath, s.Log)
// 	if err != nil {
// 		s.Log.Error(err, "failed to create firmware manager")
// 		w.WriteHeader(http.StatusInternalServerError)
// 		json.NewEncoder(w).Encode(redfishError(err))
// 		return
// 	}

// 	// Get system info
// 	sysInfo, err := firmwareMgr.GetSystemInfo()
// 	if err != nil {
// 		s.Log.Error(err, "failed to get system info")
// 		w.WriteHeader(http.StatusInternalServerError)
// 		json.NewEncoder(w).Encode(redfishError(err))
// 		return
// 	}

// 	// Get network settings
// 	netSettings, err := firmwareMgr.GetNetworkSettings()
// 	if err != nil {
// 		s.Log.Error(err, "failed to get network settings")
// 		w.WriteHeader(http.StatusInternalServerError)
// 		json.NewEncoder(w).Encode(redfishError(err))
// 		return
// 	}

// 	// Get boot entries
// 	bootEntries, err := firmwareMgr.GetBootEntries()
// 	if err != nil {
// 		s.Log.Error(err, "failed to get boot entries")
// 		w.WriteHeader(http.StatusInternalServerError)
// 		json.NewEncoder(w).Encode(redfishError(err))
// 		return
// 	}

// 	// Create BIOS response
// 	biosSettings := Bios{
// 		OdataId:   util.Ptr(fmt.Sprintf("/redfish/v1/Systems/%s/BIOS", systemId)),
// 		OdataType: util.Ptr("#Bios.v1_2_0.Bios"),
// 		Id:        util.Ptr("BIOS"),
// 		Name:      util.Ptr("UEFI BIOS Settings"),
// 		Attributes: map[string]interface{}{
// 			"SystemInfo":      sysInfo,
// 			"NetworkSettings": netSettings,
// 			"BootEntries":     bootEntries,
// 		},
// 		Actions: &BiosActions{
// 			HashBiosReset: &BiosReset{
// 				Title:  util.Ptr("Reset BIOS Settings"),
// 				Target: util.Ptr(fmt.Sprintf("/redfish/v1/Systems/%s/BIOS/Actions/Bios.Reset", systemId)),
// 			},
// 		},
// 		Links: &BiosLinks{
// 			ActiveSoftwareImage: &IdRef{
// 				OdataId: util.Ptr(fmt.Sprintf("/redfish/v1/UpdateService/FirmwareInventory/%s", filepath.Base(s.firmwarePath))),
// 			},
// 		},
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(biosSettings)
// }

// Handler for BIOS settings reset
func (s *RedfishServer) ResetBIOS(w http.ResponseWriter, r *http.Request, systemId string) {
	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.ResetBIOS")
	defer span.End()

	s.Log.Info("resetting BIOS settings", "system", systemId)

	// Check if firmware file exists
	if s.firmwarePath == "" {
		err := errors.New("firmware path not configured")
		s.Log.Error(err, "firmware path not set")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Create firmware manager for the system
	firmwareMgr, err := firmware.NewEDK2Manager(s.firmwarePath, s.Log)
	if err != nil {
		s.Log.Error(err, "failed to create firmware manager")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Reset to defaults
	err = firmwareMgr.ResetToDefaults()
	if err != nil {
		s.Log.Error(err, "failed to reset BIOS settings")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Save changes
	err = firmwareMgr.SaveChanges()
	if err != nil {
		s.Log.Error(err, "failed to save BIOS settings")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
}

// Handler for updating BIOS settings
func (s *RedfishServer) UpdateBIOS(w http.ResponseWriter, r *http.Request, systemId string) {
	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.UpdateBIOS")
	defer span.End()

	s.Log.Info("updating BIOS settings", "system", systemId)

	// Check if firmware file exists
	if s.firmwarePath == "" {
		err := errors.New("firmware path not configured")
		s.Log.Error(err, "firmware path not set")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.Log.Error(err, "failed to read request body")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Parse request body
	var request BiosUpdateRequest
	err = json.Unmarshal(body, &request)
	if err != nil {
		s.Log.Error(err, "failed to parse request body")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Create firmware manager for the system
	firmwareMgr, err := firmware.NewEDK2Manager(s.firmwarePath, s.Log)
	if err != nil {
		s.Log.Error(err, "failed to create firmware manager")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Apply settings
	if attrs := request.Attributes; attrs != nil {
		// Update network settings if provided
		if netSettings, ok := attrs["NetworkSettings"].(map[string]interface{}); ok {
			ns := firmware.NetworkSettings{}

			if mac, ok := netSettings["MacAddress"].(string); ok {
				ns.MacAddress = mac
			}

			if dhcp, ok := netSettings["EnableDHCP"].(bool); ok {
				ns.EnableDHCP = dhcp
			}

			if ipv6, ok := netSettings["EnableIPv6"].(bool); ok {
				ns.EnableIPv6 = ipv6
			}

			if vlan, ok := netSettings["VLANEnabled"].(bool); ok {
				ns.VLANEnabled = vlan
			}

			if vlanid, ok := netSettings["VLANID"].(string); ok {
				ns.VLANID = vlanid
			}

			err = firmwareMgr.SetNetworkSettings(ns)
			if err != nil {
				s.Log.Error(err, "failed to update network settings")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(redfishError(err))
				return
			}
		}

		// Update boot timeout if provided
		if timeout, ok := attrs["BootTimeout"].(float64); ok {
			err = firmwareMgr.SetFirmwareTimeoutSeconds(int(timeout))
			if err != nil {
				s.Log.Error(err, "failed to update boot timeout")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(redfishError(err))
				return
			}
		}

		// Update PXE boot if provided
		if pxeBoot, ok := attrs["EnablePXEBoot"].(bool); ok {
			err = firmwareMgr.EnablePXEBoot(pxeBoot)
			if err != nil {
				s.Log.Error(err, "failed to update PXE boot")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(redfishError(err))
				return
			}
		}

		// Update HTTP boot if provided
		if httpBoot, ok := attrs["EnableHTTPBoot"].(bool); ok {
			err = firmwareMgr.EnableHTTPBoot(httpBoot)
			if err != nil {
				s.Log.Error(err, "failed to update HTTP boot")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(redfishError(err))
				return
			}
		}
	}

	// Save changes
	err = firmwareMgr.SaveChanges()
	if err != nil {
		s.Log.Error(err, "failed to save BIOS settings")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

	if req.Boot.BootSourceOverrideTarget != nil {
		s.Log.Info("setting boot source override", "system", systemId, "bootSourceOverrideTarget", *req.Boot.BootSourceOverrideTarget)

		var nextBootIndex uint16

		switch *req.Boot.BootSourceOverrideTarget {
		case Pxe:
			s.Log.Info("setting boot source override to PXE", "system", systemId)
			nextBootIndex = 1
		case Hdd:
			s.Log.Info("setting boot source override to HDD", "system", systemId)
			nextBootIndex = 2
		case None:
			s.Log.Info("clearing boot source override", "system", systemId)
		default:
			err := fmt.Errorf("invalid boot source override target: %s", *req.Boot.BootSourceOverrideTarget)
			s.Log.Error(err, "invalid boot source override target", "system", systemId)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(redfishError(err))
			return
		}

		firmwareMgr, err := firmware.NewEDK2Manager(s.firmwarePath, s.Log)
		if err != nil {
			s.Log.Error(err, "failed to create firmware manager")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(redfishError(err))
			return
		}

		if err = firmwareMgr.SetBootNext(nextBootIndex); err != nil {
			s.Log.Error(err, "failed to set boot next", "system", systemId)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(redfishError(err))
			return
		}

		if err = firmwareMgr.SaveChanges(); err != nil {
			s.Log.Error(err, "failed to save boot settings", "system", systemId)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(redfishError(err))
			return
		}
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
	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.UpdateService")
	defer span.End()

	s.Log.Info("getting update service")

	// Create response for update service
	response := UpdateService{
		OdataId:        util.Ptr("/redfish/v1/UpdateService"),
		OdataType:      util.Ptr("#UpdateService.v1_9_0.UpdateService"),
		Id:             util.Ptr("UpdateService"),
		Name:           util.Ptr("Update Service"),
		Description:    util.Ptr("Service enables updating firmware"),
		ServiceEnabled: util.Ptr(true),
		HttpPushUri:    util.Ptr("/redfish/v1/UpdateService/Actions/UpdateService.SimpleUpdate"),
		FirmwareInventory: &IdRef{
			OdataId: util.Ptr("/redfish/v1/UpdateService/FirmwareInventory"),
		},
		Actions: &UpdateServiceActions{
			HashUpdateServiceSimpleUpdate: &VirtualMediaActionsVirtualMediaEjectMedia{
				Target: util.Ptr("/redfish/v1/UpdateService/Actions/UpdateService.SimpleUpdate"),
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateServiceSimpleUpdate implements ServerInterface.
func (s *RedfishServer) UpdateServiceSimpleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "redfish.RedfishServer.UpdateServiceSimpleUpdate")
	defer span.End()

	s.Log.Info("processing firmware update")

	// Check if firmware file exists
	if s.firmwarePath == "" {
		err := errors.New("firmware path not configured")
		s.Log.Error(err, "firmware path not set")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.Log.Error(err, "failed to read request body")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Parse request body
	var request SimpleUpdateRequest
	err = json.Unmarshal(body, &request)
	if err != nil {
		s.Log.Error(err, "failed to parse request body")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Validate request
	if request.ImageURI == nil || *request.ImageURI == "" {
		err := errors.New("image URI is required")
		s.Log.Error(err, "missing image URI")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(redfishError(err))
		return
	}

	// Handle local file update
	if strings.HasPrefix(*request.ImageURI, "file://") {
		localPath := strings.TrimPrefix(*request.ImageURI, "file://")

		// Read the file
		firmwareData, err := os.ReadFile(localPath)
		if err != nil {
			s.Log.Error(err, "failed to read firmware file")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(redfishError(err))
			return
		}

		// Create firmware manager
		firmwareMgr, err := firmware.NewEDK2Manager(s.firmwarePath, s.Log)
		if err != nil {
			s.Log.Error(err, "failed to create firmware manager")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(redfishError(err))
			return
		}

		// Update firmware
		err = firmwareMgr.UpdateFirmware(firmwareData)
		if err != nil {
			s.Log.Error(err, "failed to update firmware")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(redfishError(err))
			return
		}

		s.Log.Info("firmware updated successfully")
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// For remote URIs (HTTP, HTTPS), return a task that client can monitor
	taskId := fmt.Sprintf("firmware-update-%d", time.Now().Unix())
	response := Task{
		OdataId:     util.Ptr(fmt.Sprintf("/redfish/v1/TaskService/Tasks/%s", taskId)),
		OdataType:   util.Ptr("#Task.v1_6_0.Task"),
		Id:          &taskId,
		Name:        util.Ptr("Firmware Update Task"),
		TaskState:   util.Ptr(TaskStateNew),
		StartTime:   util.Ptr(time.Now()),
		TaskMonitor: util.Ptr(fmt.Sprintf("/redfish/v1/TaskMonitor/%s", taskId)),
	}

	// Return task information
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)

	// Start background task to download and update firmware
	go s.processFirmwareUpdate(ctx, *request.ImageURI, taskId)
}

// processFirmwareUpdate handles the firmware update process in the background
func (s *RedfishServer) processFirmwareUpdate(ctx context.Context, imageURI string, taskId string) {
	s.Log.Info("starting firmware update task", "uri", imageURI, "taskId", taskId)

	// Placeholder for task update mechanism
	// In a real implementation, you would:
	// 1. Download the firmware from imageURI
	// 2. Validate the firmware image
	// 3. Update the task status as it progresses
	// 4. Apply the firmware update using the firmware manager
	// 5. Complete the task
}

// Additional response types needed for firmware management
type BiosUpdateRequest struct {
	Attributes map[string]interface{} `json:"Attributes,omitempty"`
}

type SimpleUpdateRequest struct {
	ImageURI         *string  `json:"ImageURI,omitempty"`
	TransferProtocol *string  `json:"TransferProtocol,omitempty"`
	Targets          []string `json:"Targets,omitempty"`
}
