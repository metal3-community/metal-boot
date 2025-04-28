package firmware_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmcpi/pibmc/internal/firmware"
	"github.com/bmcpi/pibmc/internal/firmware/manager"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockLogger is a mock implementation of logr.Logger
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Enabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

func (m *MockLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	args := []interface{}{err, msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

func (m *MockLogger) V(level int) logr.Logger {
	args := m.Called(level)
	return args.Get(0).(logr.Logger)
}

func (m *MockLogger) WithValues(keysAndValues ...interface{}) logr.Logger {
	args := m.Called(keysAndValues)
	return args.Get(0).(logr.Logger)
}

func (m *MockLogger) WithName(name string) logr.Logger {
	args := m.Called(name)
	return args.Get(0).(logr.Logger)
}

func TestCreateManager(t *testing.T) {
	mockLogger := new(MockLogger)
	mockLogger.On("WithName", "edk2-manager").Return(mockLogger)

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
		mgr, err := firmware.CreateManager(tempFile.Name(), mockLogger)
		assert.NoError(t, err)
		assert.NotNil(t, mgr)
		assert.IsType(t, &manager.EDK2Manager{}, mgr)
	})

	// Test creating a manager with invalid firmware file
	t.Run("InvalidFirmware", func(t *testing.T) {
		mgr, err := firmware.CreateManager("/nonexistent/path", mockLogger)
		assert.Error(t, err)
		assert.Nil(t, mgr)
		assert.Contains(t, err.Error(), "firmware file not found")
	})

	mockLogger.AssertExpectations(t)
}

func TestCreateNetworkManager(t *testing.T) {
	mockLogger := new(MockLogger)
	mockLogger.On("WithName", "edk2-manager").Return(mockLogger)

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
		mgr, err := firmware.CreateNetworkManager(tempFile.Name(), mockLogger)
		assert.NoError(t, err)
		assert.NotNil(t, mgr)
	})

	// Test creating a network manager with invalid firmware file
	t.Run("InvalidFirmware", func(t *testing.T) {
		mgr, err := firmware.CreateNetworkManager("/nonexistent/path", mockLogger)
		assert.Error(t, err)
		assert.Nil(t, mgr)
	})

	mockLogger.AssertExpectations(t)
}
