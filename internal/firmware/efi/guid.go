package efi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

func GuidName(guid GUID) string {
	name, ok := GuidNameTable[guid.String()]
	if !ok {
		return guid.String()
	}
	return name
}

var GuidNameTable = map[string]string{
	// firmware volumes
	Ffs:          "Ffs",
	NvData:       "NvData",
	AuthVars:     "AuthVars",
	LzmaCompress: "LzmaCompress",
	ResetVector:  "ResetVector",

	"9e21fd93-9c72-4c15-8c4b-e77f1db2d792": "FvMainCompact",
	"df1ccef6-f301-4a63-9661-fc6030dcc880": "SecMain",

	"48db5e17-707c-472d-91cd-1613e7ef51b0": "OvmfMainFv",
	OvmfPeiFv:                              "OvmfPeiFv",
	OvmfDxeFv:                              "OvmfDxeFv",
	"763bed0d-de9f-48f5-81f1-3e90e1b1a015": "OvmfSecFv",

	// variable types
	EfiGlobalVariable:          "EfiGlobalVariable",
	EfiImageSecurityDatabase:   "EfiImageSecurityDatabase",
	EfiSecureBootEnableDisable: "EfiSecureBootEnableDisable",
	EfiCustomModeEnable:        "EfiCustomModeEnable",

	"eb704011-1402-11d3-8e77-00a0c969723b": "MtcVendor",
	"4c19049f-4137-4dd3-9c10-8b97a83ffdfa": "EfiMemoryTypeInformation",
	"4b47d616-a8d6-4552-9d44-ccad2e0f4cf9": "IScsiConfig",
	"d9bee56e-75dc-49d9-b4d7-b534210f637a": "EfiCertDb",
	"fd2340d0-3dab-4349-a6c7-3b4f12b48eae": "EfiTlsCaCertificate",

	// protocols (also used for variables)
	"59324945-ec44-4c0d-b1cd-9db139df070c": "EfiIScsiInitiatorNameProtocol",
	EfiDhcp6ServiceBindingProtocol:         "EfiDhcp6ServiceBindingProtocol",
	"5b446ed1-e30b-4faa-871a-3654eca36080": "EfiIp4Config2Protocol",
	EfiIp6ConfigProtocol:                   "EfiIp6ConfigProtocol",

	// signature list types
	EfiCertX509:   "EfiCertX509",
	EfiCertSha256: "EfiCertSha256",
	EfiCertPkcs7:  "EfiCertPkcs7",

	// signature owner
	MicrosoftVendor:       "MicrosoftVendor",
	OvmfEnrollDefaultKeys: "OvmfEnrollDefaultKeys",
	Shim:                  "Shim",
	LoaderInfo:            "LoaderInfo",

	// ovmf metadata
	OvmfGuidList:          "OvmfGuidList",
	OvmfSevMetadataOffset: "OvmfSevMetadataOffset",
	TdxMetadataOffset:     "TdxMetadataOffset",
	SevHashTableBlock:     "SevHashTableBlock",
	SevSecretBlock:        "SevSecretBlock",
	SevProcessorReset:     "SevProcessorReset",

	// capsule
	FwMgrCapsule:  "FwMgrCapsule",
	SignedCapsule: "SignedCapsule",

	// misc
	"00000000-0000-0000-0000-000000000000": "Zero",
	NotValid:                               "NotValid",
}

// GUID represents an EFI GUID (Globally Unique Identifier).
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

func (g GUID) BytesLE() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, g.Data1)
	binary.Write(buf, binary.LittleEndian, g.Data2)
	binary.Write(buf, binary.LittleEndian, g.Data3)
	buf.Write(g.Data4[:])
	return buf.Bytes()
}

// ParseGUID parses a GUID from its string representation.
func ParseGUID(s string) (GUID, error) {
	var guid GUID

	// Remove braces and whitespace
	s = strings.ReplaceAll(s, "{", "")
	s = strings.ReplaceAll(s, "}", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")

	// Check length
	if len(s) != 32 {
		return guid, fmt.Errorf("invalid GUID string length: %d", len(s))
	}

	// Parse the four parts
	var err error

	data1, err := strconv.ParseUint(s[0:8], 16, 32)
	if err != nil {
		return guid, fmt.Errorf("failed to parse Data1: %v", err)
	}
	guid.Data1 = uint32(data1)

	data2, err := strconv.ParseUint(s[8:12], 16, 16)
	if err != nil {
		return guid, fmt.Errorf("failed to parse Data2: %v", err)
	}
	guid.Data2 = uint16(data2)

	data3, err := strconv.ParseUint(s[12:16], 16, 16)
	if err != nil {
		return guid, fmt.Errorf("failed to parse Data3: %v", err)
	}
	guid.Data3 = uint16(data3)

	for i := 0; i < 8; i++ {
		val, err := strconv.ParseUint(s[16+i*2:18+i*2], 16, 8)
		if err != nil {
			return guid, fmt.Errorf("failed to parse Data4[%d]: %v", i, err)
		}
		guid.Data4[i] = byte(val)
	}

	return guid, nil
}

// ParseGuid parses a GUID string into a Guid struct.
func ParseGuid(s string) GUID {
	guid, err := ParseGUID(s)
	if err != nil {
		return GUID{}
	}
	return guid
}

// NewGUID creates a new GUID from its components.
func NewGUID(data1 uint32, data2, data3 uint16, data4 [8]byte) GUID {
	return GUID{
		Data1: data1,
		Data2: data2,
		Data3: data3,
		Data4: data4,
	}
}

// GUIDFromString creates a new GUID from its string representation.
func GUIDFromString(s string) (GUID, error) {
	return ParseGUID(s)
}

func StringToGUID(s string) GUID {
	guid, err := ParseGUID(s)
	if err != nil {
		return GUID{}
	}
	return guid
}

// FromBytes parses a GUID from its binary representation.
func GUIDFromBytes(data []byte) (GUID, error) {
	if len(data) < 16 {
		return GUID{}, fmt.Errorf("data too short for GUID, need 16 bytes")
	}

	var guid GUID
	guid.Data1 = binary.LittleEndian.Uint32(data[0:4])
	guid.Data2 = binary.LittleEndian.Uint16(data[4:6])
	guid.Data3 = binary.LittleEndian.Uint16(data[6:8])
	copy(guid.Data4[:], data[8:16])

	return guid, nil
}

// Bytes returns the binary representation of the GUID.
func (g GUID) Bytes() []byte {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint32(data[0:4], g.Data1)
	binary.LittleEndian.PutUint16(data[4:6], g.Data2)
	binary.LittleEndian.PutUint16(data[6:8], g.Data3)
	copy(data[8:16], g.Data4[:])
	return data
}

// String returns the standard string representation of the GUID.
func (g GUID) String() string {
	return fmt.Sprintf("%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		g.Data1, g.Data2, g.Data3,
		g.Data4[0], g.Data4[1], g.Data4[2], g.Data4[3],
		g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

// ParseBinGUID parses a binary GUID from data at offset.
func ParseBinGUID(data []byte, offset int) GUID {
	var guid GUID

	guid.Data1 = binary.LittleEndian.Uint32(data[offset : offset+4])
	guid.Data2 = binary.LittleEndian.Uint16(data[offset+4 : offset+6])
	guid.Data3 = binary.LittleEndian.Uint16(data[offset+6 : offset+8])
	guid.Data4 = [8]byte{
		data[offset+8],
		data[offset+9],
		data[offset+10],
		data[offset+11],
		data[offset+12],
		data[offset+13],
		data[offset+14],
		data[offset+15],
	}
	return guid
}

// Equal compares two GUIDs for equality.
func (g GUID) Equal(other GUID) bool {
	return g.Data1 == other.Data1 &&
		g.Data2 == other.Data2 &&
		g.Data3 == other.Data3 &&
		g.Data4 == other.Data4
}

// Common EFI GUIDs.
var (
	EFI_GLOBAL_VARIABLE_GUID = GUID{
		0x8BE4DF61,
		0x93CA,
		0x11d2,
		[8]byte{0xAA, 0x0D, 0x00, 0xE0, 0x98, 0x03, 0x2B, 0x8C},
	}
	EFI_IMAGE_SECURITY_DATABASE = GUID{
		0xd719b2cb,
		0x3d3a,
		0x4596,
		[8]byte{0xa3, 0xbc, 0xda, 0xd0, 0x0e, 0x67, 0x65, 0x6f},
	}
	MICROSOFT_GUID = GUID{
		0x77fa9abd,
		0x0359,
		0x4d32,
		[8]byte{0xbd, 0x60, 0x28, 0xf4, 0xe7, 0x8f, 0x78, 0x4b},
	}
	NvDataGUID = GUID{
		0x8d1b55ed,
		0xbebf,
		0x40b7,
		[8]byte{0x82, 0x46, 0xd8, 0xbd, 0x7d, 0x64, 0xed, 0xbe},
	}
	FfsGUID = GUID{
		0x8c8ce578,
		0x8a3d,
		0x4f1c,
		[8]byte{0x99, 0x35, 0x89, 0x61, 0x85, 0xc3, 0x2d, 0xd3},
	}
	AuthVarsGUID = GUID{
		0xaaf32c78,
		0x947b,
		0x439a,
		[8]byte{0xa1, 0x80, 0x2e, 0x14, 0x4e, 0xc3, 0x77, 0x92},
	}
)
