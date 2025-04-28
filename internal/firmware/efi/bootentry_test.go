package efi_test

import (
	"bytes"
	"testing"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootEntryFromBytes(t *testing.T) {
	// Create sample boot entry data
	// Format: [Attrs uint32][DevPathLen uint16][Description UCS16 string][DevPath bytes]
	var buf bytes.Buffer
	
	// Active and boot next = true
	attrs := uint32(efi.LOAD_OPTION_ACTIVE | efi.LOAD_OPTION_CATEGORY_BOOT)
	
	// Device path
	devPath := []byte{
		0x01, 0x02, 0x01, 0x00, // HardDrive
		0x01, 0x01, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, // PCI Root
		0x03, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // End path
	}
	devPathLen := uint16(len(devPath))
	
	// Description as UCS-16 string "Boot Entry Test"
	desc := []byte{
		0x42, 0x00, 0x6F, 0x00, 0x6F, 0x00, 0x74, 0x00, 0x20, 0x00, // "Boot "
		0x45, 0x00, 0x6E, 0x00, 0x74, 0x00, 0x72, 0x00, 0x79, 0x00, // "Entry "
		0x20, 0x00, 0x54, 0x00, 0x65, 0x00, 0x73, 0x00, 0x74, 0x00, // "Test"
		0x00, 0x00, // Null terminator
	}
	
	// Optional data
	optData := []byte{0x01, 0x02, 0x03, 0x04}
	
	// Construct the binary representation
	binary.Write(&buf, binary.LittleEndian, attrs)
	binary.Write(&buf, binary.LittleEndian, devPathLen)
	buf.Write(desc)
	buf.Write(devPath)
	buf.Write(optData)
	
	// Parse the boot entry
	entry, err := efi.ParseBootEntry(buf.Bytes())
	require.NoError(t, err)
	
	// Verify the parsed data
	assert.Equal(t, "Boot Entry Test", entry.Description)
	assert.Equal(t, devPath, entry.DevicePath)
	assert.Equal(t, optData, entry.OptionalData)
	assert.True(t, entry.Active)
	assert.Equal(t, uint32(efi.LOAD_OPTION_CATEGORY_BOOT), entry.Category)
}

func TestBootEntryToBytes(t *testing.T) {
	// Create a boot entry
	entry := &efi.BootEntry{
		Description:  "UEFI Boot Test",
		DevicePath:   []byte{0x01, 0x02, 0x03, 0x04},
		OptionalData: []byte{0xAA, 0xBB, 0xCC, 0xDD},
		Active:       true,
		Category:     efi.LOAD_OPTION_CATEGORY_BOOT,
	}
	
	// Convert to bytes
	data, err := entry.ToBytes()
	require.NoError(t, err)
	
	// Parse back to verify round trip
	parsedEntry, err := efi.ParseBootEntry(data)
	require.NoError(t, err)
	
	// Verify fields are preserved
	assert.Equal(t, entry.Description, parsedEntry.Description)
	assert.Equal(t, entry.DevicePath, parsedEntry.DevicePath)
	assert.Equal(t, entry.OptionalData, parsedEntry.OptionalData)
	assert.Equal(t, entry.Active, parsedEntry.Active)
	assert.Equal(t, entry.Category, parsedEntry.Category)
}

func TestBootEntryActiveFlag(t *testing.T) {
	// Test with active flag on
	activeEntry := &efi.BootEntry{
		Description: "Active Entry",
		DevicePath:  []byte{0x01, 0x02, 0x03, 0x04},
		Active:      true,
	}
	
	data, err := activeEntry.ToBytes()
	require.NoError(t, err)
	
	parsedActive, err := efi.ParseBootEntry(data)
	require.NoError(t, err)
	assert.True(t, parsedActive.Active)
	
	// Test with active flag off
	inactiveEntry := &efi.BootEntry{
		Description: "Inactive Entry",
		DevicePath:  []byte{0x01, 0x02, 0x03, 0x04},
		Active:      false,
	}
	
	data, err = inactiveEntry.ToBytes()
	require.NoError(t, err)
	
	parsedInactive, err := efi.ParseBootEntry(data)
	require.NoError(t, err)
	assert.False(t, parsedInactive.Active)
}

func TestBootEntryCategories(t *testing.T) {
	testCases := []struct {
		name     string
		category uint32
	}{
		{"BootCategory", efi.LOAD_OPTION_CATEGORY_BOOT},
		{"AppCategory", efi.LOAD_OPTION_CATEGORY_APP},
		{"CustomCategory", 0x10000000}, // Custom category
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := &efi.BootEntry{
				Description: tc.name,
				DevicePath:  []byte{0x01, 0x02, 0x03, 0x04},
				Category:    tc.category,
			}
			
			data, err := entry.ToBytes()
			require.NoError(t, err)
			
			parsed, err := efi.ParseBootEntry(data)
			require.NoError(t, err)
			assert.Equal(t, tc.category, parsed.Category)
		})
	}
}

func TestBootEntryEmptyFields(t *testing.T) {
	// Test with empty description
	emptyDescEntry := &efi.BootEntry{
		Description: "",
		DevicePath:  []byte{0x01, 0x02, 0x03, 0x04},
	}
	
	data, err := emptyDescEntry.ToBytes()
	require.NoError(t, err)
	
	parsed, err := efi.ParseBootEntry(data)
	require.NoError(t, err)
	assert.Equal(t, "", parsed.Description)
	
	// Test with empty device path
	emptyDevPathEntry := &efi.BootEntry{
		Description: "Empty DevPath",
		DevicePath:  []byte{},
	}
	
	data, err = emptyDevPathEntry.ToBytes()
	require.NoError(t, err)
	
	parsed, err = efi.ParseBootEntry(data)
	require.NoError(t, err)
	assert.Empty(t, parsed.DevicePath)
	
	// Test with empty optional data
	emptyOptDataEntry := &efi.BootEntry{
		Description:  "Empty OptData",
		DevicePath:   []byte{0x01, 0x02, 0x03, 0x04},
		OptionalData: []byte{},
	}
	
	data, err = emptyOptDataEntry.ToBytes()
	require.NoError(t, err)
	
	parsed, err = efi.ParseBootEntry(data)
	require.NoError(t, err)
	assert.Empty(t, parsed.OptionalData)
}

func TestBootEntryDevicePath(t *testing.T) {
	// Create various device paths and test parsing
	testCases := []struct {
		name     string
		devPath  []byte
		expected string
	}{
		{
			name: "PCI Root",
			devPath: []byte{
				0x01, 0x01, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, // PCI Root
				0x7F, 0xFF, 0x04, 0x00, // End path
			},
			expected: "PciRoot(0)",
		},
		{
			name: "PCI Device",
			devPath: []byte{
				0x01, 0x01, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, // PCI Root
				0x01, 0x01, 0x06, 0x00, 0x01, 0x02, 0x03, 0x04, // PCI Device
				0x7F, 0xFF, 0x04, 0x00, // End path
			},
			expected: "PciRoot(0)/Pci(1,2)",
		},
		{
			name: "SATA Device",
			devPath: []byte{
				0x01, 0x01, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, // PCI Root
				0x01, 0x01, 0x06, 0x00, 0x01, 0x02, 0x03, 0x04, // PCI Device
				0x01, 0x02, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, // SATA
				0x7F, 0xFF, 0x04, 0x00, // End path
			},
			expected: "PciRoot(0)/Pci(1,2)/Sata(0)",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a boot entry with this device path
			entry := &efi.BootEntry{
				Description: tc.name,
				DevicePath:  tc.devPath,
			}
			
			// Get device path as string
			devPathStr, err := entry.GetDevicePathString()
			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, devPathStr)
			}
			
			// Convert to bytes and parse back
			data, err := entry.ToBytes()
			require.NoError(t, err)
			
			parsed, err := efi.ParseBootEntry(data)
			require.NoError(t, err)
			
			// Verify device path is preserved
			assert.Equal(t, tc.devPath, parsed.DevicePath)
			
			// Verify string representation is correct
			parsedPathStr, err := parsed.GetDevicePathString()
			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, parsedPathStr)
			}
		})
	}
}

func TestBootEntryErrors(t *testing.T) {
	// Test parsing invalid data (too short)
	_, err := efi.ParseBootEntry([]byte{0x01, 0x02})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "boot entry data too short")
	
	// Test parsing with invalid device path length
	var buf bytes.Buffer
	
	// Set attributes
	attrs := uint32(efi.LOAD_OPTION_ACTIVE)
	binary.Write(&buf, binary.LittleEndian, attrs)
	
	// Set invalid device path length (larger than available data)
	devPathLen := uint16(1000)
	binary.Write(&buf, binary.LittleEndian, devPathLen)
	
	// Add some description and a short device path
	buf.Write([]byte{0x41, 0x00, 0x00, 0x00}) // "A" in UCS-16
	buf.Write([]byte{0x01, 0x02, 0x03, 0x04}) // Too short for the claimed length
	
	_, err = efi.ParseBootEntry(buf.Bytes())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid device path length")
}

// missing import for binary package
import "encoding/binary"
