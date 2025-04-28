package efi_test

import (
	"testing"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/stretchr/testify/assert"
)

func TestGUIDParsing(t *testing.T) {
	testCases := []struct {
		name      string
		guidStr   string
		expectErr bool
		expected  [16]byte
	}{
		{
			name:      "Valid GUID",
			guidStr:   "12345678-1234-5678-1234-567812345678",
			expectErr: false,
			expected:  [16]byte{0x78, 0x56, 0x34, 0x12, 0x34, 0x12, 0x78, 0x56, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78},
		},
		{
			name:      "EFI Global Variable GUID",
			guidStr:   efi.EfiGlobalVariableGUID,
			expectErr: false,
			expected:  [16]byte{0x61, 0xdf, 0xe4, 0x8b, 0xd4, 0x11, 0x11, 0x42, 0x9d, 0xcd, 0x00, 0xd0, 0x80, 0x84, 0x2c, 0xc4},
		},
		{
			name:      "Invalid Format",
			guidStr:   "not-a-guid",
			expectErr: true,
		},
		{
			name:      "Too Short",
			guidStr:   "12345678-1234-5678-1234-56781234567",
			expectErr: true,
		},
		{
			name:      "Too Long",
			guidStr:   "12345678-1234-5678-1234-5678123456789",
			expectErr: true,
		},
		{
			name:      "Missing Hyphens",
			guidStr:   "1234567812345678123456781234567812345678",
			expectErr: true,
		},
		{
			name:      "Invalid Characters",
			guidStr:   "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			guidBytes, err := efi.GUIDStringToBytes(tc.guidStr)
			
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, guidBytes)
				
				// Test round trip
				guidStrBack, err := efi.GUIDBytesToString(guidBytes)
				assert.NoError(t, err)
				assert.Equal(t, tc.guidStr, guidStrBack)
			}
		})
	}
}

func TestGUIDBytesToString(t *testing.T) {
	testCases := []struct {
		name      string
		guidBytes [16]byte
		expectErr bool
		expected  string
	}{
		{
			name:      "Valid GUID Bytes",
			guidBytes: [16]byte{0x78, 0x56, 0x34, 0x12, 0x34, 0x12, 0x78, 0x56, 0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78},
			expectErr: false,
			expected:  "12345678-1234-5678-1234-567812345678",
		},
		{
			name:      "EFI Global Variable GUID Bytes",
			guidBytes: [16]byte{0x61, 0xdf, 0xe4, 0x8b, 0xd4, 0x11, 0x11, 0x42, 0x9d, 0xcd, 0x00, 0xd0, 0x80, 0x84, 0x2c, 0xc4},
			expectErr: false,
			expected:  efi.EfiGlobalVariableGUID,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			guidStr, err := efi.GUIDBytesToString(tc.guidBytes)
			
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, guidStr)
			}
		})
	}
}

func TestCompareGUID(t *testing.T) {
	guid1 := efi.EfiGlobalVariableGUID
	guid2 := "8be4df61-11d4-4211-9dcd-00d0802cc412" // Different ordering
	guid3 := "8be4df61-11d4-4211-9dcd-00d0802cc411" // Off by one digit
	
	// Convert to bytes
	guid1Bytes, err := efi.GUIDStringToBytes(guid1)
	assert.NoError(t, err)
	
	guid2Bytes, err := efi.GUIDStringToBytes(guid2)
	assert.NoError(t, err)
	
	guid3Bytes, err := efi.GUIDStringToBytes(guid3)
	assert.NoError(t, err)
	
	// Compare
	assert.True(t, efi.CompareGUID(guid1Bytes, guid1Bytes))
	assert.False(t, efi.CompareGUID(guid1Bytes, guid2Bytes))
	assert.False(t, efi.CompareGUID(guid1Bytes, guid3Bytes))
	assert.False(t, efi.CompareGUID(guid2Bytes, guid3Bytes))
}

func TestIsKnownGUID(t *testing.T) {
	// Test known GUIDs
	assert.True(t, efi.IsKnownGUID(efi.EfiGlobalVariableGUID))
	assert.True(t, efi.IsKnownGUID(efi.EfiImageSecurityDatabaseGUID))
	assert.True(t, efi.IsKnownGUID(efi.EfiSecureBootEnableDisableGUID))
	
	// Test unknown GUID
	assert.False(t, efi.IsKnownGUID("12345678-1234-5678-1234-567812345678"))
}

func TestFormatGUID(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Already Formatted GUID",
			input:    "12345678-1234-5678-1234-567812345678",
			expected: "12345678-1234-5678-1234-567812345678",
		},
		{
			name:     "GUID Without Hyphens",
			input:    "12345678123456781234567812345678",
			expected: "12345678-1234-5678-1234-567812345678",
		},
		{
			name:     "Invalid GUID",
			input:    "not-a-guid",
			expected: "not-a-guid", // Should return the original string
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := efi.FormatGUID(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
