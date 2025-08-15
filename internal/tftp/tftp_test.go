package tftp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmcpi/pibmc/internal/backend"
	"github.com/bmcpi/pibmc/internal/dhcp/data"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tinkerbell/ipxedust/binary"
)

// mockBackend implements backend.BackendReader for testing.
type mockBackend struct {
	mock.Mock
}

func (m *mockBackend) GetByIP(
	ctx context.Context,
	ip net.IP,
) (*data.DHCP, *data.Netboot, error) {
	args := m.Called(ctx, ip)
	dhcp := args.Get(0)
	netboot := args.Get(1)
	err := args.Error(2)

	var dhcpPtr *data.DHCP
	if dhcp != nil {
		dhcpPtr = dhcp.(*data.DHCP)
	}

	var netbootPtr *data.Netboot
	if netboot != nil {
		netbootPtr = netboot.(*data.Netboot)
	}

	return dhcpPtr, netbootPtr, err
}

func (m *mockBackend) GetByMac(
	ctx context.Context,
	mac net.HardwareAddr,
) (*data.DHCP, *data.Netboot, error) {
	args := m.Called(ctx, mac)
	return args.Get(0).(*data.DHCP), args.Get(1).(*data.Netboot), args.Error(
		2,
	)
}

func (m *mockBackend) GetKeys(ctx context.Context) ([]net.HardwareAddr, error) {
	args := m.Called(ctx)
	return args.Get(0).([]net.HardwareAddr), args.Error(1)
}

// mockReaderFrom implements io.ReaderFrom for testing.
type mockReaderFrom struct {
	*bytes.Buffer
}

func newMockReaderFrom() *mockReaderFrom {
	return &mockReaderFrom{
		Buffer: &bytes.Buffer{},
	}
}

func (m *mockReaderFrom) ReadFrom(r io.Reader) (int64, error) {
	return m.Buffer.ReadFrom(r)
}

// mockOutgoingTransfer implements tftp.OutgoingTransfer for testing.
type mockOutgoingTransfer struct {
	io.ReaderFrom
	remoteAddr net.UDPAddr
}

func (m *mockOutgoingTransfer) RemoteAddr() net.UDPAddr {
	return m.remoteAddr
}

// mockIncomingTransfer implements tftp.IncomingTransfer for testing.
type mockIncomingTransfer struct {
	io.WriterTo
	remoteAddr net.UDPAddr
}

func (m *mockIncomingTransfer) RemoteAddr() net.UDPAddr {
	return m.remoteAddr
}

func TestHandler_HandleRead(t *testing.T) {
	// Setup test directory
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := []byte("test file content")
	require.NoError(t, os.WriteFile(testFile, testContent, 0o644))

	// Create MAC address for testing
	mac, err := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	require.NoError(t, err)

	tests := []struct {
		name          string
		fullfilepath  string
		setupBackend  func(*mockBackend)
		isWrite       bool
		expectedError error
		expectedData  []byte
		rootDirectory string
	}{
		{
			name:         "read: serve iPXE binary with default patch",
			fullfilepath: "undionly.kpxe",
			setupBackend: func(mb *mockBackend) {
				dhcp := &data.DHCP{MACAddress: mac}
				mb.On("GetByIP", mock.Anything, mock.Anything).
					Return(dhcp, (*data.Netboot)(nil), (*data.Power)(nil), nil)
			},
			rootDirectory: tempDir,
		},
		{
			name:         "read: serve iPXE binary with custom patch from netboot",
			fullfilepath: "undionly.kpxe",
			setupBackend: func(mb *mockBackend) {
				dhcp := &data.DHCP{MACAddress: mac}
				netboot := &data.Netboot{IPXEScript: "custom script"}
				mb.On("GetByIP", mock.Anything, mock.Anything).
					Return(dhcp, netboot, (*data.Power)(nil), nil)
			},
			rootDirectory: tempDir,
		},
		{
			name:         "read: serve file from filesystem",
			fullfilepath: "test.txt",
			setupBackend: func(mb *mockBackend) {
				dhcp := &data.DHCP{MACAddress: mac}
				mb.On("GetByIP", mock.Anything, mock.Anything).
					Return(dhcp, (*data.Netboot)(nil), (*data.Power)(nil), nil)
			},
			expectedData:  testContent,
			rootDirectory: tempDir,
		},
		{
			name:         "read: resolve serial to MAC address",
			fullfilepath: "12a34567/test.txt",
			setupBackend: func(mb *mockBackend) {
				dhcp := &data.DHCP{MACAddress: mac}
				mb.On("GetByIP", mock.Anything, mock.Anything).
					Return(dhcp, (*data.Netboot)(nil), (*data.Power)(nil), nil)
			},
			rootDirectory: tempDir,
		},
		{
			name:         "read: file not found returns ErrNotExist",
			fullfilepath: "nonexistent.txt",
			setupBackend: func(mb *mockBackend) {
				dhcp := &data.DHCP{MACAddress: mac}
				mb.On("GetByIP", mock.Anything, mock.Anything).
					Return(dhcp, (*data.Netboot)(nil), (*data.Power)(nil), nil)
			},
			expectedError: os.ErrNotExist,
			rootDirectory: tempDir,
		},
		{
			name:         "read: DHCP info error logs but continues",
			fullfilepath: "test.txt",
			setupBackend: func(mb *mockBackend) {
				mb.On("GetByIP", mock.Anything, mock.Anything).
					Return((*data.DHCP)(nil), (*data.Netboot)(nil), (*data.Power)(nil), errors.New("backend error"))
			},
			expectedData:  testContent,
			rootDirectory: tempDir,
		},
		{
			name:         "read: invalid root directory",
			fullfilepath: "test.txt",
			setupBackend: func(mb *mockBackend) {
				dhcp := &data.DHCP{MACAddress: mac}
				mb.On("GetByIP", mock.Anything, mock.Anything).
					Return(dhcp, (*data.Netboot)(nil), (*data.Power)(nil), nil)
			},
			rootDirectory: "/nonexistent/path",
		},
		{
			name:         "read: MAC-specific path fallback to generic",
			fullfilepath: "aa:bb:cc:dd:ee:ff/generic.txt",
			setupBackend: func(mb *mockBackend) {
				dhcp := &data.DHCP{MACAddress: mac}
				mb.On("GetByIP", mock.Anything, mock.Anything).
					Return(dhcp, (*data.Netboot)(nil), (*data.Power)(nil), nil)
			},
			expectedError: os.ErrNotExist,
			rootDirectory: tempDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockBackend := &mockBackend{}
			if tt.setupBackend != nil {
				tt.setupBackend(mockBackend)
			}

			// Create handler
			handler := &Handler{
				ctx:           context.Background(),
				RootDirectory: tt.rootDirectory,
				Patch:         "default patch",
				Log:           logr.Discard(),
				backend:       mockBackend,
			}

			// Setup transfer
			var err error
			if tt.isWrite {
				wt := &mockIncomingTransfer{
					WriterTo:   &bytes.Buffer{},
					remoteAddr: net.UDPAddr{IP: net.ParseIP("192.168.1.100")},
				}
				err = handler.HandleWrite(tt.fullfilepath, wt)
			} else {
				rf := &mockOutgoingTransfer{
					ReaderFrom: newMockReaderFrom(),
					remoteAddr: net.UDPAddr{IP: net.ParseIP("192.168.1.100")},
				}
				err = handler.HandleRead(tt.fullfilepath, rf)
				if tt.expectedData != nil {
					if mrf, ok := rf.ReaderFrom.(*mockReaderFrom); ok {
						assert.Equal(t, tt.expectedData, mrf.Bytes())
					}
				}
			}

			// Verify results
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockBackend.AssertExpectations(t)
		})
	}
}

func TestHandler_HandleRead_iPXEBinary(t *testing.T) {
	// Mock binary.Files for testing
	originalFiles := binary.Files
	binary.Files = map[string][]byte{
		"test.kpxe": []byte("test ipxe binary"),
	}
	defer func() { binary.Files = originalFiles }()

	mockBackend := &mockBackend{}
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	dhcp := &data.DHCP{MACAddress: mac}
	netboot := &data.Netboot{IPXEScript: "custom script content"}
	mockBackend.On("GetByIP", mock.Anything, mock.Anything).
		Return(dhcp, netboot, (*data.Power)(nil), nil)

	handler := &Handler{
		ctx:           context.Background(),
		RootDirectory: t.TempDir(),
		Patch:         "default patch",
		Log:           logr.Discard(),
		backend:       mockBackend,
	}

	rf := &mockOutgoingTransfer{
		ReaderFrom: newMockReaderFrom(),
		remoteAddr: net.UDPAddr{IP: net.ParseIP("192.168.1.100")},
	}

	err := handler.HandleRead("test.kpxe", rf)
	assert.NoError(t, err)

	mockBackend.AssertExpectations(t)
}

func TestHandler_HandleRead_GetRemoteIPError(t *testing.T) {
	handler := &Handler{
		ctx:           context.Background(),
		RootDirectory: t.TempDir(),
		Log:           logr.Discard(),
		backend:       &mockBackend{},
	}

	transfer := &mockOutgoingTransfer{} // No remote address

	err := handler.HandleRead("test.txt", transfer)
	assert.Error(t, err) // Expect an error because getRemoteIP will fail
}

// mockBackend needs to implement the full backend.BackendReader interface.
func (m *mockBackend) Create(ctx context.Context, dhcp *data.DHCP, netboot *data.Netboot) error {
	args := m.Called(ctx, dhcp, netboot)
	return args.Error(0)
}

func (m *mockBackend) Delete(ctx context.Context, mac net.HardwareAddr) error {
	args := m.Called(ctx, mac)
	return args.Error(0)
}

func (m *mockBackend) GetByMacAndIP(
	ctx context.Context,
	mac net.HardwareAddr,
	ip net.IP,
) (*data.DHCP, *data.Netboot, error) {
	args := m.Called(ctx, mac, ip)
	return args.Get(0).(*data.DHCP), args.Get(1).(*data.Netboot), args.Error(2)
}

func (m *mockBackend) ListAll(ctx context.Context) ([]*data.DHCP, []*data.Netboot, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*data.DHCP), args.Get(1).([]*data.Netboot), args.Error(2)
}

func (m *mockBackend) Update(ctx context.Context, dhcp *data.DHCP, netboot *data.Netboot) error {
	args := m.Called(ctx, dhcp, netboot)
	return args.Error(0)
}

var _ backend.BackendReader = &mockBackend{}
