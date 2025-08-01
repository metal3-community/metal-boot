// Package manager provides interfaces and implementations for firmware management.
package manager

import (
	"net"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/bmcpi/pibmc/internal/firmware/types"
)

// FirmwareManager provides methods to manipulate UEFI firmware variables.
type FirmwareManager interface {
	// Boot Order Management
	GetBootOrder() ([]string, error)
	SetBootOrder([]string) error
	GetBootEntries() ([]types.BootEntry, error)
	AddBootEntry(entry types.BootEntry) error
	UpdateBootEntry(id string, entry types.BootEntry) error
	DeleteBootEntry(id string) error

	SetBootNext(index uint16) error
	GetBootNext() (uint16, error)

	// Network Management
	GetNetworkSettings() (types.NetworkSettings, error)
	SetNetworkSettings(settings types.NetworkSettings) error
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
	GetSystemInfo() (types.SystemInfo, error)

	// Firmware Updates
	UpdateFirmware(firmwareData []byte) error
	GetFirmwareVersion() (string, error)

	// Operations
	SaveChanges() error
	RevertChanges() error
	ResetToDefaults() error
}
