package main

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/bmcpi/pibmc/internal/firmware/varstore"
)

// type BootEntry struct {
// 	Attr    uint32
// 	Title   string
// 	DevPath string
// 	MACAddr string
// 	OptData []byte
// }

// func (b *BootEntry) UnmarshalJSON(data []byte) error {
// 	// Extract the title (UTF-16LE decoding)
// 	var titleBuilder strings.Builder
// 	for i := 4; i < len(data); i += 2 {
// 		if data[i] == 0 && data[i+1] == 0 { // Null-terminator for UTF-16
// 			break
// 		}
// 		titleBuilder.WriteByte(data[i]) // Extract ASCII characters from UTF-16
// 	}
// 	title := titleBuilder.String()

// 	// Check if it starts with "D" (Boot Entry Identifier)
// 	isBootEntry := false
// 	if strings.HasPrefix(title, "D") {
// 		isBootEntry = true
// 		title = title[1:] // Remove "D" prefix
// 	}

// 	// Extract MAC address (fixed position based on observed structure)
// 	macStart := 24 // Position where MAC is expected
// 	macBytes := data[macStart : macStart+6]
// 	macAddr := fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", macBytes[0], macBytes[1], macBytes[2], macBytes[3], macBytes[4], macBytes[5])

// 	b.MACAddr = macAddr

// 	// Extract optional data (last 16 bytes)
// 	optDataStart := len(data) - 16
// 	optDataHex := hex.EncodeToString(data[optDataStart:])

// 	b.OptData = data[optDataStart:]

// 	b.Title = title

// 	// Format decoded output
// 	decoded := fmt.Sprintf(`title="%s (MAC:%s)" devpath=MAC()/IPv4() optdata=%s`, title, macAddr, optDataHex)

// 	// Include boot entry info if applicable
// 	if isBootEntry {
// 		decoded = fmt.Sprintf("[Boot Entry] %s", decoded)
// 	}

// 	return nil
// }

// type Edk2EfiVars struct {
// 	Boot0000 struct {
// 		Title   string `json:"title"`
// 		DevPath string `json:"devpath"`
// 		OptData string `json:"optdata"`
// 	} `json:"Boot0000"`
// }

// parseEncodedData decodes the binary+hex encoded UEFI PXE data
func parseEncodedData(encodedHex string) (string, error) {
	// Convert hex string to byte array
	data, err := hex.DecodeString(encodedHex)
	if err != nil {
		return "", fmt.Errorf("error decoding hex: %v", err)
	}

	if len(data) == 4 {
		return decodeDWord(data)
	}

	if len(data) >= 24 {
		return decodeBootEntry(data)
	}

	return "", errors.New("invalid encoded data")
}

func decodeBootEntry(data []byte) (string, error) {
	// Extract the title (UTF-16LE decoding)
	var titleBuilder strings.Builder
	for i := 4; i < len(data); i += 2 {
		if data[i] == 0 && data[i+1] == 0 { // Null-terminator for UTF-16
			break
		}
		titleBuilder.WriteByte(data[i]) // Extract ASCII characters from UTF-16
	}
	title := titleBuilder.String()

	// Check if it starts with "D" (Boot Entry Identifier)
	isBootEntry := false
	if strings.HasPrefix(title, "D") {
		isBootEntry = true
		title = title[1:] // Remove "D" prefix
	}

	// Extract MAC address (fixed position based on observed structure)
	macStart := 24 // Position where MAC is expected
	macBytes := data[macStart : macStart+6]
	macAddr := fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", macBytes[0], macBytes[1], macBytes[2], macBytes[3], macBytes[4], macBytes[5])

	// Extract optional data (last 16 bytes)
	optDataStart := len(data) - 16
	optDataHex := hex.EncodeToString(data[optDataStart:])

	// Format decoded output
	decoded := fmt.Sprintf(`title="%s (MAC:%s)" devpath=MAC()/IPv4() optdata=%s`, title, macAddr, optDataHex)

	// Include boot entry info if applicable
	if isBootEntry {
		decoded = fmt.Sprintf("[Boot Entry] %s", decoded)
	}
	return decoded, nil
}

// decodeDWord decodes a little-endian DWord from hex
func decodeDWord(data []byte) (string, error) {
	// Convert little-endian bytes to uint32
	value := binary.LittleEndian.Uint32(data)
	return fmt.Sprintf("0x%08X", value), nil
}

// encodeDWord encodes a uint32 value to little-endian hex
func encodeDWord(value uint32) string {
	// Create a 4-byte array
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, value)

	// Return as hex string
	return hex.EncodeToString(data)
}

// func (self *BootEntry) Parse(data []byte) error {
// 	// Extract attr (uint32) and pathsize (uint16)
// 	self.Attr = binary.LittleEndian.Uint32(data[0:4])
// 	pathsize := binary.LittleEndian.Uint16(data[4:6])

// 	// Parse title using UCS-16
// 	self.Title = efi.FromUCS16(data[6:])

// 	// Extract path data
// 	titleSize := self.Title.Size()
// 	pathStart := 6 + titleSize
// 	pathEnd := pathStart + int(pathsize)
// 	path := data[pathStart:pathEnd]
// 	self.DevPath = efi.NewDevicePath(path)

// 	// Extract optional data if it exists
// 	if len(data) > pathEnd {
// 		self.OptData = data[pathEnd:]
// 	}

// 	return nil
// }

func main() {

	readFile()
	readFile()

	return

	// Encoded UEFI PXE data
	encodedHex := "0100000044005500450046004900200050005800450076003400200028004d00410043003a0044003800330041004400440035004100340034003000430029000000030b2500d83add5a440c000000000000000000000000000000000000000000000000000001030c1b0000000000000000000000000000000000000000000000007fff04004eac0881119f594d850ee21a522c59b2"

	data, err := hex.DecodeString(encodedHex)
	if err != nil {
		panic(fmt.Errorf("error decoding hex: %v", err))
	}

	efiVarList := efi.EfiVarList{}

	efiVarList.UnmarshalJSON([]byte(efiConfig))

	err = efiVarList.UnmarshalJSON([]byte(efiConfig))
	if err != nil {
		panic(fmt.Errorf("error decoding EFI JSON: %v", err))
	}

	bootEntries, err := efiVarList.ListBootEntries()
	if err != nil {
		panic(fmt.Errorf("error listing boot entries: %v", err))
	}

	for index, entry := range bootEntries {
		fmt.Printf("Boot%04X: %s\n", index, entry)
	}

	for k, v := range efiVarList {
		fmt.Printf("%s: %s\n", k, v)
	}

	bootent := efi.NewBootEntry(data, 0, nil, nil, nil)

	fmt.Println("Decoding UEFI PXE data...")

	fmt.Printf("title=\"%s\" devpath=%s optdata=%s\n", bootent.Title, bootent.DevicePath.String(), bootent.OptData)

	fmt.Println(bootent)

	return

	// // Decode the input and print the output
	// decodedOutput, err := parseEncodedData(encodedHex)
	// if err != nil {
	// 	fmt.Println("Error decoding:", err)
	// 	return
	// }
	// fmt.Println(decodedOutput)
}

func readFile() {
	filename := "/Users/atkini01/src/go/pibmc/cmd/pibmc/RPI_EFI.fd"
	vs := varstore.NewEdk2VarStore(filename)

	efiVarList := vs.GetVarList()

	bootEntries, err := efiVarList.ListBootEntries()
	if err != nil {
		panic(fmt.Errorf("error listing boot entries: %v", err))
	}

	bootEntries[5].Title = *efi.NewUCS16String("Test Boot Entry")

	for index, entry := range bootEntries {
		fmt.Printf("Boot%04X: %s\n", index, entry)
	}

	for k, v := range efiVarList {
		fmt.Printf("%s: %s\n", k, v)
	}

	vs.WriteVarStore(filename, efiVarList)
}
