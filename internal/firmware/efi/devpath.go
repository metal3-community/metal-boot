package efi

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"
)

const (
	DevTypeMessage = 0x03
	DevTypeMedia   = 0x04
	DevTypeFile    = 0x05
)

//
// The following helper code emulates the functionality of the Python modules:
//   - guids (provides GUID parsing and formatting)
//   - ucs16 (provides UCS-16 conversions)
//

// Guid represents a GUID with its little-endian bytes.
type Guid struct {
	BytesLe []byte
}

func (g Guid) String() string {
	// If length is not 16 bytes, return an error string.
	if len(g.BytesLe) != 16 {
		return "InvalidGUID"
	}
	// Reorder the first 3 fields from little-endian to big-endian for display.
	data1 := binary.LittleEndian.Uint32(g.BytesLe[0:4])
	data2 := binary.LittleEndian.Uint16(g.BytesLe[4:6])
	data3 := binary.LittleEndian.Uint16(g.BytesLe[6:8])
	return fmt.Sprintf("%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		data1, data2, data3,
		g.BytesLe[8], g.BytesLe[9],
		g.BytesLe[10], g.BytesLe[11],
		g.BytesLe[12], g.BytesLe[13], g.BytesLe[14], g.BytesLe[15])
}

// guidsParseStr parses a GUID string and returns a Guid struct with little-endian bytes.
func guidsParseStr(guidStr string) (Guid, error) {
	cleaned := strings.ReplaceAll(guidStr, "-", "")
	if len(cleaned) != 32 {
		return Guid{}, errors.New("invalid GUID format")
	}
	bytesArr, err := hex.DecodeString(cleaned)
	if err != nil {
		return Guid{}, err
	}
	le := make([]byte, 16)
	// Convert the first 4 bytes.
	binary.LittleEndian.PutUint32(le[0:4], binary.BigEndian.Uint32(bytesArr[0:4]))
	// Next 2 bytes.
	binary.LittleEndian.PutUint16(le[4:6], binary.BigEndian.Uint16(bytesArr[4:6]))
	// Next 2 bytes.
	binary.LittleEndian.PutUint16(le[6:8], binary.BigEndian.Uint16(bytesArr[6:8]))
	// The remaining 8 bytes are copied as is.
	copy(le[8:16], bytesArr[8:16])
	return Guid{BytesLe: le}, nil
}

// guidsParseBin parses a GUID from binary data starting at the specified offset.
func guidsParseBin(data []byte, offset int) (Guid, error) {
	if len(data) < offset+16 {
		return Guid{}, errors.New("not enough data for GUID")
	}
	guidBytes := data[offset : offset+16]
	le := make([]byte, 16)
	copy(le, guidBytes)
	return Guid{BytesLe: le}, nil
}

// ucs16FromString converts a string to a UCS-16 little-endian byte slice.
func ucs16FromString(s string) []byte {
	codepoints := utf16.Encode([]rune(s))
	buf := new(bytes.Buffer)
	for _, cp := range codepoints {
		binary.Write(buf, binary.LittleEndian, cp)
	}
	return buf.Bytes()
}

// ucs16FromUcs16 converts a UCS-16 little-endian byte slice starting at offset to a string.
// It stops conversion at a zero terminator if found.
func ucs16FromUcs16(data []byte, offset int) string {
	if offset >= len(data) {
		return ""
	}
	n := (len(data) - offset) / 2
	codepoints := make([]uint16, n)
	for i := range n {
		codepoints[i] = binary.LittleEndian.Uint16(data[offset+2*i : offset+2*i+2])
		if codepoints[i] == 0 {
			codepoints = codepoints[:i]
			break
		}
	}
	runes := utf16.Decode(codepoints)
	return string(runes)
}

// DevicePathElem is a class representing an efi device path element.
type DevicePathElem struct {
	Devtype uint8
	Subtype uint8
	Data    []byte
}

// NewDevicePathElem creates a new DevicePathElem.
// If data is provided, it unpacks devtype, subtype, and the size from the data.
func NewDevicePathElem(data []byte) *DevicePathElem {
	dpe := &DevicePathElem{
		Devtype: 0x7f,
		Subtype: 0xff,
		Data:    []byte{},
	}
	if len(data) >= 4 {
		dpe.Devtype = data[0]
		dpe.Subtype = data[1]
		size := binary.LittleEndian.Uint16(data[2:4])
		if int(size) > 4 && int(size) <= len(data) {
			dpe.Data = data[4:int(size)]
		}
	}
	return dpe
}

func (dpe *DevicePathElem) set_mac() {
	dpe.Devtype = DevTypeMessage // msg
	dpe.Subtype = 0x0b           // mac
	dpe.Data = make([]byte, 6)   // use dhcp
}

func (dpe *DevicePathElem) set_ipv4() {
	dpe.Devtype = DevTypeMessage // msg
	dpe.Subtype = 0x0c           // ipv4
	dpe.Data = make([]byte, 23)  // use dhcp
}

func (dpe *DevicePathElem) set_ipv6() {
	dpe.Devtype = DevTypeMessage // msg
	dpe.Subtype = 0x0d           // ipv6
	dpe.Data = make([]byte, 39)  // use dhcp
}

func (dpe *DevicePathElem) set_iscsi(target string) {
	dpe.Devtype = DevTypeMessage // msg
	dpe.Subtype = 0x13           // iscsi
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(0)) // reserved
	buf.WriteString(target)
	dpe.Data = buf.Bytes()
}

func (dpe *DevicePathElem) set_sata(port uint16) {
	dpe.Devtype = DevTypeMessage // msg
	dpe.Subtype = 0x12           // sata
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, port)
	dpe.Data = buf.Bytes()
}

func (dpe *DevicePathElem) set_usb(port uint8) {
	dpe.Devtype = DevTypeMessage // msg
	dpe.Subtype = 0x05           // usb
	dpe.Data = []byte{port, 0}   // port, interface (not used)
}

func (dpe *DevicePathElem) set_uri(uri string) {
	dpe.Devtype = DevTypeMessage // msg
	dpe.Subtype = 0x18           // uri
	dpe.Data = []byte(uri)
}

func (dpe *DevicePathElem) set_filepath(filepath string) {
	dpe.Devtype = DevTypeMedia // media
	dpe.Subtype = 0x04         // filepath
	dpe.Data = ucs16FromString(filepath)
}

func (dpe *DevicePathElem) set_fvname(guid string) {
	dpe.Devtype = DevTypeMedia // media
	dpe.Subtype = 0x07         // fv name
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint8(0x02)) // version
	binary.Write(&buf, binary.LittleEndian, uint8(0x02)) // revision
	guidObj, err := guidsParseStr(guid)
	if err == nil {
		buf.Write(guidObj.BytesLe)
	}
	dpe.Data = buf.Bytes()
}

func (dpe *DevicePathElem) set_fvfilename(guid string) {
	dpe.Devtype = DevTypeMedia // media
	dpe.Subtype = 0x06         // fv filename
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint8(0x02)) // version
	binary.Write(&buf, binary.LittleEndian, uint8(0x02)) // revision
	guidObj, err := guidsParseStr(guid)
	if err == nil {
		buf.Write(guidObj.BytesLe)
	}
	dpe.Data = buf.Bytes()
}

func (dpe *DevicePathElem) set_gpt(pnr uint32, poff uint64, plen uint64, guid string) {
	dpe.Devtype = DevTypeMedia // media
	dpe.Subtype = 0x01         // hard drive
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, pnr)
	binary.Write(&buf, binary.LittleEndian, poff)
	binary.Write(&buf, binary.LittleEndian, plen)
	guidObj, err := guidsParseStr(guid)
	if err == nil {
		buf.Write(guidObj.BytesLe)
	}
	binary.Write(&buf, binary.LittleEndian, uint8(0x02))
	binary.Write(&buf, binary.LittleEndian, uint8(0x02))
	dpe.Data = buf.Bytes()
}

func (dpe *DevicePathElem) fmt_hw() string {
	if dpe.Subtype == 0x01 && len(dpe.Data) >= 2 {
		funcVal := dpe.Data[0]
		devVal := dpe.Data[1]
		return fmt.Sprintf("PCI(dev=%02x:%x)", devVal, funcVal)
	}
	if dpe.Subtype == 0x04 {
		guidObj, err := guidsParseBin(dpe.Data, 0)
		if err == nil {
			return fmt.Sprintf("VendorHW(%s)", guidObj.String())
		}
		return fmt.Sprintf("VendorHW(ERROR:%v)", err)
	}
	return fmt.Sprintf("HW(subtype=0x%x)", dpe.Subtype)
}

func (dpe *DevicePathElem) fmt_acpi() string {
	if dpe.Subtype == 0x01 && len(dpe.Data) >= 8 {
		hid := binary.LittleEndian.Uint32(dpe.Data[0:4])
		uid := binary.LittleEndian.Uint32(dpe.Data[4:8])
		if hid == 0xa0341d0 {
			return "PciRoot()"
		}
		return fmt.Sprintf("ACPI(hid=0x%x,uid=0x%x)", hid, uid)
	}
	if dpe.Subtype == 0x03 && len(dpe.Data) >= 4 {
		adr := binary.LittleEndian.Uint32(dpe.Data[0:4])
		return fmt.Sprintf("GOP(adr=0x%x)", adr)
	}
	return fmt.Sprintf("ACPI(subtype=0x%x)", dpe.Subtype)
}

func (dpe *DevicePathElem) fmt_msg() string {
	if dpe.Subtype == 0x02 {
		if len(dpe.Data) >= 4 {
			pun := binary.LittleEndian.Uint16(dpe.Data[0:2])
			lun := binary.LittleEndian.Uint16(dpe.Data[2:4])
			return fmt.Sprintf("SCSI(pun=%d,lun=%d)", pun, lun)
		}
	}
	if dpe.Subtype == 0x05 {
		if len(dpe.Data) >= 2 {
			port := dpe.Data[0]
			intf := dpe.Data[1]
			_ = intf
			return fmt.Sprintf("USB(port=%d)", port)
		}
	}
	if dpe.Subtype == 0x0b {
		return "MAC()"
	}
	if dpe.Subtype == 0x0c {
		return "IPv4()"
	}
	if dpe.Subtype == 0x0d {
		return "IPv6()"
	}
	if dpe.Subtype == 0x12 {
		if len(dpe.Data) >= 6 {
			port := binary.LittleEndian.Uint16(dpe.Data[0:2])
			return fmt.Sprintf("SATA(port=%d)", port)
		}
	}
	if dpe.Subtype == 0x13 {
		if len(dpe.Data) >= 14 {
			target := string(dpe.Data[14:])
			return fmt.Sprintf("ISCSI(%s)", target)
		}
	}
	if dpe.Subtype == 0x18 {
		return fmt.Sprintf("URI(%s)", string(dpe.Data))
	}
	if dpe.Subtype == 0x1f {
		return "DNS()"
	}
	return fmt.Sprintf("Msg(subtype=0x%x)", dpe.Subtype)
}

func (dpe *DevicePathElem) fmt_media() string {
	if dpe.Subtype == 0x01 && len(dpe.Data) >= 20 {
		pnr := binary.LittleEndian.Uint32(dpe.Data[0:4])
		return fmt.Sprintf("Partition(nr=%d)", pnr)
	}
	if dpe.Subtype == 0x04 {
		path := ucs16FromUcs16(dpe.Data, 0)
		return fmt.Sprintf("FilePath(%s)", path)
	}
	if dpe.Subtype == 0x06 {
		guidObj, err := guidsParseBin(dpe.Data, 0)
		if err == nil {
			return fmt.Sprintf("FvFileName(%s)", guidObj.String())
		}
		return fmt.Sprintf("FvFileName(ERROR:%v)", err)
	}
	if dpe.Subtype == 0x07 {
		guidObj, err := guidsParseBin(dpe.Data, 0)
		if err == nil {
			return fmt.Sprintf("FvName(%s)", guidObj.String())
		}
		return fmt.Sprintf("FvName(ERROR:%v)", err)
	}
	return fmt.Sprintf("Media(subtype=0x%x)", dpe.Subtype)
}

func (dpe *DevicePathElem) size() int {
	return len(dpe.Data) + 4
}

func (dpe *DevicePathElem) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, dpe.Devtype)
	binary.Write(buf, binary.LittleEndian, dpe.Subtype)
	binary.Write(buf, binary.LittleEndian, uint16(dpe.size()))
	buf.Write(dpe.Data)
	return buf.Bytes()
}

func (dpe *DevicePathElem) String() string {
	switch dpe.Devtype {
	case 0x01:
		return dpe.fmt_hw()
	case 0x02:
		return dpe.fmt_acpi()
	case 0x03:
		return dpe.fmt_msg()
	case 0x04:
		return dpe.fmt_media()
	}
	return fmt.Sprintf("Unknown(type=0x%x,subtype=0x%x)", dpe.Devtype, dpe.Subtype)
}

func (dpe *DevicePathElem) Equal(other *DevicePathElem) bool {
	if dpe.Devtype != other.Devtype {
		return false
	}
	if dpe.Subtype != other.Subtype {
		return false
	}
	if dpe.Devtype == 0x04 && dpe.Subtype == 0x04 {
		p1 := strings.ToLower(ucs16FromUcs16(dpe.Data, 0))
		p2 := strings.ToLower(ucs16FromUcs16(other.Data, 0))
		return p1 == p2
	}
	return bytes.Equal(dpe.Data, other.Data)
}

// DevicePath represents an efi device path.
type DevicePath struct {
	elems []*DevicePathElem
}

func (dp *DevicePath) VendorHW(guid GUID) *DevicePath {
	elem := NewDevicePathElem(nil)
	elem.Devtype = DevTypeMedia // media
	elem.Subtype = 0x04         // vendor hardware
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint8(0x02)) // version
	binary.Write(&buf, binary.LittleEndian, uint8(0x02)) // revision
	binary.Write(&buf, binary.LittleEndian, guid.BytesLE())
	elem.Data = buf.Bytes()
	dp.elems = append(dp.elems, elem)
	return dp
}

func (dp *DevicePath) Mac() *DevicePath {
	elem := NewDevicePathElem(nil)
	elem.set_mac()
	dp.elems = append(dp.elems, elem)
	return dp
}

func (dp *DevicePath) IPv4() *DevicePath {
	elem := NewDevicePathElem(nil)
	elem.set_ipv4()
	dp.elems = append(dp.elems, elem)
	return dp
}

func (dp *DevicePath) IPv6() *DevicePath {
	elem := NewDevicePathElem(nil)
	elem.set_ipv6()
	dp.elems = append(dp.elems, elem)
	return dp
}

func (dp *DevicePath) ISCSI(target string) *DevicePath {
	elem := NewDevicePathElem(nil)
	elem.set_iscsi(target)
	dp.elems = append(dp.elems, elem)
	return dp
}

func (dp *DevicePath) SATA(port uint16) *DevicePath {
	elem := NewDevicePathElem(nil)
	elem.set_sata(port)
	dp.elems = append(dp.elems, elem)
	return dp
}

func (dp *DevicePath) USB(port uint8) *DevicePath {
	elem := NewDevicePathElem(nil)
	elem.set_usb(port)
	dp.elems = append(dp.elems, elem)
	return dp
}

func (dp *DevicePath) FvName(guid string) *DevicePath {
	elem := NewDevicePathElem(nil)
	elem.set_fvname(guid)
	dp.elems = append(dp.elems, elem)
	return dp
}

func (dp *DevicePath) FVFileName(guid string) *DevicePath {
	elem := NewDevicePathElem(nil)
	elem.set_fvfilename(guid)
	dp.elems = append(dp.elems, elem)
	return dp
}

func (dp *DevicePath) FilePath(filepath string) *DevicePath {
	elem := NewDevicePathElem(nil)
	elem.set_filepath(filepath)
	dp.elems = append(dp.elems, elem)
	return dp
}

func (dp *DevicePath) GptPartition(pnr uint32, poff uint64, plen uint64, guid string) *DevicePath {
	elem := NewDevicePathElem(nil)
	elem.set_gpt(pnr, poff, plen, guid)
	dp.elems = append(dp.elems, elem)
	return dp
}

func (dp *DevicePath) Append(elem *DevicePathElem) *DevicePath {
	dp.elems = append(dp.elems, elem)
	return dp
}

// NewDevicePath creates a new DevicePath from data.
// It parses each DevicePathElem until a terminating element is found.
func NewDevicePath(data []byte) *DevicePath {
	dp := &DevicePath{elems: []*DevicePathElem{}}
	if data != nil {
		pos := 0
		for pos < len(data) {
			elem := NewDevicePathElem(data[pos:])
			if elem.Devtype == 0x7f {
				break
			}
			dp.elems = append(dp.elems, elem)
			pos += elem.size()
		}
	}
	return dp
}

// DevicePathUri creates a DevicePath with a URI element.
func DevicePathUri(uri string) *DevicePath {
	dp := &DevicePath{elems: []*DevicePathElem{}}
	elem := NewDevicePathElem(nil)
	elem.set_uri(uri)
	dp.elems = append(dp.elems, elem)
	return dp
}

// DevicePathFilepath creates a DevicePath with a filepath element.
func DevicePathFilepath(filepath string) *DevicePath {
	dp := &DevicePath{elems: []*DevicePathElem{}}
	elem := NewDevicePathElem(nil)
	elem.set_filepath(filepath)
	dp.elems = append(dp.elems, elem)
	return dp
}

func (dp *DevicePath) Bytes() []byte {
	var blob bytes.Buffer
	for _, elem := range dp.elems {
		blob.Write(elem.Bytes())
	}
	// Append terminating DevicePathElem
	term := NewDevicePathElem(nil)
	blob.Write(term.Bytes())
	return blob.Bytes()
}

func (dp *DevicePath) String() string {
	strs := []string{}
	for _, elem := range dp.elems {
		strs = append(strs, elem.String())
	}
	return strings.Join(strs, "/")
}

func (dp *DevicePath) Equal(other *DevicePath) bool {
	if len(dp.elems) != len(other.elems) {
		return false
	}
	for i := 0; i < len(dp.elems); i++ {
		if !dp.elems[i].Equal(other.elems[i]) {
			return false
		}
	}
	return true
}
