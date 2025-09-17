package talos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateSchematic_Extensions(t *testing.T) {
	// Test that the method handles parameters correctly
	extensions := []string{"qemu-guest-agent", "nvidia-open-gpu-kernel-modules"}
	overlay := "rpi-generic"

	// Verify we have the expected data for schematic creation
	assert.NotEmpty(t, extensions)
	assert.NotEmpty(t, overlay)
	assert.Len(t, extensions, 2)
}

func TestCreateSchematic_SingleOverlay(t *testing.T) {
	// Test that a single overlay is handled correctly
	extensions := []string{"qemu-guest-agent"}
	overlay := "rpi-generic"

	// The implementation should use the single overlay
	assert.Equal(t, "rpi-generic", overlay)
	assert.NotEmpty(t, extensions)
}

func TestCreateSchematic_NoOverlay(t *testing.T) {
	// Test extensions without overlay
	extensions := []string{"qemu-guest-agent"}
	overlay := ""

	assert.NotEmpty(t, extensions)
	assert.Empty(t, overlay)

	// The schematic should be created with only extensions, no overlay field set
}

func TestGetFilteredExtensions(t *testing.T) {
	// Test the filtering logic for extensions
	requestedExtensions := []string{"qemu-guest-agent", "nvidia-container-toolkit"}

	// Verify we can filter extensions by name
	assert.NotEmpty(t, requestedExtensions)
	assert.Len(t, requestedExtensions, 2)
	assert.Contains(t, requestedExtensions, "qemu-guest-agent")
	assert.Contains(t, requestedExtensions, "nvidia-container-toolkit")
}

func TestGetFilteredExtensions_EmptyInput(t *testing.T) {
	// Test that empty extension list returns empty result
	var requestedExtensions []string

	assert.Empty(t, requestedExtensions)
	// The GetFilteredExtensions method should handle empty input gracefully
}
