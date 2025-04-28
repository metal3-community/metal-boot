package firmware_test

import (
	"os"
	"testing"

	"github.com/bmcpi/pibmc/internal/firmware"
	"github.com/bmcpi/pibmc/internal/firmware/manager"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateManager(t *testing.T) {
	logger := logr.FromContextOrDiscard(t.Context()).WithName("edk2-manager-test")

	// Create a temporary firmware file for testing
	tempFile, err := os.CreateTemp("", "test-firmware-*.bin")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write some dummy data to the file
	_, err = tempFile.Write([]byte("dummy firmware data"))
	require.NoError(t, err)

	// Test creating a manager with valid firmware file
	t.Run("ValidFirmware", func(t *testing.T) {
		mgr, err := firmware.CreateManager(tempFile.Name(), logger)
		assert.NoError(t, err)
		assert.NotNil(t, mgr)
		assert.IsType(t, &manager.EDK2Manager{}, mgr)
	})

	// Test creating a manager with invalid firmware file
	t.Run("InvalidFirmware", func(t *testing.T) {
		mgr, err := firmware.CreateManager("/nonexistent/path", logger)
		assert.Error(t, err)
		assert.Nil(t, mgr)
		assert.Contains(t, err.Error(), "firmware file not found")
	})
}

func TestCreateNetworkManager(t *testing.T) {
	logger := logr.FromContextOrDiscard(t.Context()).WithName("edk2-manager-test")

	// Create a temporary firmware file for testing
	tempFile, err := os.CreateTemp("", "test-firmware-*.bin")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write some dummy data to the file
	_, err = tempFile.Write([]byte("dummy firmware data"))
	require.NoError(t, err)

	// Test creating a network manager with valid firmware file
	t.Run("ValidFirmware", func(t *testing.T) {
		mgr, err := firmware.CreateNetworkManager(tempFile.Name(), logger)
		assert.NoError(t, err)
		assert.NotNil(t, mgr)
	})

	// Test creating a network manager with invalid firmware file
	t.Run("InvalidFirmware", func(t *testing.T) {
		mgr, err := firmware.CreateNetworkManager("/nonexistent/path", logger)
		assert.Error(t, err)
		assert.Nil(t, mgr)
	})
}
