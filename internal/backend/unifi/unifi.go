// Package file watches a file for changes and updates the in memory DHCP data.
package unifi

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/netip"
	"slices"
	"time"

	"github.com/go-logr/logr"
	"github.com/metal3-community/metal-boot/internal/backend"
	"github.com/metal3-community/metal-boot/internal/config"
	"github.com/metal3-community/metal-boot/internal/dhcp/data"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

const tracerName = "github.com/metal3-community/metal-boot/backend/remote"

type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "no client found"
}

// Remote represents the backend for watching a file for changes and updating the in memory DHCP data.
type Remote struct {
	// Log is the logger to be used in the File backend.
	Log logr.Logger

	config *config.Config

	client *unifi.Client

	jar *cookiejar.Jar

	power map[string]data.PowerState
}

// NewRemote creates a new file watcher.
func NewRemote(
	ctx context.Context,
	l logr.Logger,
	config *config.Config,
) (backend.BackendPower, error) {
	client := unifi.Client{}

	if err := client.SetBaseURL(config.Unifi.Endpoint); err != nil {
		panic(fmt.Sprintf("failed to set base url: %s", err))
	}

	client.SetAPIKey(config.Unifi.APIKey)

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

	if err := client.Login(ctx, config.Unifi.Username, config.Unifi.Password); err != nil {
		panic(fmt.Sprintf("failed to login: %s", err))
	}

	backend := &Remote{
		Log:    l,
		client: &client,
		config: config,
		jar:    jar,
	}

	return backend, nil
}

func (w *Remote) getClient(ctx context.Context, mac net.HardwareAddr) (*unifi.ClientInfo, error) {
	client, err := w.client.GetClientLocal(
		ctx,
		w.config.Unifi.Site,
		mac.String(),
	)
	if err == nil {
		return client, nil
	}
	clientList, err := w.client.ListClientsActive(
		ctx,
		w.config.Unifi.Site,
	)
	if err != nil {
		var notFoundErr *unifi.NotFoundError
		if errors.As(err, &notFoundErr) {
			return nil, &NotFoundError{}
		}
		return nil, err
	}
	client = getClientInfo(clientList, func(c unifi.ClientInfo) bool {
		return c.Mac == mac.String()
	})
	return client, nil
}

func (w *Remote) getClientByIP(ctx context.Context, ip net.IP) (*unifi.ClientInfo, error) {
	clientList, err := w.client.ListClientsActive(
		ctx,
		w.config.Unifi.Site,
	)
	if err != nil {
		return nil, err
	}
	client := getClientInfo(clientList, func(c unifi.ClientInfo) bool {
		var activeIP net.IP
		if net.ParseIP(c.IP) != nil {
			activeIP = net.ParseIP(c.IP)
		} else if net.ParseIP(c.LastIP) != nil {
			activeIP = net.ParseIP(c.LastIP)
		}
		return ip.Equal(activeIP)
	})
	return client, nil
}

func (w *Remote) clientToDHCP(ctx context.Context, client *unifi.ClientInfo, dhcp *data.DHCP) {
	if ipAddr, err := netip.ParseAddr(client.IP); err == nil {
		dhcp.IPAddress = ipAddr
	} else if ipAddr, err := netip.ParseAddr(client.FixedIP); err == nil {
		dhcp.IPAddress = ipAddr
	}

	if macAddress, err := net.ParseMAC(client.Mac); err == nil {
		dhcp.MACAddress = macAddress
	}

	dhcp.Hostname = client.Hostname
	if client.VirtualNetworkOverrideId != "" {
		dhcp.VLANID = client.VirtualNetworkOverrideId
	}
	dhcp.LeaseTime = 604800
	dhcp.Arch = "arm64"
	dhcp.Disabled = false

	if network, err := w.client.GetNetwork(ctx, w.config.Unifi.Site, client.NetworkID); err == nil {

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
	}
}

func getClientInfo(
	clients unifi.ClientList,
	predicate func(unifi.ClientInfo) bool,
) *unifi.ClientInfo {
	var result unifi.ClientInfo
	for _, item := range clients {
		if predicate(item) {
			result = item
			break
		}
	}
	return &result
}

func (w *Remote) getDevice(ctx context.Context, mac net.HardwareAddr) (*unifi.Device, error) {
	client, err := w.getClient(ctx, mac)
	if err != nil {
		return nil, err
	}
	var deviceMac string
	if client.UplinkMac != "" {
		deviceMac = client.UplinkMac
	} else if client.LastUplinkMac != "" {
		deviceMac = client.LastUplinkMac
	} else {
		return nil, fmt.Errorf("no uplink mac found for client %s", mac.String())
	}

	device, err := w.client.GetDeviceByMAC(ctx, w.config.Unifi.Site, deviceMac)
	if err != nil {
		return nil, err
	}

	return device, nil
}

func (w *Remote) getPortIdx(
	mac net.HardwareAddr,
	device *unifi.Device,
) (int, error) {
	i := slices.IndexFunc(device.PortTable, func(i unifi.DevicePortTable) bool {
		return i.LastConnection.MAC == mac.String()
	})
	if i == -1 {
		return -1, fmt.Errorf("no port found for mac %s", mac.String())
	}
	return device.PortTable[i].PortIdx, nil
}

func (w *Remote) getPortTable(
	ctx context.Context,
	mac net.HardwareAddr,
	device *unifi.Device,
) (*unifi.DevicePortTable, error) {
	i := slices.IndexFunc(device.PortTable, func(i unifi.DevicePortTable) bool {
		return i.LastConnection.MAC == mac.String()
	})
	if i == -1 {
		return nil, fmt.Errorf("no port found for mac %s", mac.String())
	}
	return &device.PortTable[i], nil
}
