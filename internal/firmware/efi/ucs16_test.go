package efi_test

import (
	"testing"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/stretchr/testify/assert"
)

func TestUCS16StringConversion(t *testing.T) {
	testCases := []struct {
		name    string
		utf8Str string
		ucs16   []byte
	}{
		{
			name:    "Simple ASCII",
			utf8Str: "Hello",
			ucs16:   []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00, 0x00, 0x00},
		},
		{
			name:    "With Spaces",
			utf8Str: "Hello World",
			ucs16:   []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00, 0x20, 0x00, 0x57, 0x00, 0x6F, 0x00, 0x72, 0x00, 0x6C, 0x00, 0x64, 0x00, 0x00, 0x00},
		},
		{
			name:    "With Unicode",
			utf8Str: "你好",
			ucs16:   []byte{0x60, 0x4f, 0x7d, 0x59, 0x00, 0x00},
		},
		{
			name:    "Empty String",
			utf8Str: "",
			ucs16:   []byte{0x00, 0x00},
		},
		{
			name:    "With Symbols",
			utf8Str: "Test-123!@#",
			ucs16:   []byte{0x54, 0x00, 0x65, 0x00, 0x73, 0x00, 0x74, 0x00, 0x2D, 0x00, 0x31, 0x00, 0x32, 0x00, 0x33, 0x00, 0x21, 0x00, 0x40, 0x00, 0x23, 0x00, 0x00, 0x00},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// UTF-8 to UCS-16
			result := efi.UTF8ToUCS16(tc.utf8Str)
			assert.Equal(t, tc.ucs16, result)

			// UCS-16 to UTF-8
			back := efi.UCS16ToUTF8(tc.ucs16)
			assert.Equal(t, tc.utf8Str, back)
		})
	}
}

func TestUCS16StringWithLength(t *testing.T) {
	testCases := []struct {
		name         string
		utf8Str      string
		expectedSize int
	}{
		{
			name:         "Simple ASCII",
			utf8Str:      "Hello",
			expectedSize: 12, // 5 chars * 2 bytes + 2 bytes null terminator
		},
		{
			name:         "With Spaces",
			utf8Str:      "Hello World",
			expectedSize: 24, // 11 chars * 2 bytes + 2 bytes null terminator
		},
		{
			name:         "With Unicode",
			utf8Str:      "你好",
			expectedSize: 6, // 2 chars * 2 bytes + 2 bytes null terminator
		},
		{
			name:         "Empty String",
			utf8Str:      "",
			expectedSize: 2, // Just the null terminator
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get the UCS-16 bytes
			ucs16Bytes := efi.UTF8ToUCS16(tc.utf8Str)

			// Check the length
			assert.Equal(t, tc.expectedSize, len(ucs16Bytes))

			// Ensure the string is null terminated
			assert.Equal(t, byte(0), ucs16Bytes[len(ucs16Bytes)-2])
			assert.Equal(t, byte(0), ucs16Bytes[len(ucs16Bytes)-1])

			// Ensure round trip conversion works
			utf8Back := efi.UCS16ToUTF8(ucs16Bytes)
			assert.Equal(t, tc.utf8Str, utf8Back)
		})
	}
}

func TestUCS16StringWithNullTermination(t *testing.T) {
	// Test with embedded nulls
	utf8WithNull := "Hello\x00World"
	ucs16WithNull := efi.UTF8ToUCS16(utf8WithNull)

	// The conversion should handle embedded nulls correctly
	// The string should be terminated at the first null in UTF-8
	assert.Equal(t, "Hello", efi.UCS16ToUTF8(ucs16WithNull))

	// Test with manually constructed UCS-16 with embedded nulls
	manualUcs16 := []byte{
		0x48, 0x00, // H
		0x65, 0x00, // e
		0x6C, 0x00, // l
		0x6C, 0x00, // l
		0x6F, 0x00, // o
		0x00, 0x00, // null terminator
		0x57, 0x00, // W (should be ignored)
		0x6F, 0x00, // o (should be ignored)
		0x72, 0x00, // r (should be ignored)
		0x6C, 0x00, // l (should be ignored)
		0x64, 0x00, // d (should be ignored)
	}

	assert.Equal(t, "Hello", efi.UCS16ToUTF8(manualUcs16))
}

func TestUCS16StringEdgeCases(t *testing.T) {
	// Test with very long string
	longString := string(make([]rune, 1000))
	for i := range longString {
		longString = longString[:i] + "A" + longString[i+1:]
	}

	ucs16Long := efi.UTF8ToUCS16(longString)
	assert.Equal(t, 2002, len(ucs16Long)) // 1000 chars * 2 bytes + 2 bytes null terminator
	assert.Equal(t, longString, efi.UCS16ToUTF8(ucs16Long))

	// Test with odd length UCS-16 (invalid, but should be handled gracefully)
	oddUcs16 := []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00, 0x00} // Missing last byte of null terminator
	assert.Equal(t, "Hello", efi.UCS16ToUTF8(oddUcs16))                                  // Should handle it gracefully

	// Test with nil or empty byte array
	assert.Equal(t, "", efi.UCS16ToUTF8(nil))
	assert.Equal(t, "", efi.UCS16ToUTF8([]byte{}))

	// Test conversion of empty string
	assert.Equal(t, []byte{0x00, 0x00}, efi.UTF8ToUCS16(""))
}

func TestFindUCS16NullTerminator(t *testing.T) {
	testCases := []struct {
		name     string
		ucs16    []byte
		expected int
	}{
		{
			name:     "Simple String",
			ucs16:    []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00, 0x00, 0x00},
			expected: 10, // Index of the first byte of the null terminator
		},
		{
			name:     "Empty String",
			ucs16:    []byte{0x00, 0x00},
			expected: 0, // Index of the first byte of the null terminator
		},
		{
			name:     "No Null Terminator",
			ucs16:    []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00},
			expected: 10, // Should return the length of the array
		},
		{
			name:     "Embedded Null",
			ucs16:    []byte{0x48, 0x00, 0x65, 0x00, 0x00, 0x00, 0x6C, 0x00, 0x6F, 0x00},
			expected: 4, // Index of the first byte of the first null terminator
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			index := efi.FindUCS16NullTerminator(tc.ucs16)
			assert.Equal(t, tc.expected, index)
		})
	}
}
