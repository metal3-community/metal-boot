package efi

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"
)

// EFI variable attributes
const (
	EfiVariableNonVolatile                       uint32 = 0x00000001
	EfiVariableBootserviceAccess                 uint32 = 0x00000002
	EfiVariableRuntimeAccess                     uint32 = 0x00000004
	EfiVariableHardwareErrorRecord               uint32 = 0x00000008
	EfiVariableAuthenticatedWriteAccess          uint32 = 0x00000010 // deprecated
	EfiVariableTimeBasedAuthenticatedWriteAccess uint32 = 0x00000020
	EfiVariableAppendWrite                       uint32 = 0x00000040

	EfiVariableDefault = EfiVariableNonVolatile | EfiVariableBootserviceAccess
)

// Default configurations for well-known EFI variables
var efivarDefaults = map[string]struct {
	Attr uint32
	Guid string
}{
	"SecureBoot": {
		Attr: EfiVariableBootserviceAccess | EfiVariableRuntimeAccess,
		Guid: EFI_GLOBAL_VARIABLE,
	},
	// "SecureBootEnable": {
	// 	Attr: EfiVariableNonVolatile | EfiVariableBootserviceAccess,
	// 	Guid: guids.EfiSecureBootEnableDisable,
	// },
	// "CustomMode": {
	// 	Attr: EfiVariableNonVolatile | EfiVariableBootserviceAccess,
	// 	Guid: guids.EfiCustomModeEnable,
	// },
	// "PK": {
	// 	Attr: EfiVariableNonVolatile | EfiVariableBootserviceAccess |
	// 		EfiVariableRuntimeAccess | EfiVariableTimeBasedAuthenticatedWriteAccess,
	// 	Guid: guids.EfiGlobalVariable,
	// },
}

var bootDefaults = struct {
	Attr uint32
	Guid string
}{
	Attr: EfiVariableNonVolatile | EfiVariableBootserviceAccess | EfiVariableRuntimeAccess,
	Guid: EFI_GLOBAL_VARIABLE,
}

var (
	boolNames  = []string{"SecureBootEnable", "CustomMode"}
	asciiNames = []string{"Lang", "PlatformLang", "SbatLevel"}
	blistNames = []string{"BootOrder", "BootNext"}
	dpathNames = []string{"ConIn", "ConOut", "ErrOut"}
)

// EfiVar represents an EFI variable
type EfiVar struct {
	Name  *UCS16String
	Guid  GUID
	Attr  uint32
	Data  []byte
	Count int
	Time  *time.Time
	PkIdx int
}

// NewEfiVar creates a new EFI variable
func NewEfiVar(name any, guid *string, attr uint32, data []byte, count int) (*EfiVar, error) {
	v := &EfiVar{
		Data:  data,
		Count: count,
	}

	// Handle name
	switch n := name.(type) {
	case *UCS16String:
		v.Name = n
	case string:
		v.Name = FromString(string(n))
	case []byte:
		v.Name = FromString(string(n))
	default:
		return nil, errors.New("invalid name type")
	}

	// Parse GUID
	if guid != nil {
		var err error
		v.Guid, err = ParseGUID(*guid)
		if err != nil {
			return nil, err
		}
	}

	// Set attribute
	v.Attr = attr

	// Apply defaults
	nameStr := v.Name.String()
	defaults, ok := efivarDefaults[nameStr]
	if !ok && strings.HasPrefix(nameStr, "Boot") {
		v.Guid = EFI_GLOBAL_VARIABLE_GUID
		if v.Attr == 0 {
			v.Attr = bootDefaults.Attr
		}
	} else if ok {
		if v.Guid.String() == "" {
			v.Guid = EFI_GLOBAL_VARIABLE_GUID
		}
		if v.Attr == 0 {
			v.Attr = defaults.Attr
		}
	} else if v.Guid.String() == "" {
		v.Guid = EFI_GLOBAL_VARIABLE_GUID
		if v.Attr == 0 {
			v.Attr = EfiVariableDefault
		}
	}

	return v, nil
}

// ParseTime parses an EFI_TIME structure
func (v *EfiVar) ParseTime(data []byte, offset int) error {
	if len(data) < offset+16 {
		return errors.New("data too short for EFI_TIME")
	}

	year := binary.LittleEndian.Uint16(data[offset:])
	month := data[offset+2]
	day := data[offset+3]
	hour := data[offset+4]
	minute := data[offset+5]
	second := data[offset+6]
	// Skip pad byte at offset+7
	ns := binary.LittleEndian.Uint32(data[offset+8:])
	// Skip timezone, daylight savings and pad

	if year != 0 {
		t := time.Date(int(year), time.Month(month), int(day),
			int(hour), int(minute), int(second),
			int(ns)/1000, time.UTC)
		v.Time = &t
	} else {
		v.Time = nil
	}

	return nil
}

// BytesTime generates an EFI_TIME structure
func (v *EfiVar) BytesTime() []byte {
	if v.Time == nil {
		return bytes.Repeat([]byte{0}, 16)
	}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint16(v.Time.Year()))
	buf.WriteByte(byte(v.Time.Month()))
	buf.WriteByte(byte(v.Time.Day()))
	buf.WriteByte(byte(v.Time.Hour()))
	buf.WriteByte(byte(v.Time.Minute()))
	buf.WriteByte(byte(v.Time.Second()))
	buf.WriteByte(0) // pad
	binary.Write(buf, binary.LittleEndian, uint32(v.Time.Nanosecond()/1000))
	binary.Write(buf, binary.LittleEndian, int16(0)) // timezone
	buf.WriteByte(0)                                 // daylight
	buf.WriteByte(0)                                 // pad

	return buf.Bytes()
}

// updateTime updates the time field if needed
func (v *EfiVar) updateTime(ts *time.Time) {
	if v.Attr&EfiVariableTimeBasedAuthenticatedWriteAccess == 0 {
		return
	}

	now := time.Now().UTC()
	if ts == nil {
		ts = &now
	}

	if v.Time == nil || v.Time.Before(*ts) {
		v.Time = ts
	}
}

// SetBool sets a boolean value
func (v *EfiVar) SetBool(value bool) {
	if value {
		v.Data = []byte{1}
	} else {
		v.Data = []byte{0}
	}
	v.updateTime(nil)
}

func (v *EfiVar) SetString(value string) {
	buf := []byte(value)
	// Ensure the string is null-terminated
	if len(buf) == 0 || buf[len(buf)-1] != 0 {
		buf = append(buf, 0)
	}
	v.Data = buf
	v.updateTime(nil)
}

func (v *EfiVar) SetHexString(value string) error {
	data, err := hex.DecodeString(value)
	if err != nil {
		return err
	}
	v.Data = data
	return nil
}

// SetUint32 sets a 32-bit unsigned integer value
func (v *EfiVar) SetUint32(value uint32) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, value)
	v.Data = buf
	v.updateTime(nil)
}

func (v *EfiVar) GetUint32() (uint32, error) {
	if len(v.Data) < 4 {
		return 0, errors.New("data too short for uint32")
	}
	return binary.LittleEndian.Uint32(v.Data), nil
}

func (v *EfiVar) GetBootEntry() (*BootEntry, error) {
	return NewBootEntry(v.Data, v.Attr, nil, nil, nil), nil
}

// SetBootEntry sets a boot entry
func (v *EfiVar) SetBootEntry(attr uint32, title string, path string, optdata []byte) error {
	t := NewUCS16String(title)

	p := NewDevicePath([]byte{})

	if strings.Contains(path, "(") {
		var err error
		p, err = ParseDevicePathFromString(path)
		if err != nil {
			return fmt.Errorf("failed to parse device path from string: %s", path)
		}
	} else {
		p = NewDevicePath([]byte(path))
	}

	entry := NewBootEntry(nil, attr, t, p, &optdata)

	v.Data = entry.Bytes()
	v.updateTime(nil)
	return nil
}

func (v *EfiVar) GetBootNext() (uint16, error) {
	if len(v.Data) < 2 {
		return 0, errors.New("data too short for BootNext")
	}
	return binary.LittleEndian.Uint16(v.Data), nil
}

// SetBootNext sets the BootNext variable
func (v *EfiVar) SetBootNext(index uint16) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, index)
	v.Data = buf
	v.updateTime(nil)
}

// GetBootOrder retrieves the BootOrder variable
func (v *EfiVar) GetBootOrder() ([]uint16, error) {
	var order []uint16
	for pos := range len(v.Data) / 2 {
		nr := binary.LittleEndian.Uint16(v.Data[pos*2:])
		order = append(order, nr)
	}
	return order, nil
}

// SetBootOrder sets the BootOrder variable
func (v *EfiVar) SetBootOrder(order []uint16) {
	buf := new(bytes.Buffer)
	for _, item := range order {
		binary.Write(buf, binary.LittleEndian, item)
	}
	v.Data = buf.Bytes()
	v.updateTime(nil)
}

// AppendBootOrder appends to the BootOrder variable
func (v *EfiVar) AppendBootOrder(index uint16) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, index)
	v.Data = append(v.Data, buf...)
	v.updateTime(nil)
}

// SetFromFile sets the variable data from a file
func (v *EfiVar) SetFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	v.Data = data
	v.updateTime(nil)
	return nil
}

// FmtBool formats a boolean variable
func (v *EfiVar) FmtBool() string {
	if len(v.Data) == 0 {
		return "bool: invalid"
	}

	if v.Data[0] != 0 {
		return "bool: ON"
	}
	return "bool: off"
}

// FmtAscii formats an ASCII variable
func (v *EfiVar) FmtAscii() string {
	str := strings.ReplaceAll(strings.TrimRight(string(v.Data), "\x00"), "\n", "\\n")
	return fmt.Sprintf("ascii: \"%s\"", str)
}

// FmtBootEntry formats a boot entry variable
func (v *EfiVar) FmtBootEntry() (string, error) {
	entry := NewBootEntry(v.Data, 0, nil, nil, nil)
	return fmt.Sprintf("boot entry: %s", entry), nil
}

// FmtBootList formats a boot list variable
func (v *EfiVar) FmtBootList() string {
	var bootlist []string
	for pos := 0; pos < len(v.Data)/2; pos++ {
		nr := binary.LittleEndian.Uint16(v.Data[pos*2:])
		bootlist = append(bootlist, fmt.Sprintf("%04x", nr))
	}

	desc := strings.Join(bootlist, ", ")
	return fmt.Sprintf("boot order: %s", desc)
}

// FmtDevPath formats a device path variable
func (v *EfiVar) FmtDevPath() (string, error) {
	path := NewDevicePath(v.Data)
	return fmt.Sprintf("devpath: %s", path), nil
}

// FmtData formats the variable data based on its name and content
func (v *EfiVar) FmtData() (string, error) {
	name := v.Name.String()

	// Handle boolean variables
	if slices.Contains(boolNames, name) {
		return v.FmtBool(), nil
	}

	// Handle ASCII variables
	if slices.Contains(asciiNames, name) {
		return v.FmtAscii(), nil
	}

	// Handle boot list variables
	if slices.Contains(blistNames, name) {
		return v.FmtBootList(), nil
	}

	// Handle device path variables
	if slices.Contains(dpathNames, name) {
		return v.FmtDevPath()
	}

	// Handle boot entry variables
	if strings.HasPrefix(name, "Boot0") {
		return v.FmtBootEntry()
	}

	// Handle simple numeric values
	if len(v.Data) == 1 || len(v.Data) == 2 || len(v.Data) == 4 || len(v.Data) == 8 {
		typeNames := map[int]string{
			1: "byte",
			2: "word",
			4: "dword",
			8: "qword",
		}

		typeName := typeNames[len(v.Data)]
		d := make([]byte, len(v.Data))
		for i := 0; i < len(v.Data); i++ {
			d[i] = v.Data[len(v.Data)-i-1]
		}

		return fmt.Sprintf("%s: 0x%s", typeName, hex.EncodeToString(d)), nil
	}

	return "", nil
}

// String returns a string representation of the EFI variable
func (v *EfiVar) String() string {
	name := v.Name.String()
	guid := v.Guid.String()
	attr := fmt.Sprintf("0x%08x", v.Attr)
	data, _ := v.FmtData()

	if v.Time != nil {
		return fmt.Sprintf("name=%s guid=%s attr=%s data=%s time=%s",
			name, guid, attr, data, v.Time.Format(time.RFC3339))
	}

	return fmt.Sprintf("name=%s guid=%s attr=%s data=%s",
		name, guid, attr, data)
}
