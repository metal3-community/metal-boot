package util_test

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/bmcpi/pibmc/internal/firmware/types"
	"github.com/bmcpi/pibmc/internal/firmware/util"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockFirmwareManager is a mock implementation of the FirmwareManager interface
type MockFirmwareManager struct {
	mock.Mock
}

// Implement all the required methods of the FirmwareManager interface
func (m *MockFirmwareManager) GetBootOrder() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockFirmwareManager) SetBootOrder(order []string) error {
	args := m.Called(order)
	return args.Error(0)
}

func (m *MockFirmwareManager) GetBootEntries() ([]types.BootEntry, error) {
	args := m.Called()
	return args.Get(0).([]types.BootEntry), args.Error(1)
}

func (m *MockFirmwareManager) AddBootEntry(entry types.BootEntry) error {
	args := m.Called(entry)
	return args.Error(0)
}

func (m *MockFirmwareManager) UpdateBootEntry(id string, entry types.BootEntry) error {
	args := m.Called(id, entry)
	return args.Error(0)
}

func (m *MockFirmwareManager) DeleteBootEntry(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockFirmwareManager) SetBootNext(index uint16) error {
	args := m.Called(index)
	return args.Error(0)
}

func (m *MockFirmwareManager) GetBootNext() (uint16, error) {
	args := m.Called()
	return args.Get(0).(uint16), args.Error(1)
}

func (m *MockFirmwareManager) GetNetworkSettings() (types.NetworkSettings, error) {
	args := m.Called()
	return args.Get(0).(types.NetworkSettings), args.Error(1)
}

func (m *MockFirmwareManager) SetNetworkSettings(settings types.NetworkSettings) error {
	args := m.Called(settings)
	return args.Error(0)
}

func (m *MockFirmwareManager) GetMacAddress() (net.HardwareAddr, error) {
	args := m.Called()
	return args.Get(0).(net.HardwareAddr), args.Error(1)
}

func (m *MockFirmwareManager) SetMacAddress(mac net.HardwareAddr) error {
	args := m.Called(mac)
	return args.Error(0)
}

func (m *MockFirmwareManager) GetVariable(name string) (*efi.EfiVar, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*efi.EfiVar), args.Error(1)
}

func (m *MockFirmwareManager) SetVariable(name string, value *efi.EfiVar) error {
	args := m.Called(name, value)
	return args.Error(0)
}

func (m *MockFirmwareManager) ListVariables() (map[string]*efi.EfiVar, error) {
	args := m.Called()
	return args.Get(0).(map[string]*efi.EfiVar), args.Error(1)
}

func (m *MockFirmwareManager) EnablePXEBoot(enable bool) error {
	args := m.Called(enable)
	return args.Error(0)
}

func (m *MockFirmwareManager) EnableHTTPBoot(enable bool) error {
	args := m.Called(enable)
	return args.Error(0)
}

func (m *MockFirmwareManager) SetFirmwareTimeoutSeconds(seconds int) error {
	args := m.Called(seconds)
	return args.Error(0)
}

func (m *MockFirmwareManager) SetConsoleConfig(consoleName string, baudRate int) error {
	args := m.Called(consoleName, baudRate)
	return args.Error(0)
}

func (m *MockFirmwareManager) GetSystemInfo() (types.SystemInfo, error) {
	args := m.Called()
	return args.Get(0).(types.SystemInfo), args.Error(1)
}

func (m *MockFirmwareManager) UpdateFirmware(firmwareData []byte) error {
	args := m.Called(firmwareData)
	return args.Error(0)
}

func (m *MockFirmwareManager) GetFirmwareVersion() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockFirmwareManager) SaveChanges() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockFirmwareManager) RevertChanges() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockFirmwareManager) ResetToDefaults() error {
	args := m.Called()
	return args.Error(0)
}

func TestCreateBootNetworkManager(t *testing.T) {
	// Create a temporary file for the test
	tmpFile, err := os.CreateTemp("", "firmware-*.bin")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	logger := logr.FromContextOrDiscard(t.Context()).WithName("create-boot-network-manager-test")

	t.Run("SuccessfulCreation", func(t *testing.T) {
		// Test successful creation
		manager, err := util.CreateBootNetworkManager(tmpFile.Name(), logger)
		assert.NoError(t, err)
		assert.NotNil(t, manager)
	})

	t.Run("FileNotExist", func(t *testing.T) {
		// Test with file that doesn't exist
		manager, err := util.CreateBootNetworkManager("/non/existent/path", logger)
		assert.Error(t, err)
		assert.Nil(t, manager)
	})
}

func TestConfigureNetworkBoot(t *testing.T) {
	mockManager := new(MockFirmwareManager)
	mac, _ := net.ParseMAC("00:11:22:33:44:55")

	t.Run("SuccessfulConfiguration", func(t *testing.T) {
		// Setup expectations
		mockManager.On("SetMacAddress", mac).Return(nil)
		mockManager.On("EnablePXEBoot", true).Return(nil)
		mockManager.On("EnableHTTPBoot", true).Return(nil)
		mockManager.On("SetFirmwareTimeoutSeconds", 5).Return(nil)
		mockManager.On("SaveChanges").Return(nil)

		// Configure network boot
		err := util.ConfigureNetworkBoot(mockManager, mac, true, 5)
		assert.NoError(t, err)
	})

	t.Run("ErrorInSetMacAddress", func(t *testing.T) {
		// Clear previous calls
		mockManager = new(MockFirmwareManager)

		// Setup expectations for failure
		mockManager.On("SetMacAddress", mac).Return(errors.New("mac error"))

		// Configure network boot
		err := util.ConfigureNetworkBoot(mockManager, mac, true, 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set MAC address")
	})

	t.Run("ErrorInEnablePXEBoot", func(t *testing.T) {
		// Clear previous calls
		mockManager = new(MockFirmwareManager)

		// Setup expectations for failure
		mockManager.On("SetMacAddress", mac).Return(nil)
		mockManager.On("EnablePXEBoot", true).Return(errors.New("pxe error"))

		// Configure network boot
		err := util.ConfigureNetworkBoot(mockManager, mac, true, 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to enable PXE boot")
	})

	t.Run("ErrorInEnableHTTPBoot", func(t *testing.T) {
		// Clear previous calls
		mockManager = new(MockFirmwareManager)

		// Setup expectations for failure
		mockManager.On("SetMacAddress", mac).Return(nil)
		mockManager.On("EnablePXEBoot", true).Return(nil)
		mockManager.On("EnableHTTPBoot", true).Return(errors.New("http error"))

		// Configure network boot
		err := util.ConfigureNetworkBoot(mockManager, mac, true, 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to enable HTTP boot")
	})

	t.Run("ErrorInSetFirmwareTimeoutSeconds", func(t *testing.T) {
		// Clear previous calls
		mockManager = new(MockFirmwareManager)

		// Setup expectations for failure
		mockManager.On("SetMacAddress", mac).Return(nil)
		mockManager.On("EnablePXEBoot", true).Return(nil)
		mockManager.On("EnableHTTPBoot", true).Return(nil)
		mockManager.On("SetFirmwareTimeoutSeconds", 5).Return(errors.New("timeout error"))

		// Configure network boot
		err := util.ConfigureNetworkBoot(mockManager, mac, true, 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set boot timeout")
	})

	t.Run("ErrorInSaveChanges", func(t *testing.T) {
		// Clear previous calls
		mockManager = new(MockFirmwareManager)

		// Setup expectations for failure
		mockManager.On("SetMacAddress", mac).Return(nil)
		mockManager.On("EnablePXEBoot", true).Return(nil)
		mockManager.On("EnableHTTPBoot", true).Return(nil)
		mockManager.On("SetFirmwareTimeoutSeconds", 5).Return(nil)
		mockManager.On("SaveChanges").Return(errors.New("save error"))

		// Configure network boot
		err := util.ConfigureNetworkBoot(mockManager, mac, true, 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save changes")
	})
}

func TestFileExists(t *testing.T) {
	// Create a temporary file for the test
	tmpFile, err := os.CreateTemp("", "file-exists-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Test with existing file
	assert.True(t, util.FileExists(tmpFile.Name()))

	// Test with non-existent file
	assert.False(t, util.FileExists("/path/to/nonexistent/file"))
}

func TestCopyFirmwareFile(t *testing.T) {
	srcDir, err := os.MkdirTemp("", "firmware-src-*")
	require.NoError(t, err)
	defer os.RemoveAll(srcDir)

	destDir, err := os.MkdirTemp("", "firmware-dest-*")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	// Create source file
	srcFile := filepath.Join(srcDir, "firmware.bin")
	testContent := []byte("firmware content")
	err = os.WriteFile(srcFile, testContent, 0644)
	require.NoError(t, err)

	// Test copying
	destFile := filepath.Join(destDir, "firmware-copy.bin")
	err = util.CopyFirmwareFile(srcFile, destFile)
	assert.NoError(t, err)

	// Verify file content
	destContent, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, testContent, destContent)

	// Test with non-existent source
	err = util.CopyFirmwareFile("/nonexistent/source", destFile)
	assert.Error(t, err)
}
