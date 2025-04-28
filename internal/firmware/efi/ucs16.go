package efi

import (
	"fmt"
	"unicode/utf16"
)

// UCS16String represents an EFI UCS-16 string
type UCS16String struct {
	data []byte
}

// NewUCS16String creates a new StringUCS16, optionally initialized with a string
func NewUCS16String(string ...string) *UCS16String {
	s := &UCS16String{
		data: []byte{},
	}
	if len(string) > 0 && string[0] != "" {
		s.ParseStr(string[0])
	}
	return s
}

// ParseBin sets StringUCS16 from bytes data, reads to terminating 0
func (s *UCS16String) ParseBin(data []byte, offset int) {
	s.data = []byte{}
	pos := offset

	for {
		if pos+2 > len(data) || (data[pos] == 0 && data[pos+1] == 0) {
			break
		}
		s.data = append(s.data, data[pos], data[pos+1])
		pos += 2
	}
}

// ParseStr sets StringUCS16 from Go string
func (s *UCS16String) ParseStr(str string) {
	// Convert to UTF-16 code points
	runes := []rune(str)
	utf16Chars := utf16.Encode(runes)

	// Convert UTF-16 code points to bytes (little endian)
	s.data = make([]byte, len(utf16Chars)*2)
	for i, char := range utf16Chars {
		s.data[i*2] = byte(char)
		s.data[i*2+1] = byte(char >> 8)
	}
}

// Bytes returns bytes representing StringUCS16, with terminating 0
func (s *UCS16String) Bytes() []byte {
	return append(s.data, 0, 0)
}

// Size returns the number of bytes returned by Bytes()
func (s *UCS16String) Size() int {
	return len(s.data) + 2
}

// String converts StringUCS16 to a Go string
func (s *UCS16String) String() string {
	// Check for empty data
	if len(s.data) == 0 {
		return ""
	}

	// Convert bytes to UTF-16 code points (little endian)
	utf16Chars := make([]uint16, len(s.data)/2)
	for i := 0; i < len(s.data)/2; i++ {
		utf16Chars[i] = uint16(s.data[i*2]) | (uint16(s.data[i*2+1]) << 8)
	}

	// Convert UTF-16 code points to runes
	runes := utf16.Decode(utf16Chars)

	// Convert runes to string
	return string(runes)
}

// GoString implements the fmt.GoStringer interface
func (s *UCS16String) GoString() string {
	return fmt.Sprintf("UCS16String(%q)", s.String())
}

// FromUCS16 converts UCS-16 bytes to StringUCS16
func FromUCS16(data []byte, offset ...int) *UCS16String {
	off := 0
	if len(offset) > 0 {
		off = offset[0]
	}
	obj := NewUCS16String()
	obj.ParseBin(data, off)
	return obj
}

// FromString converts Go string to StringUCS16
func FromString(str string) *UCS16String {
	return NewUCS16String(str)
}

// ToUCS16 is a convenience function that converts a string to UCS16String
func ToUCS16(str string) *UCS16String {
	return FromString(str)
}

// Ucs16ToString converts a UCS-16 string to a regular Go string
func Ucs16ToString(s *UCS16String) string {
	if s == nil {
		return ""
	}
	return s.String()
}
