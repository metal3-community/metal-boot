// Package reservation is the handler for responding to DHCPv4 messages with only host reservations.
package reservation

import (
	"net"
	"net/netip"
	"net/url"

	"github.com/bmcpi/pibmc/internal/backend"
	"github.com/bmcpi/pibmc/internal/dhcp"
	"github.com/bmcpi/pibmc/internal/dhcp/data"
	"github.com/bmcpi/pibmc/internal/dhcp/lease"
	"github.com/go-logr/logr"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

// Handler holds the configuration details for the running the DHCP server.
type Handler struct {
	// Backend is the backend to use for getting DHCP data.
	Backend backend.BackendReader

	// IPAddr is the IP address to use in DHCP responses.
	// Option 54 and the sname DHCP header.
	// This could be a load balancer IP address or an ingress IP address or a local IP address.
	IPAddr netip.Addr

	// Log is used to log messages.
	// `logr.Discard()` can be used if no logging is desired.
	Log logr.Logger

	// Netboot configuration
	Netboot Netboot

	// OTELEnabled is used to determine if netboot options include otel naming.
	// When true, the netboot filename will be appended with otel information.
	// For example, the filename will be "snp.efi-00-23b1e307bb35484f535a1f772c06910e-d887dc3912240434-01".
	// <original filename>-00-<trace id>-<span id>-<trace flags>
	OTELEnabled bool

	// SyslogAddr is the address to send syslog messages to. DHCP Option 7.
	SyslogAddr netip.Addr

	// LeaseManager handles DHCP lease file operations
	LeaseManager *lease.Manager

	// ConfigManager handles DHCP configuration file operations
	ConfigManager *lease.ConfigManager

	// FallbackConfig holds the fallback configuration for unknown devices
	FallbackConfig *FallbackConfig
}

// FallbackConfig holds configuration for providing default DHCP responses to unknown devices.
type FallbackConfig struct {
	// Enabled determines if fallback responses should be sent to unknown devices
	Enabled bool

	// IPPoolStart is the starting IP address for dynamic assignment to unknown devices
	IPPoolStart netip.Addr

	// IPPoolEnd is the ending IP address for dynamic assignment to unknown devices
	IPPoolEnd netip.Addr

	// DefaultGateway is the default gateway to assign to unknown devices
	DefaultGateway netip.Addr

	// DefaultSubnetMask is the subnet mask to assign to unknown devices
	DefaultSubnetMask net.IPMask

	// DefaultNameServers are the DNS servers to assign to unknown devices
	DefaultNameServers []net.IP

	// DefaultDomainName is the domain name to assign to unknown devices
	DefaultDomainName string

	// DefaultLeaseTime is the lease time in seconds for unknown devices
	DefaultLeaseTime uint32

	// AllowNetboot determines if unknown devices should be allowed to netboot
	AllowNetboot bool
}

// generateDefaultDHCP creates default DHCP configuration for unknown devices.
func (h *Handler) generateDefaultDHCP(mac net.HardwareAddr) *data.DHCP {
	if h.FallbackConfig == nil || !h.FallbackConfig.Enabled {
		return nil
	}

	// Generate a consistent IP address based on MAC address hash
	ip := h.generateIPFromMAC(mac)

	return &data.DHCP{
		MACAddress:     mac,
		IPAddress:      ip,
		SubnetMask:     h.FallbackConfig.DefaultSubnetMask,
		DefaultGateway: h.FallbackConfig.DefaultGateway,
		NameServers:    h.FallbackConfig.DefaultNameServers,
		DomainName:     h.FallbackConfig.DefaultDomainName,
		LeaseTime:      h.FallbackConfig.DefaultLeaseTime,
		Disabled:       false,
	}
}

// generateDefaultNetboot creates default netboot configuration for unknown devices.
func (h *Handler) generateDefaultNetboot() *data.Netboot {
	if h.FallbackConfig == nil || !h.FallbackConfig.Enabled {
		return &data.Netboot{AllowNetboot: false}
	}

	return &data.Netboot{
		AllowNetboot: h.FallbackConfig.AllowNetboot,
	}
}

// generateIPFromMAC generates a consistent IP address within the configured pool based on MAC address.
func (h *Handler) generateIPFromMAC(mac net.HardwareAddr) netip.Addr {
	if h.FallbackConfig == nil {
		return netip.Addr{}
	}

	// Calculate pool size
	startBytes := h.FallbackConfig.IPPoolStart.As4()
	endBytes := h.FallbackConfig.IPPoolEnd.As4()

	// Convert to uint32 for calculation
	startIP := uint32(startBytes[0])<<24 | uint32(startBytes[1])<<16 |
		uint32(startBytes[2])<<8 | uint32(startBytes[3])
	endIP := uint32(endBytes[0])<<24 | uint32(endBytes[1])<<16 |
		uint32(endBytes[2])<<8 | uint32(endBytes[3])

	if startIP >= endIP {
		return h.FallbackConfig.IPPoolStart // fallback to start if misconfigured
	}

	poolSize := endIP - startIP + 1

	// Generate hash from MAC address
	var hash uint32
	for _, b := range mac {
		hash = hash*31 + uint32(b)
	}

	// Calculate offset within pool
	offset := hash % poolSize
	assignedIP := startIP + offset

	// Convert back to netip.Addr
	return netip.AddrFrom4([4]byte{
		byte(assignedIP >> 24),
		byte(assignedIP >> 16),
		byte(assignedIP >> 8),
		byte(assignedIP),
	})
}

// Netboot holds the netboot configuration details used in running a DHCP server.
type Netboot struct {
	// iPXE binary server IP:Port serving via TFTP.
	IPXEBinServerTFTP netip.AddrPort

	// IPXEBinServerHTTP is the URL to the IPXE binary server serving via HTTP(s).
	IPXEBinServerHTTP *url.URL

	// IPXEScriptURL is the URL to the IPXE script to use.
	IPXEScriptURL func(*dhcpv4.DHCPv4) *url.URL

	// Enabled is whether to enable sending netboot DHCP options.
	Enabled bool

	// UserClass (for network booting) allows a custom DHCP option 77 to be used to break out of an iPXE loop.
	UserClass dhcp.UserClass
}
