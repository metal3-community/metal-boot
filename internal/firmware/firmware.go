// Package firmware provides firmware management functionality.
package firmware

import (
	"github.com/bmcpi/pibmc/internal/firmware/manager"
	"github.com/bmcpi/pibmc/internal/firmware/util"
	"github.com/go-logr/logr"
)

// CreateManager creates a new firmware manager for the given firmware file.
func CreateManager(firmwarePath string, logger logr.Logger) (manager.FirmwareManager, error) {
	return manager.NewEDK2Manager(firmwarePath, logger)
}

// CreateNetworkManager creates a firmware manager optimized for network booting.
func CreateNetworkManager(
	firmwarePath string,
	logger logr.Logger,
) (manager.FirmwareManager, error) {
	return util.CreateBootNetworkManager(firmwarePath, logger)
}
