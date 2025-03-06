package efi

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// BootEntry represents an EFI boot entry

type BootEntry struct {
	Attr       uint32
	Title      UCS16String
	DevicePath DevicePath
	OptData    []byte
}

func (b *BootEntry) UnmarshalJSON(data []byte) error {
	return b.Parse(data)
}

func (b *BootEntry) GetMacAddr() string {
	return ""
}

// NewBootEntry creates a new BootEntry
func NewBootEntry(data []byte, attr uint32, title *UCS16String, devicePath *DevicePath, optData *[]byte) *BootEntry {
	entry := &BootEntry{
		Attr:       0,
		Title:      UCS16String{},
		DevicePath: DevicePath{},
		OptData:    nil,
	}

	if data != nil {
		entry.Parse(data)
	}
	if attr != 0 {
		entry.Attr = attr
	}
	if title != nil {
		if title.data != nil {
			entry.Title = *title
		}
	}
	if devicePath != nil {
		if devicePath.elems != nil {
			entry.DevicePath = *devicePath
		}
	}
	if optData != nil {
		entry.OptData = *optData
	}

	return entry
}

// Parse parses binary data into a BootEntry
func (entry *BootEntry) Parse(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("data too short to parse boot entry")
	}

	// Read the attribute and path size
	entry.Attr = binary.LittleEndian.Uint32(data[0:4])
	pathSize := binary.LittleEndian.Uint16(data[4:6])

	// Parse the title
	title := FromUCS16(data, 6)
	titleSize := title.Size()
	entry.Title = *title

	// Extract and parse the device path
	pathOffset := 6 + titleSize
	if pathOffset+int(pathSize) > len(data) {
		return fmt.Errorf("data too short for device path")
	}
	entry.DevicePath = *NewDevicePath(data[pathOffset : pathOffset+int(pathSize)])

	// Extract optional data if present
	optOffset := pathOffset + int(pathSize)
	if optOffset < len(data) {
		entry.OptData = data[optOffset:]
	}

	return nil
}

// Bytes returns the binary representation of the BootEntry
func (entry *BootEntry) Bytes() []byte {
	var buf bytes.Buffer

	// Write attributes and path size
	pathData := entry.DevicePath.Bytes()
	pathSize := uint16(len(pathData))

	binary.Write(&buf, binary.LittleEndian, entry.Attr)
	binary.Write(&buf, binary.LittleEndian, pathSize)

	// Write title
	buf.Write(entry.Title.Bytes())

	// Write device path
	buf.Write(pathData)

	// Write optional data if present
	if entry.OptData != nil {
		buf.Write(entry.OptData)
	}

	return buf.Bytes()
}

// String returns a string representation of the BootEntry
func (entry *BootEntry) String() string {
	result := fmt.Sprintf("title=\"%s\" devpath=%s", entry.Title.String(), entry.DevicePath.String())
	if entry.OptData != nil {
		result += fmt.Sprintf(" optdata=%s", hex.EncodeToString(entry.OptData))
	}
	return result
}

// GetActiveStatus returns whether the boot entry is active
func (entry *BootEntry) GetActiveStatus() bool {
	return (entry.Attr & LOAD_OPTION_ACTIVE) != 0
}

// SetActiveStatus sets or clears the active flag
func (entry *BootEntry) SetActiveStatus(active bool) {
	if active {
		entry.Attr |= LOAD_OPTION_ACTIVE
	} else {
		entry.Attr &= ^uint32(LOAD_OPTION_ACTIVE)
	}
}

// GetHiddenStatus returns whether the boot entry is hidden
func (entry *BootEntry) GetHiddenStatus() bool {
	return (entry.Attr & LOAD_OPTION_HIDDEN) != 0
}

// SetHiddenStatus sets or clears the hidden flag
func (entry *BootEntry) SetHiddenStatus(hidden bool) {
	if hidden {
		entry.Attr |= LOAD_OPTION_HIDDEN
	} else {
		entry.Attr &= ^uint32(LOAD_OPTION_HIDDEN)
	}
}

// GetCategory returns the category of the boot entry
func (entry *BootEntry) GetCategory() uint32 {
	return entry.Attr & LOAD_OPTION_CATEGORY
}

// SetCategory sets the category of the boot entry
func (entry *BootEntry) SetCategory(category uint32) {
	// Clear existing category bits
	entry.Attr &= ^uint32(LOAD_OPTION_CATEGORY)
	// Set new category bits
	entry.Attr |= (category & LOAD_OPTION_CATEGORY)
}
