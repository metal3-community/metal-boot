package firmware

import (
	"fmt"
	"net"
	"os"

	"github.com/go-logr/logr"
)

// CreateBootNetworkManager creates a firmware manager configured specifically for network booting
func CreateBootNetworkManager(firmwarePath string, logger logr.Logger) (FirmwareManager, error) {
	// Create the manager with the specified firmware file
	manager, err := NewEDK2Manager(firmwarePath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create firmware manager: %w", err)
	}

	return manager, nil
}

// ConfigureNetworkBoot sets up the firmware for optimal network booting
func ConfigureNetworkBoot(manager FirmwareManager, mac net.HardwareAddr, enableIPv6 bool, timeout int) error {
	// Set the MAC address
	if err := manager.SetMacAddress(mac); err != nil {
		return fmt.Errorf("failed to set MAC address: %w", err)
	}

	// Set network settings
	networkSettings := NetworkSettings{
		MacAddress:  mac.String(),
		EnableDHCP:  true, // Typically we want DHCP for netbooting
		EnableIPv6:  enableIPv6,
		VLANEnabled: false, // Set to true and provide VLANID if needed
	}

	if err := manager.SetNetworkSettings(networkSettings); err != nil {
		return fmt.Errorf("failed to set network settings: %w", err)
	}

	// Enable PXE booting
	if err := manager.EnablePXEBoot(true); err != nil {
		return fmt.Errorf("failed to enable PXE boot: %w", err)
	}

	// Set boot timeout (0 means no timeout)
	if err := manager.SetFirmwareTimeoutSeconds(timeout); err != nil {
		return fmt.Errorf("failed to set boot timeout: %w", err)
	}

	// Save the changes to the firmware
	if err := manager.SaveChanges(); err != nil {
		return fmt.Errorf("failed to save firmware changes: %w", err)
	}

	return nil
}

// BackupFirmware creates a backup of the firmware file
func BackupFirmware(firmwarePath string) (string, error) {
	backupPath := firmwarePath + ".backup"

	sourceFile, err := os.ReadFile(firmwarePath)
	if err != nil {
		return "", fmt.Errorf("failed to read firmware file: %w", err)
	}

	if err := os.WriteFile(backupPath, sourceFile, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupPath, nil
}

// CreateCustomBootEntry creates a custom boot entry for an iPXE binary
func CreateCustomBootEntry(manager FirmwareManager, name string, ipxeUrl string, position int) error {
	// Create a new boot entry for iPXE
	entry := BootEntry{
		Name:     name,
		DevPath:  fmt.Sprintf("URI(%s)", ipxeUrl),
		Enabled:  true,
		Position: position,
	}

	if err := manager.AddBootEntry(entry); err != nil {
		return fmt.Errorf("failed to add iPXE boot entry: %w", err)
	}

	return nil
}
