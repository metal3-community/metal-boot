// Package file watches a file for changes and updates the in memory DHCP data.
package remote

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/netip"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"

	"github.com/bmcpi/pibmc/internal/config"
	"github.com/bmcpi/pibmc/internal/dhcp/data"
	"github.com/bmcpi/pibmc/internal/util"
	"github.com/go-logr/logr"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"go.opentelemetry.io/otel"
	"gopkg.in/yaml.v2"
)

const tracerName = "github.com/bmcpi/pibmc/backend/remote"

var (
	errRecordNotFound = fmt.Errorf("record not found")
)

// Remote represents the backend for watching a file for changes and updating the in memory DHCP data.
type Remote struct {
	// Log is the logger to be used in the File backend.
	Log logr.Logger

	config *config.Config

	client *unifi.Client

	jar *cookiejar.Jar

	dhcp map[string]data.DHCP

	netboot map[string]data.Netboot

	power map[string]data.Power
}

// NewRemote creates a new file watcher.
func NewRemote(l logr.Logger, config *config.Config) (*Remote, error) {

	client := unifi.Client{}

	if err := client.SetBaseURL(config.Unifi.Endpoint); err != nil {
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
			InsecureSkipVerify: config.Unifi.Insecure,
		},
	}

	jar, _ := cookiejar.New(nil)
	httpClient.Jar = jar

	if err := client.SetHTTPClient(httpClient); err != nil {
		panic(fmt.Sprintf("failed to set http client: %s", err))
	}

	if err := client.Login(context.Background(), config.Unifi.Username, config.Unifi.Password); err != nil {
		panic(fmt.Sprintf("failed to login: %s", err))
	}

	backend := &Remote{
		Log:     l,
		client:  &client,
		config:  config,
		jar:     jar,
		dhcp:    map[string]data.DHCP{},
		netboot: map[string]data.Netboot{},
		power:   map[string]data.Power{},
	}

	backend.loadConfigs()

	return backend, nil
}

func (w *Remote) isTokenExpired() bool {
	if w.jar == nil {
		w.jar, _ = cookiejar.New(nil)
	}

	cookies := w.jar.Cookies(&url.URL{
		Path: "/",
	})

	i := slices.IndexFunc(cookies, func(i *http.Cookie) bool {
		return i.Name == "TOKEN"
	})

	if i == -1 {
		return true
	}

	cookie := cookies[i]

	return cookie.Expires.Before(time.Now())
}

func (w *Remote) login(ctx context.Context) error {
	if w.isTokenExpired() {
		if err := w.client.Login(ctx, w.config.Unifi.Username, w.config.Unifi.Password); err != nil {
			return err
		}
	}

	return nil
}

func (w *Remote) getNetBoot(mac net.HardwareAddr) *data.Netboot {

	if w.netboot != nil {
		if netboot, ok := w.netboot[mac.String()]; ok {
			return &netboot
		}
	}

	return &data.Netboot{
		AllowNetboot: true,
		OSIE:         data.OSIE{},
	}
}

func (w *Remote) loadConfigs() error {
	errors := []error{}

	configs := map[string]any{
		"dhcp":    &w.dhcp,
		"netboot": &w.netboot,
		"power":   &w.power,
	}

	for k, v := range configs {

		backendDir := path.Dir(w.config.BackendFilePath)
		configFile := filepath.Join(backendDir, fmt.Sprintf("%s.yaml", k))

		if util.Exists(configFile) {
			f, err := os.OpenFile(configFile, os.O_RDONLY, 0644)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			defer f.Close()

			b, err := io.ReadAll(f)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			if err := yaml.Unmarshal(b, v); err != nil {
				errors = append(errors, err)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to load configs: %v", errors)
	}

	return nil
}

func (w *Remote) saveConfigs() error {

	errors := []error{}

	configs := map[string]any{
		"dhcp":    w.dhcp,
		"netboot": w.netboot,
		"power":   w.power,
	}

	for k, v := range configs {
		b, err := yaml.Marshal(v)

		if err != nil {
			errors = append(errors, err)
			continue
		}

		backendDir := path.Dir(w.config.BackendFilePath)
		configFile := filepath.Join(backendDir, fmt.Sprintf("%s.yaml", k))

		if err := os.WriteFile(configFile, b, 0644); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to save configs: %v", errors)
	}

	return nil
}

// GetByMac is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Remote) GetByMac(ctx context.Context, mac net.HardwareAddr) (*data.DHCP, *data.Netboot, *data.Power, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.GetByMac")
	defer span.End()

	dhcp, ok := w.dhcp[mac.String()]
	if !ok {
		dhcp = data.DHCP{}
		w.dhcp[mac.String()] = dhcp
	}

	dhcp.MACAddress = mac

	power, ok := w.power[mac.String()]
	if !ok {
		power = data.Power{}
		w.power[mac.String()] = power
	}

	netboot := w.getNetBoot(mac)

	if activeClient, err := w.getActiveClientByMac(ctx, mac); err == nil {

		power.Port = activeClient.SwPort

		if ipAddr, err := netip.ParseAddr(activeClient.IP); err == nil {
			dhcp.IPAddress = ipAddr
		}

		dhcp.Hostname = activeClient.Hostname
		if activeClient.VirtualNetworkOverrideID != "" {
			dhcp.VLANID = activeClient.VirtualNetworkOverrideID
		}
		dhcp.LeaseTime = 604800
		dhcp.Arch = "arm64"
		dhcp.Disabled = false

		networkId := activeClient.NetworkID
		if networkId == "" {
			networkId = activeClient.LastConnectionNetworkID
		}

		if network, err := w.client.GetNetwork(ctx, w.config.Unifi.Site, networkId); err == nil {

			if _, cidr, err := net.ParseCIDR(network.IPSubnet); err == nil {
				dhcp.SubnetMask = cidr.Mask
			}

			if network.DHCPDGateway != "" {
				if dhcpGateway, err := netip.ParseAddr(network.DHCPDGateway); err == nil {
					dhcp.DefaultGateway = dhcpGateway
				}
			}

			dhcp.NameServers = []net.IP{}

			if network.DHCPDDNS1 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS1))
			}
			if network.DHCPDDNS2 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS2))
			}
			if network.DHCPDDNS3 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS3))
			}
			if network.DHCPDDNS4 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS4))
			}

			dhcp.NTPServers = []net.IP{}

			if network.DHCPDNtp1 != "" {
				dhcp.NTPServers = append(dhcp.NTPServers, net.ParseIP(network.DHCPDNtp1))
			}
			if network.DHCPDNtp2 != "" {
				dhcp.NTPServers = append(dhcp.NTPServers, net.ParseIP(network.DHCPDNtp2))
			}
		} else {
			w.Log.Error(err, "failed to get network")
		}

	} else {
		w.Log.Error(err, "failed to get active client by mac")
	}

	if portOverrides, err := w.getPortOverride(ctx, power.Port); err == nil {
		power.State = portOverrides.PoeMode
		power.DeviceId = w.config.Unifi.Device
		power.SiteId = w.config.Unifi.Site
		power.Port = portOverrides.PortIDX
	} else {
		w.Log.Error(err, "failed to get port override")
	}

	return &dhcp, netboot, &power, nil
}

type NotFoundError struct {
}

func (e *NotFoundError) Error() string {
	return "no client found"
}

func (w *Remote) getActiveClientByMac(ctx context.Context, mac net.HardwareAddr) (*unifi.ClientInfo, error) {

	filteredClients, err := w.getActiveClientsForDevice(ctx)
	if err != nil {
		return nil, err
	}

	i := slices.IndexFunc(filteredClients, func(i unifi.ClientInfo) bool {
		if macAddr, err := net.ParseMAC(i.Mac); err == nil && macAddr != nil && macAddr.String() != "" {
			return macAddr.String() == mac.String()
		} else {
			return false
		}
	})
	if i == -1 {
		return nil, &NotFoundError{}
	}

	return &filteredClients[i], nil
}

func (w *Remote) getActiveClientsForDevice(ctx context.Context) (unifi.ClientList, error) {
	deviceMac := w.config.Unifi.Device
	lastSeenMacLookup := map[string]int{}

	device, err := w.client.GetDeviceByMACv2(ctx, w.config.Unifi.Site, deviceMac)
	if err != nil {
		return nil, err
	}
	if device.PortTable != nil {
		for _, portTable := range device.PortTable {
			if portTable.PortIdx > 10 {
				continue
			}
			if portTable.LastConnection.Mac != "" {
				if _, err := net.ParseMAC(portTable.LastConnection.Mac); err == nil {
					lastSeenMacLookup[portTable.LastConnection.Mac] = portTable.PortIdx
				}
			}
		}
	}

	filteredClients := unifi.ClientList{}

	for k, v := range lastSeenMacLookup {
		client, err := w.client.GetClientLocal(ctx, w.config.Unifi.Site, k)
		if err != nil {
			return nil, err
		}
		if client.SwPort == 0 {
			client.SwPort = v
		}
		filteredClients = append(filteredClients, *client)
	}

	return filteredClients, nil
}

func (w *Remote) getActiveClientByIP(ctx context.Context, ip net.IP) (*unifi.ClientInfo, error) {
	clients, err := w.getActiveClientsForDevice(ctx)
	if err != nil {
		return nil, err
	}

	i := slices.IndexFunc(clients, func(i unifi.ClientInfo) bool {

		ipAddr, err := netip.ParseAddr(i.IP)
		if err != nil {
			return false
		}

		return ipAddr.String() == ip.String()
	})
	if i == -1 {
		return nil, fmt.Errorf("no client found")
	}

	return &clients[i], nil
}

func (w *Remote) getPortOverride(ctx context.Context, port int) (*unifi.DevicePortOverrides, error) {

	device, err := w.client.GetDeviceByMAC(ctx, w.config.Unifi.Site, w.config.Unifi.Device)
	if err != nil {
		return nil, err
	}

	idx := slices.IndexFunc(device.PortOverrides, func(i unifi.DevicePortOverrides) bool {
		return i.PortIDX == port
	})
	if idx == -1 {
		return nil, fmt.Errorf("no port %d found", port)
	}

	return &device.PortOverrides[idx], nil
}

// GetByIP is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Remote) GetByIP(ctx context.Context, ip net.IP) (*data.DHCP, *data.Netboot, *data.Power, error) {

	err := w.login(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.GetByIP")
	defer span.End()

	var ipAddr netip.Addr

	if addr, ok := netip.AddrFromSlice(ip); !ok {
		addr, err := netip.ParseAddr(ip.String())
		if err != nil {

		} else {
			ipAddr = addr
		}
	} else {
		ipAddr = addr
	}

	dhcp := data.DHCP{
		IPAddress: ipAddr,
	}

	power := data.Power{}

	netboot := w.getNetBoot(dhcp.MACAddress)

	if activeClient, err := w.getActiveClientByIP(ctx, ip); err == nil {

		power.Port = activeClient.SwPort

		if ipAddr, err := netip.ParseAddr(activeClient.IP); err == nil {
			dhcp.IPAddress = ipAddr
		} else if ipAddr, err := netip.ParseAddr(activeClient.FixedIP); err == nil {
			dhcp.IPAddress = ipAddr
		}

		if macAddress, err := net.ParseMAC(activeClient.Mac); err == nil {
			dhcp.MACAddress = macAddress
		}

		dhcp.Hostname = activeClient.Hostname
		if activeClient.VirtualNetworkOverrideID != "" {
			dhcp.VLANID = activeClient.VirtualNetworkOverrideID
		}
		dhcp.LeaseTime = 604800
		dhcp.Arch = "arm64"
		dhcp.Disabled = false

		if network, err := w.client.GetNetwork(ctx, w.config.Unifi.Site, activeClient.NetworkID); err == nil {

			if _, cidr, err := net.ParseCIDR(network.IPSubnet); err == nil {
				dhcp.SubnetMask = cidr.Mask
			}

			if defaultGateway, err := netip.ParseAddr(network.DHCPDGateway); err == nil {
				dhcp.DefaultGateway = defaultGateway
			}

			dhcp.NameServers = []net.IP{}

			if network.DHCPDDNS1 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS1))
			}
			if network.DHCPDDNS2 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS2))
			}
			if network.DHCPDDNS3 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS3))
			}
			if network.DHCPDDNS4 != "" {
				dhcp.NameServers = append(dhcp.NameServers, net.ParseIP(network.DHCPDDNS4))
			}

			dhcp.NTPServers = []net.IP{}

			if network.DHCPDNtp1 != "" {
				dhcp.NTPServers = append(dhcp.NTPServers, net.ParseIP(network.DHCPDNtp1))
			}
			if network.DHCPDNtp2 != "" {
				dhcp.NTPServers = append(dhcp.NTPServers, net.ParseIP(network.DHCPDNtp2))
			}
		} else {
			return nil, nil, nil, err
		}

	} else {
		return nil, nil, nil, err
	}

	if dhcp.MACAddress.String() == "" {
		return nil, nil, nil, errRecordNotFound
	}

	if portOverrides, err := w.getPortOverride(ctx, power.Port); err == nil {
		power.State = portOverrides.PoeMode
		power.DeviceId = w.config.Unifi.Device
		power.SiteId = w.config.Unifi.Site
		power.Port = portOverrides.PortIDX
	} else {
		return nil, nil, nil, err
	}

	return &dhcp, netboot, &power, nil
}

func (w *Remote) Put(ctx context.Context, mac net.HardwareAddr, d *data.DHCP, n *data.Netboot, p *data.Power) error {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.Put")
	defer span.End()

	err := w.login(ctx)
	if err != nil {
		return err
	}

	if p != nil {

		pwr := data.Power{}

		if pd, ok := w.power[mac.String()]; ok {
			pwr = pd
		}

		if p.DeviceId != "" {
			pwr.DeviceId = p.DeviceId
		}

		if p.SiteId != "" {
			pwr.SiteId = p.SiteId
		}

		if p.Port != 0 {
			pwr.Port = p.Port
		}

		if p.State != "" {
			pwr.State = p.State
		}

		w.power[mac.String()] = pwr

		device, err := w.client.GetDeviceByMAC(ctx, w.config.Unifi.Site, w.config.Unifi.Device)
		if err != nil {
			return err
		}

		i := slices.IndexFunc(device.PortOverrides, func(i unifi.DevicePortOverrides) bool {
			return i.PortIDX == pwr.Port
		})
		if i == -1 {
			return fmt.Errorf("no port %d found", pwr.Port)
		}

		if device.PortOverrides[i].PoeMode != pwr.State {
			device.PortOverrides[i].PoeMode = pwr.State

			if _, err := w.client.UpdateDevice(ctx, w.config.Unifi.Site, device); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *Remote) GetKeys(ctx context.Context) ([]net.HardwareAddr, error) {

	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.GetKeys")
	defer span.End()

	err := w.login(ctx)
	if err != nil {
		return nil, err
	}

	var keys []net.HardwareAddr

	clients, err := w.getActiveClientsForDevice(ctx)
	if err != nil {
		return nil, err
	}

	for _, client := range clients {
		if mac, err := net.ParseMAC(client.Mac); err == nil {
			keys = append(keys, mac)
		}
	}

	return keys, nil
}

// PowerCycle is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Remote) PowerCycle(ctx context.Context, mac net.HardwareAddr) error {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.PowerCycle")
	defer span.End()

	err := w.login(ctx)
	if err != nil {
		return err
	}

	pwr, ok := w.power[mac.String()]

	if pwr.Port == 0 || !ok {
		pwr = data.Power{}

		activeClient, err := w.getActiveClientByMac(ctx, mac)
		if err != nil {
			if _, ok := err.(*NotFoundError); ok {
				localClient, err := w.client.GetClientLocal(ctx, w.config.Unifi.Site, mac.String())
				if err != nil {
					return err
				}
				pwr.DeviceId = localClient.LastUplinkMac
				pwr.SiteId = w.config.Unifi.Site
				pwr.Port = localClient.SwPort
				pwr.Mode = "auto"
			}
		} else {
			if activeClient.SwPort == 0 {
				return fmt.Errorf("no port found")
			}
			pwr.Port = activeClient.SwPort
			pwr.DeviceId = w.config.Unifi.Device
			pwr.SiteId = w.config.Unifi.Site
			pwr.Mode = "auto"
		}

		w.power[mac.String()] = pwr
	}

	if _, err := w.client.ExecuteCmd(ctx, w.config.Unifi.Site, "devmgr", unifi.Cmd{
		Command: "power-cycle",
		MAC:     w.config.Unifi.Device,
		PortIDX: util.Ptr(pwr.Port),
	}); err != nil {

		w.Log.Error(err, "failed to power cycle")
		return err
	}

	return nil
}

// Start starts watching a file for changes and updates the in memory data (w.data) on changew.
// Start is a blocking method. Use a context cancellation to exit.
func (w *Remote) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.Log.Info("stopping remote")
			w.Log.Info("saving dhcp")
			if err := w.saveConfigs(); err != nil {
				w.Log.Error(err, "failed to save configs")
			} else {
				w.Log.Info("configs saved")
			}
			return
		}
	}
}

func (w *Remote) Sync(ctx context.Context) error {
	k, err := w.GetKeys(ctx)
	if err != nil {
		return err
	}

	for _, key := range k {
		_, _, _, err := w.GetByMac(ctx, key)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Remote) Close() error {
	if err := w.saveConfigs(); err != nil {
		return err
	}

	return nil
}
