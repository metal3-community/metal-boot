package efi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBootEntryFromBytes(t *testing.T) {
	// Simple test for now to ensure the method doesn't panic
	entry, err := ParseBootEntry([]byte{0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00})
	assert.Error(t, err)
	assert.Nil(t, entry)
}
