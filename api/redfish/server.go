package redfish

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

type RedfishServerConfig struct {
	Insecure      bool
	UnifiUser     string
	UnifiPass     string
	UnifiEndpoint string
	UnifiSite     string
	UnifiDevice   string
}

type RedfishSystem struct {
	MacAddress string `yaml:"mac"`
	IpAddress  string `yaml:"ip"`
	UnifiPort  int    `yaml:"port"`
	SiteID     string `yaml:"site"`
	DeviceMac  string `yaml:"device_mac"`
	PoeMode    string `yaml:"poe_mode"`
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

type RedfishServer struct {
	Systems map[int]RedfishSystem

	Config *RedfishServerConfig

	client *unifi.Client
}

func NewRedfishServer(cfg RedfishServerConfig) ServerInterface {
	client := unifi.Client{}

	if err := client.SetBaseURL(cfg.UnifiEndpoint); err != nil {
		panic(fmt.Sprintf("failed to set base url: %s", err))
	}

	httpClient := &http.Client{}
	httpClient.Transport = &http.Transport{
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
			InsecureSkipVerify: cfg.Insecure,
		},
	}

	jar, _ := cookiejar.New(nil)
	httpClient.Jar = jar

	if err := client.SetHTTPClient(httpClient); err != nil {
		panic(fmt.Sprintf("failed to set http client: %s", err))
	}

	if err := client.Login(context.Background(), cfg.UnifiUser, cfg.UnifiPass); err != nil {
		panic(fmt.Sprintf("failed to login: %s", err))
	}

	rfSystems := make(map[int]RedfishSystem)

	server := &RedfishServer{
		Systems: rfSystems,
		client:  &client,
		Config:  &cfg,
	}

	server.refreshSystems(context.Background())

	return server
}

func (r *RedfishServer) refreshSystems(ctx context.Context) (err error) {
	device, err := r.client.GetDeviceByMAC(ctx, r.Config.UnifiSite, r.Config.UnifiDevice)
	if err != nil {
		panic(err)
	}

	if device.PortOverrides == nil {
		panic("no port overrides found")
	}

	for _, port := range device.PortOverrides {

		sys, ok := r.Systems[port.PortIDX]
		if !ok {
			sys = RedfishSystem{
				UnifiPort: port.PortIDX,
				DeviceMac: device.MAC,
				SiteID:    device.SiteID,
			}
		}
		sys.PoeMode = port.PoeMode

		r.Systems[port.PortIDX] = sys
	}

	if clients, err := r.client.ListActiveClients(ctx, r.Config.UnifiSite); err != nil {
		panic(err)
	} else {
		for _, c := range clients {

			if c.UplinkMac == r.Config.UnifiDevice {

				sys, ok := r.Systems[c.SwPort]
				if !ok {
					sys = RedfishSystem{
						UnifiPort: c.SwPort,
						DeviceMac: c.UplinkMac,
						SiteID:    c.SiteID,
					}
				}

				sys.MacAddress = c.Mac
				sys.IpAddress = c.IP

				r.Systems[c.SwPort] = sys
			}
		}
	}

	return
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
		err = fmt.Errorf("port %d not found on device %s", p, deviceId)
		return
	}

	port = dev.PortOverrides[iPort]

	return
}

// CreateVirtualDisk implements ServerInterface.
func (r *RedfishServer) CreateVirtualDisk(c *gin.Context, systemId string, storageControllerId string) {

	req := CreateVirtualDiskRequestBody{}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	panic("unimplemented")
}

// DeleteVirtualdisk implements ServerInterface.
func (r *RedfishServer) DeleteVirtualdisk(c *gin.Context, systemId string, storageId string) {
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

	systemIdInt, err := strconv.ParseInt(systemId, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	s := r.Systems[int(systemIdInt)]

	resp := ComputerSystem{
		Id:         &systemId,
		PowerState: s.GetPowerState(),
	}

	c.JSON(200, &resp)
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
func (r *RedfishServer) GetVolumes(c *gin.Context, systemId string, storageControllerId string) {
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
func (r *RedfishServer) ResetSystem(c *gin.Context, systemId string) {
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
