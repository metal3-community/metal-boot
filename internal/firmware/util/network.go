// Package util provides utility functions for firmware management.
package util

import (
	"fmt"
	"net"
	"os"

	"github.com/bmcpi/pibmc/internal/firmware/manager"
	"github.com/go-logr/logr"
)

// CreateBootNetworkManager creates a firmware manager configured specifically for network booting.
func CreateBootNetworkManager(
	firmwarePath string,
	logger logr.Logger,
) (manager.FirmwareManager, error) {
	// Create the manager with the specified firmware file
	mgr, err := manager.NewEDK2Manager(firmwarePath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create firmware manager: %w", err)
	}

	return mgr, nil
}

// ConfigureNetworkBoot sets up the firmware for optimal network booting.
func ConfigureNetworkBoot(
	mgr manager.FirmwareManager,
	mac net.HardwareAddr,
	enableIPv6 bool,
	timeout int,
) error {
	// Set the MAC address
	if err := mgr.SetMacAddress(mac); err != nil {
		return fmt.Errorf("failed to set MAC address: %w", err)
	}

	// Enable PXE boot
	if err := mgr.EnablePXEBoot(true); err != nil {
		return fmt.Errorf("failed to enable PXE boot: %w", err)
	}

	// Enable HTTP boot if needed
	if err := mgr.EnableHTTPBoot(true); err != nil {
		return fmt.Errorf("failed to enable HTTP boot: %w", err)
	}

	// Set boot timeout
	if err := mgr.SetFirmwareTimeoutSeconds(timeout); err != nil {
		return fmt.Errorf("failed to set boot timeout: %w", err)
	}

	// Save changes
	if err := mgr.SaveChanges(); err != nil {
		return fmt.Errorf("failed to save changes: %w", err)
	}

	return nil
}

// FileExists checks if a file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CopyFirmwareFile copies a firmware file to the specified destination.
func CopyFirmwareFile(src, dst string) error {
	return CopyFile(src, dst)
}
