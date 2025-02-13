package redfish

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

type RedfishSystem struct {
	MacAddress string `yaml:"mac"`
	IpAddress  string `yaml:"ip"`
	UnifiPort  int    `yaml:"port"`
	SiteID     string `yaml:"site"`
	DeviceMac  string `yaml:"device_mac"`
}

type RedfishServer struct {
	Systems map[string]RedfishSystem

	client *unifi.Client
}

func NewRedfishServer(insecure bool) ServerInterface {
	client := unifi.Client{}

	httpClient := &http.Client{}
	httpClient.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}

	jar, _ := cookiejar.New(nil)
	httpClient.Jar = jar

	if err := client.SetHTTPClient(httpClient); err != nil {
		panic(fmt.Sprintf("failed to set http client: %s", err))
	}

	if err := client.Login(context.Background(), "admin", "password"); err != nil {
		panic(fmt.Sprintf("failed to login: %s", err))
	}

	rfSystems := make(map[string]RedfishSystem)

	if clients, err := client.ListActiveClients(context.Background(), "default"); err != nil {
		panic(err)
	} else {
		for _, c := range clients {

			rfSystem := RedfishSystem{
				MacAddress: c.Mac,
				IpAddress:  c.IP,
				UnifiPort:  c.SwPort,
				SiteID:     c.SiteID,
				DeviceMac:  c.UplinkMac,
			}

			rfSystems[c.Mac] = rfSystem
		}
	}

	return &RedfishServer{
		Systems: rfSystems,
		client:  &client,
	}
}

func (r *RedfishServer) getPortState(ctx context.Context, macAddress string, p int) (deviceId string, port unifi.DevicePortOverrides, err error) {
	dev, err := r.client.GetDeviceByMAC(ctx, "default", macAddress)
	if err != nil {
		err = fmt.Errorf("error getting device by MAC Address %s: %v", macAddress, err)
		return
	}

	deviceId = dev.ID

	iPort := slices.IndexFunc(dev.PortOverrides, func(pd unifi.DevicePortOverrides) bool {
		return pd.PortIDX == p
	})

	if iPort == -1 {
		err = fmt.Errorf("port %s not found on device %s", p, deviceId)
		return
	}

	port = dev.PortOverrides[iPort]

	return
}

// CreateVirtualDisk implements ServerInterface.
func (r *RedfishServer) CreateVirtualDisk(c *gin.Context, computerSystemId string, storageControllerId string) {

	req := CreateVirtualDiskRequestBody{}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

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

	s := r.Systems[systemId]

	_, powerState, err := r.getPortState(c, s.DeviceMac, s.UnifiPort)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	powerstate := PowerState(Off)

	switch powerState.PoeMode {
	case "auto":
		powerstate = On
	case "off":
		powerstate = Off
	}

	resp := ComputerSystem{
		Id:         &systemId,
		PowerState: &powerstate,
	}

	c.JSON(200, &resp)

	return
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

	req := SetSystemJSONRequestBody{}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

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
