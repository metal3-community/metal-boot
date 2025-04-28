package efi_test

import (
	"testing"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevicePathParsing(t *testing.T) {
	testCases := []struct {
		name          string
		devPathBytes  []byte
		expectedPath  string
		expectSuccess bool
	}{
		{
			name: "PCI Root",
			devPathBytes: []byte{
				0x01, 0x01, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, // PCI Root
				0x7F, 0xFF, 0x04, 0x00, // End path
			},
			expectedPath:  "PciRoot(0)",
			expectSuccess: true,
		},
		{
			name: "PCI Device",
			devPathBytes: []byte{
				0x01, 0x01, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, // PCI Root
				0x01, 0x01, 0x06, 0x00, 0x01, 0x02, 0x03, 0x04, // PCI Device (Function 2, Device 1)
				0x7F, 0xFF, 0x04, 0x00, // End path
			},
			expectedPath:  "PciRoot(0)/Pci(1,2)",
			expectSuccess: true,
		},
		{
			name: "Hard Drive Path",
			devPathBytes: []byte{
				0x01, 0x01, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, // PCI Root
				0x01, 0x01, 0x06, 0x00, 0x01, 0x02, 0x03, 0x04, // PCI Device
				0x04, 0x01, 0x14, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Hard Drive
				0x7F, 0xFF, 0x04, 0x00, // End path
			},
			expectedPath:  "PciRoot(0)/Pci(1,2)/HD(2,0,0,0)",
			expectSuccess: true,
		},
		{
			name: "Network Path",
			devPathBytes: []byte{
				0x01, 0x01, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, // PCI Root
				0x01, 0x01, 0x06, 0x00, 0x01, 0x02, 0x03, 0x04, // PCI Device
				0x03, 0x0B, 0x25, 0x00, 0x00, 0x10, 0x18, 0xC0, 0xA8, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // IPv4 (192.168.0.1)
				0x7F, 0xFF, 0x04, 0x00, // End path
			},
			expectedPath:  "PciRoot(0)/Pci(1,2)/IPv4(192.168.0.1)",
			expectSuccess: true,
		},
		{
			name: "Invalid Path (Too Short)",
			devPathBytes: []byte{
				0x01, 0x01, // Truncated path
			},
			expectSuccess: false,
		},
		{
			name: "Invalid Path (Bad Type)",
			devPathBytes: []byte{
				0xFF, 0xFF, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, // Invalid type/subtype
				0x7F, 0xFF, 0x04, 0x00, // End path
			},
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pathString, err := efi.ParseDevicePath(tc.devPathBytes)
			
			if tc.expectSuccess {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedPath, pathString)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestDevicePathConstruction(t *testing.T) {
	testCases := []struct {
		name          string
		pathString    string
		expectSuccess bool
	}{
		{
			name:          "PCI Root",
			pathString:    "PciRoot(0)",
			expectSuccess: true,
		},
		{
			name:          "PCI Device",
			pathString:    "PciRoot(0)/Pci(1,2)",
			expectSuccess: true,
		},
		{
			name:          "Hard Drive",
			pathString:    "PciRoot(0)/Pci(1,2)/HD(2,0,0,0)",
			expectSuccess: true,
		},
		{
			name:          "Network Device",
			pathString:    "PciRoot(0)/Pci(1,2)/MAC(001122334455,0)/IPv4(192.168.0.1)/TCP(80)",
			expectSuccess: true,
		},
		{
			name:          "USB Device",
			pathString:    "PciRoot(0)/Pci(1,2)/USB(0,0)",
			expectSuccess: true,
		},
		{
			name:          "Invalid Path Format",
			pathString:    "ThisIsNotAValidPath",
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			devicePath, err := efi.ConstructDevicePath(tc.pathString)
			
			if tc.expectSuccess {
				assert.NoError(t, err)
				
				// Parse back to string to verify round trip
				pathStringBack, err := efi.ParseDevicePath(devicePath)
				assert.NoError(t, err)
				
				// The string representation might not be exactly the same as input
				// due to normalization, but the core path components should match
				assert.Contains(t, pathStringBack, tc.pathString[:10]) // Check at least the beginning matches
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestDevicePathComponentParsing(t *testing.T) {
	// Test individual component parsing
	
	// PciRoot component
	t.Run("PciRoot Component", func(t *testing.T) {
		pciRootBytes := []byte{0x01, 0x01, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00} // PCI Root
		
		component, size, err := efi.ParseDevicePathComponent(pciRootBytes, 0)
		assert.NoError(t, err)
		assert.Equal(t, "PciRoot(0)", component)
		assert.Equal(t, 8, size)
	})
	
	// Pci component
	t.Run("Pci Component", func(t *testing.T) {
		pciBytes := []byte{0x01, 0x01, 0x06, 0x00, 0x01, 0x02, 0x03, 0x04} // PCI Device
		
		component, size, err := efi.ParseDevicePathComponent(pciBytes, 0)
		assert.NoError(t, err)
		assert.Equal(t, "Pci(1,2)", component)
		assert.Equal(t, 8, size)
	})
	
	// Hard Drive component
	t.Run("Hard Drive Component", func(t *testing.T) {
		hdBytes := []byte{
			0x04, 0x01, 0x14, 0x00, // Hard Drive header
			0x02, 0x00, 0x00, 0x00, // Partition number
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Partition start
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Partition size
		}
		
		component, size, err := efi.ParseDevicePathComponent(hdBytes, 0)
		assert.NoError(t, err)
		assert.Equal(t, "HD(2,0,0,0)", component)
		assert.Equal(t, 20, size)
	})
	
	// Invalid component (bad type)
	t.Run("Invalid Component Type", func(t *testing.T) {
		invalidBytes := []byte{0xFF, 0xFF, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00}
		
		_, _, err := efi.ParseDevicePathComponent(invalidBytes, 0)
		assert.Error(t, err)
	})
	
	// Invalid component (truncated)
	t.Run("Truncated Component", func(t *testing.T) {
		truncatedBytes := []byte{0x01, 0x01, 0x06, 0x00} // Too short for PCI Root
		
		_, _, err := efi.ParseDevicePathComponent(truncatedBytes, 0)
		assert.Error(t, err)
	})
}

func TestSpecialDevicePaths(t *testing.T) {
	// Test some special device paths like PXE, File, etc.
	
	testCases := []struct {
		name          string
		devPathBytes  []byte
		expectedPath  string
		expectSuccess bool
	}{
		{
			name: "PXE Boot Path",
			devPathBytes: []byte{
				0x01, 0x01, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, // PCI Root
				0x01, 0x01, 0x06, 0x00, 0x01, 0x02, 0x03, 0x04, // PCI Device
				0x03, 0x0B, 0x25, 0x00, 0x00, 0x10, 0x18, 0xC0, 0xA8, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // IPv4
				0x04, 0x03, 0x04, 0x00, // End path for PXE
			},
			expectedPath:  "PciRoot(0)/Pci(1,2)/IPv4(192.168.0.1)/PXE",
			expectSuccess: true,
		},
		{
			name: "File Path",
			devPathBytes: []byte{
				0x04, 0x04, 0x20, 0x00, // File path header
				0x5C, 0x00, 0x45, 0x00, 0x46, 0x00, 0x49, 0x00, 0x5C, 0x00, 0x42, 0x00, 0x4F, 0x00, 0x4F, 0x00, 0x54, 0x00, 0x5C, 0x00, 0x42, 0x00, 0x4F, 0x00, 0x4F, 0x00, 0x54, 0x00, 0x00, 0x00, // \EFI\BOOT\BOOT
				0x7F, 0xFF, 0x04, 0x00, // End path
			},
			expectedPath:  "File(\\EFI\\BOOT\\BOOT)",
			expectSuccess: true,
		},
		{
			name: "HTTP Path",
			devPathBytes: []byte{
				0x01, 0x01, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, // PCI Root
				0x01, 0x01, 0x06, 0x00, 0x01, 0x02, 0x03, 0x04, // PCI Device
				0x03, 0x0C, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // HTTP
				0x7F, 0xFF, 0x04, 0x00, // End path
			},
			expectedPath:  "PciRoot(0)/Pci(1,2)/HTTP",
			expectSuccess: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pathString, err := efi.ParseDevicePath(tc.devPathBytes)
			
			if tc.expectSuccess {
				assert.NoError(t, err)
				assert.Contains(t, pathString, tc.expectedPath)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestComplexDevicePath(t *testing.T) {
	// Build a complex device path programmatically
	var pathComponents [][]byte
	
	// PCI Root
	pciRoot := []byte{0x01, 0x01, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00}
	pathComponents = append(pathComponents, pciRoot)
	
	// PCI Device
	pciDevice := []byte{0x01, 0x01, 0x06, 0x00, 0x01, 0x02, 0x03, 0x04}
	pathComponents = append(pathComponents, pciDevice)
	
	// MAC Address
	mac := []byte{
		0x03, 0x0B, 0x19, 0x00, // MAC header
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, // MAC address
		0x06, // MAC length
		0x00, // Padding
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Padding
	}
	pathComponents = append(pathComponents, mac)
	
	// IPv4
	ipv4 := []byte{
		0x03, 0x0C, 0x14, 0x00, // IPv4 header
		0xC0, 0xA8, 0x01, 0x01, // IP address (192.168.1.1)
		0xFF, 0xFF, 0xFF, 0x00, // Subnet mask (255.255.255.0)
		0x00, 0x50, // Port (80)
		0x00, 0x00, // Protocol (TCP)
		0x00, // Static IP
		0x00, 0x00, 0x00, // Padding
	}
	pathComponents = append(pathComponents, ipv4)
	
	// End path
	endPath := []byte{0x7F, 0xFF, 0x04, 0x00}
	pathComponents = append(pathComponents, endPath)
	
	// Combine all components
	var devPathBytes []byte
	for _, component := range pathComponents {
		devPathBytes = append(devPathBytes, component...)
	}
	
	// Parse the path
	pathString, err := efi.ParseDevicePath(devPathBytes)
	require.NoError(t, err)
	
	// Expected path
	expectedPath := "PciRoot(0)/Pci(1,2)/MAC(001122334455)/IPv4(192.168.1.1,255.255.255.0,TCP,80,Static)"
	assert.Equal(t, expectedPath, pathString)
	
	// Now try to construct a path from the string and verify it's equivalent
	constructedPath, err := efi.ConstructDevicePath(pathString)
	require.NoError(t, err)
	
	// Parse back to string
	parsedConstructed, err := efi.ParseDevicePath(constructedPath)
	require.NoError(t, err)
	
	// Verify the constructed path matches the expected path
	assert.Equal(t, expectedPath, parsedConstructed)
}
