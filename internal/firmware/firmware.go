package firmware

import (
	"net"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
)

// NetworkSettings contains network-related UEFI settings
type NetworkSettings struct {
	MacAddress  string
	IPAddress   string
	SubnetMask  string
	Gateway     string
	DNSServers  []string
	EnableIPv6  bool
	EnableDHCP  bool
	VLANEnabled bool
	VLANID      string
}

// BootEntry represents a single UEFI boot entry
type BootEntry struct {
	ID       string
	Name     string
	DevPath  string
	Enabled  bool
	Position int
}

// FirmwareManager provides methods to manipulate UEFI firmware variables
type FirmwareManager interface {
	// Boot Order Management
	GetBootOrder() ([]string, error)
	SetBootOrder([]string) error
	GetBootEntries() ([]BootEntry, error)
	AddBootEntry(entry BootEntry) error
	UpdateBootEntry(id string, entry BootEntry) error
	DeleteBootEntry(id string) error

	// Network Management
	GetNetworkSettings() (NetworkSettings, error)
	SetNetworkSettings(settings NetworkSettings) error
	GetMacAddress() (net.HardwareAddr, error)
	SetMacAddress(mac net.HardwareAddr) error

	// UEFI Variable Management
	GetVariable(name string) (*efi.EfiVar, error)
	SetVariable(name string, value *efi.EfiVar) error
	ListVariables() (map[string]*efi.EfiVar, error)

	// Boot Configuration
	EnablePXEBoot(enable bool) error
	EnableHTTPBoot(enable bool) error
	SetFirmwareTimeoutSeconds(seconds int) error

	// Device Specific Settings
	SetConsoleConfig(consoleName string, baudRate int) error
	GetSystemInfo() (map[string]string, error)

	// Firmware Updates
	UpdateFirmware(firmwareData []byte) error
	GetFirmwareVersion() (string, error)

	// Operations
	SaveChanges() error
	RevertChanges() error
	ResetToDefaults() error
}
