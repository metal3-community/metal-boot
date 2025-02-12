package redfish

import (
	"github.com/gin-gonic/gin"
)

type RedfishSystem struct {
	MacAddress string `yaml:"mac"`
	IpAddress  string `yaml:"ip"`
	UnifiPort  int    `yaml:"port"`
}

type RedfishServer struct {
	Systems map[string]RedfishSystem
}

// CreateVirtualDisk implements ServerInterface.
func (r *RedfishServer) CreateVirtualDisk(c *gin.Context, computerSystemId string, storageControllerId string) {
	panic("unimplemented")
}

// DeleteVirtualdisk implements ServerInterface.
func (r *RedfishServer) DeleteVirtualdisk(c *gin.Context, computerSystemId string, storageId string) {
	panic("unimplemented")
}

// EjectVirtualMedia implements ServerInterface.
func (r *RedfishServer) EjectVirtualMedia(c *gin.Context, managerId string, virtualMediaId string) {
	panic("unimplemented")
}

// FirmwareInventory implements ServerInterface.
func (r *RedfishServer) FirmwareInventory(c *gin.Context) {
	panic("unimplemented")
}

// FirmwareInventoryDownloadImage implements ServerInterface.
func (r *RedfishServer) FirmwareInventoryDownloadImage(c *gin.Context) {
	panic("unimplemented")
}

// GetManager implements ServerInterface.
func (r *RedfishServer) GetManager(c *gin.Context, managerId string) {
	panic("unimplemented")
}

// GetManagerVirtualMedia implements ServerInterface.
func (r *RedfishServer) GetManagerVirtualMedia(c *gin.Context, managerId string, virtualMediaId string) {
	panic("unimplemented")
}

// GetRoot implements ServerInterface.
func (r *RedfishServer) GetRoot(c *gin.Context) {
	panic("unimplemented")
}

// GetSoftwareInventory implements ServerInterface.
func (r *RedfishServer) GetSoftwareInventory(c *gin.Context, softwareId string) {
	panic("unimplemented")
}

// GetSystem implements ServerInterface.
func (r *RedfishServer) GetSystem(c *gin.Context, systemId string) {
	panic("unimplemented")
}

// GetTask implements ServerInterface.
func (r *RedfishServer) GetTask(c *gin.Context, taskId string) {
	panic("unimplemented")
}

// GetTaskList implements ServerInterface.
func (r *RedfishServer) GetTaskList(c *gin.Context) {
	panic("unimplemented")
}

// GetVolumes implements ServerInterface.
func (r *RedfishServer) GetVolumes(c *gin.Context, computerSystemId string, storageControllerId string) {
	panic("unimplemented")
}

// InsertVirtualMedia implements ServerInterface.
func (r *RedfishServer) InsertVirtualMedia(c *gin.Context, managerId string, virtualMediaId string) {
	panic("unimplemented")
}

// ListManagerVirtualMedia implements ServerInterface.
func (r *RedfishServer) ListManagerVirtualMedia(c *gin.Context, managerId string) {
	panic("unimplemented")
}

// ListManagers implements ServerInterface.
func (r *RedfishServer) ListManagers(c *gin.Context) {
	panic("unimplemented")
}

// ListSystems implements ServerInterface.
func (r *RedfishServer) ListSystems(c *gin.Context) {
	panic("unimplemented")
}

// ResetIdrac implements ServerInterface.
func (r *RedfishServer) ResetIdrac(c *gin.Context) {
	panic("unimplemented")
}

// ResetSystem implements ServerInterface.
func (r *RedfishServer) ResetSystem(c *gin.Context, computerSystemId string) {
	panic("unimplemented")
}

// SetSystem implements ServerInterface.
func (r *RedfishServer) SetSystem(c *gin.Context, systemId string) {
	panic("unimplemented")
}

// UpdateService implements ServerInterface.
func (r *RedfishServer) UpdateService(c *gin.Context) {
	panic("unimplemented")
}

// UpdateServiceSimpleUpdate implements ServerInterface.
func (r *RedfishServer) UpdateServiceSimpleUpdate(c *gin.Context) {
	panic("unimplemented")
}

func NewRedfishServer() ServerInterface {
	return &RedfishServer{
		Systems: make(map[string]RedfishSystem),
	}
}
