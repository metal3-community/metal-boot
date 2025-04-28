// Package manager provides implementations for firmware management interfaces.
package manager

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/bmcpi/pibmc/internal/firmware/types"
	"github.com/bmcpi/pibmc/internal/firmware/varstore"
	"github.com/go-logr/logr"
)

// EDK2Manager implements the FirmwareManager interface for Raspberry Pi EDK2 firmware.
type EDK2Manager struct {
	firmwarePath string
	varStore     *varstore.Edk2VarStore
	varList      efi.EfiVarList
	logger       logr.Logger
}

// NewEDK2Manager creates a new EDK2Manager for the given firmware file.
func NewEDK2Manager(firmwarePath string, logger logr.Logger) (*EDK2Manager, error) {
	manager := &EDK2Manager{
		firmwarePath: firmwarePath,
		logger:       logger.WithName("edk2-manager"),
	}

	if _, err := os.Stat(firmwarePath); err != nil {
		return nil, fmt.Errorf("firmware file not found: %w", err)
	}

	manager.varStore = varstore.NewEdk2VarStore(firmwarePath)
	if manager.varStore == nil {
		return nil, fmt.Errorf("failed to create varstore for %s", firmwarePath)
	}

	varList, err := manager.varStore.GetVarList()
	if err != nil {
		return nil, fmt.Errorf("failed to get variable list: %w", err)
	}
	manager.varList = varList

	return manager, nil
}

// GetBootOrder retrieves the current boot order.
func (m *EDK2Manager) GetBootOrder() ([]string, error) {
	bootOrderVar := m.varList.FindFirst("BootOrder")
	if bootOrderVar == nil {
		return nil, fmt.Errorf("BootOrder variable not found")
	}

	bootOrder := []string{}
	data := bootOrderVar.Data
	for i := 0; i < len(data); i += 2 {
		if i+1 >= len(data) {
			break
		}
		val := binary.LittleEndian.Uint16(data[i : i+2])
		bootOrder = append(bootOrder, fmt.Sprintf("Boot%04X", val))
	}

	return bootOrder, nil
}

// SetBootOrder sets a new boot order.
func (m *EDK2Manager) SetBootOrder(bootOrder []string) error {
	if len(bootOrder) == 0 {
		return fmt.Errorf("empty boot order")
	}

	data := make([]byte, len(bootOrder)*2)
	for i, bootName := range bootOrder {
		if !strings.HasPrefix(bootName, "Boot") {
			return fmt.Errorf("invalid boot entry name: %s", bootName)
		}

		hexStr := strings.TrimPrefix(bootName, "Boot")
		val, err := strconv.ParseUint(hexStr, 16, 16)
		if err != nil {
			return fmt.Errorf("invalid boot entry ID: %s", hexStr)
		}

		binary.LittleEndian.PutUint16(data[i*2:], uint16(val))
	}

	bootOrderVar := m.varList.FindFirst("BootOrder")
	if bootOrderVar == nil {
		// Create new BootOrder variable if it doesn't exist
		bootOrderVar = &efi.EfiVar{
			Name:       "BootOrder",
			Guid:       efi.EfiGlobalVariableGuid,
			Attributes: efi.EfiAttrBootserviceAccess | efi.EfiAttrRuntimeAccess | efi.EfiAttrNonVolatile,
			Data:       data,
		}
		m.varList = append(m.varList, bootOrderVar)
	} else {
		bootOrderVar.Data = data
	}

	return nil
}

// GetBootEntries retrieves all boot entries.
func (m *EDK2Manager) GetBootEntries() ([]types.BootEntry, error) {
	var bootEntries []types.BootEntry

	for _, v := range m.varList {
		if !strings.HasPrefix(v.Name, "Boot") || v.Name == "BootOrder" || v.Name == "BootNext" {
			continue
		}

		// Extract the boot ID (e.g., "0000" from "Boot0000")
		bootID := strings.TrimPrefix(v.Name, "Boot")

		// Parse boot entry data
		if len(v.Data) < 8 {
			m.logger.Info("Skipping invalid boot entry", "name", v.Name, "data_len", len(v.Data))
			continue
		}

		// First 4 bytes are attributes
		attributes := binary.LittleEndian.Uint32(v.Data[0:4])

		// Next 2 bytes are device path length
		pathLen := binary.LittleEndian.Uint16(v.Data[4:6])

		// Description starts at offset 6
		descEnd := 6
		for descEnd < len(v.Data) && v.Data[descEnd] != 0 && v.Data[descEnd+1] != 0 {
			descEnd += 2
		}

		// Get the description (UCS-16 encoded)
		name := efi.Ucs16ToString(v.Data[6 : descEnd+2])

		// Device path starts after the description (aligned to even boundary)
		dpStart := descEnd + 2
		if dpStart%2 != 0 {
			dpStart++
		}

		// Device path data
		var devPath string
		var optData string

		if dpStart+int(pathLen) <= len(v.Data) {
			dpData := v.Data[dpStart : dpStart+int(pathLen)]
			dp := efi.ParseDevicePath(dpData)
			devPath = dp.String()

			// Optional data if present
			if dpStart+int(pathLen) < len(v.Data) {
				optData = fmt.Sprintf("%x", v.Data[dpStart+int(pathLen):])
			}
		}

		enabled := (attributes & 1) == 1 // Check if LOAD_OPTION_ACTIVE is set

		entry := types.BootEntry{
			ID:       bootID,
			Name:     name,
			DevPath:  devPath,
			Enabled:  enabled,
			OptData:  optData,
			Position: -1, // Will be set based on BootOrder
		}

		bootEntries = append(bootEntries, entry)
	}

	// Update positions based on boot order
	bootOrder, err := m.GetBootOrder()
	if err == nil && len(bootOrder) > 0 {
		for i, bootName := range bootOrder {
			for j := range bootEntries {
				if "Boot"+bootEntries[j].ID == bootName {
					bootEntries[j].Position = i
					break
				}
			}
		}
	}

	return bootEntries, nil
}

// AddBootEntry adds a new boot entry.
func (m *EDK2Manager) AddBootEntry(entry types.BootEntry) error {
	// Validate entry
	if entry.ID == "" || entry.Name == "" {
		return fmt.Errorf("boot entry must have ID and Name")
	}

	// Create the boot variable name (e.g., "Boot0000")
	bootName := "Boot" + entry.ID

	// Check if this boot entry already exists
	for _, v := range m.varList {
		if v.Name == bootName {
			return fmt.Errorf("boot entry %s already exists", bootName)
		}
	}

	// Construct device path from string
	dp := efi.NewDevicePath()
	if err := dp.ParseFromString(entry.DevPath); err != nil {
		return fmt.Errorf("failed to parse device path: %w", err)
	}

	dpData := dp.Bytes()

	// Build the boot entry data
	// Format: [4 bytes attributes][2 bytes path length][description string (UCS-16)][null terminator][device path][optional data]

	attributes := uint32(0)
	if entry.Enabled {
		attributes |= 1 // Set LOAD_OPTION_ACTIVE flag
	}

	// Convert description to UCS-16
	descData := efi.StringToUcs16(entry.Name)

	// Calculate total size
	totalSize := 6 + len(descData) + len(dpData)
	if entry.OptData != "" {
		// Parse optional data from hex
		optBytes, err := parseHexString(entry.OptData)
		if err != nil {
			return fmt.Errorf("invalid optional data: %w", err)
		}
		totalSize += len(optBytes)
	}

	// Construct the variable data
	data := make([]byte, totalSize)
	binary.LittleEndian.PutUint32(data[0:], attributes)
	binary.LittleEndian.PutUint16(data[4:], uint16(len(dpData)))

	offset := 6
	copy(data[offset:], descData)
	offset += len(descData)

	copy(data[offset:], dpData)
	offset += len(dpData)

	if entry.OptData != "" {
		optBytes, _ := parseHexString(entry.OptData) // Error already checked
		copy(data[offset:], optBytes)
	}

	// Create the EFI variable
	efiVar := &efi.EfiVar{
		Name:       bootName,
		Guid:       efi.EfiGlobalVariableGuid,
		Attributes: efi.EfiAttrBootserviceAccess | efi.EfiAttrRuntimeAccess | efi.EfiAttrNonVolatile,
		Data:       data,
	}

	// Add to var list
	m.varList = append(m.varList, efiVar)

	// Update boot order if position is specified
	if entry.Position >= 0 {
		bootOrder, err := m.GetBootOrder()
		if err != nil {
			bootOrder = []string{}
		}

		// Create new boot order with the entry at the specified position
		newBootOrder := make([]string, 0, len(bootOrder)+1)
		entryAdded := false

		for i := 0; i <= len(bootOrder); i++ {
			if i == entry.Position {
				newBootOrder = append(newBootOrder, bootName)
				entryAdded = true
			}

			if i < len(bootOrder) {
				newBootOrder = append(newBootOrder, bootOrder[i])
			}
		}

		// If position was beyond the end, append at the end
		if !entryAdded {
			newBootOrder = append(newBootOrder, bootName)
		}

		if err := m.SetBootOrder(newBootOrder); err != nil {
			return fmt.Errorf("failed to update boot order: %w", err)
		}
	}

	return nil
}

// parseHexString converts a hex string to bytes.
func parseHexString(hexStr string) ([]byte, error) {
	// Remove any whitespace and "0x" prefixes
	cleaned := strings.ReplaceAll(hexStr, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "0x", "")

	// If odd length, pad with a leading zero
	if len(cleaned)%2 != 0 {
		cleaned = "0" + cleaned
	}

	bytes := make([]byte, len(cleaned)/2)
	for i := 0; i < len(cleaned); i += 2 {
		byteVal, err := strconv.ParseUint(cleaned[i:i+2], 16, 8)
		if err != nil {
			return nil, err
		}
		bytes[i/2] = byte(byteVal)
	}

	return bytes, nil
}

// UpdateBootEntry updates an existing boot entry.
func (m *EDK2Manager) UpdateBootEntry(id string, entry types.BootEntry) error {
	// Validate entry
	if entry.Name == "" {
		return fmt.Errorf("boot entry must have a Name")
	}

	// Create the boot variable name (e.g., "Boot0000")
	bootName := "Boot" + id

	// Find the existing boot entry
	var existingVar *efi.EfiVar
	for i, v := range m.varList {
		if v.Name == bootName {
			existingVar = m.varList[i]
			break
		}
	}

	if existingVar == nil {
		return fmt.Errorf("boot entry %s not found", bootName)
	}

	// Construct device path from string
	dp := efi.NewDevicePath()
	if err := dp.ParseFromString(entry.DevPath); err != nil {
		return fmt.Errorf("failed to parse device path: %w", err)
	}

	dpData := dp.Bytes()

	// Build the boot entry data
	// Format: [4 bytes attributes][2 bytes path length][description string (UCS-16)][null terminator][device path][optional data]

	attributes := uint32(0)
	if entry.Enabled {
		attributes |= 1 // Set LOAD_OPTION_ACTIVE flag
	}

	// Convert description to UCS-16
	descData := efi.StringToUcs16(entry.Name)

	// Calculate total size
	totalSize := 6 + len(descData) + len(dpData)
	var optBytes []byte
	if entry.OptData != "" {
		// Parse optional data from hex
		var err error
		optBytes, err = parseHexString(entry.OptData)
		if err != nil {
			return fmt.Errorf("invalid optional data: %w", err)
		}
		totalSize += len(optBytes)
	}

	// Construct the variable data
	data := make([]byte, totalSize)
	binary.LittleEndian.PutUint32(data[0:], attributes)
	binary.LittleEndian.PutUint16(data[4:], uint16(len(dpData)))

	offset := 6
	copy(data[offset:], descData)
	offset += len(descData)

	copy(data[offset:], dpData)
	offset += len(dpData)

	if len(optBytes) > 0 {
		copy(data[offset:], optBytes)
	}

	// Update the EFI variable
	existingVar.Data = data

	// Update boot order if position has changed
	if entry.Position >= 0 {
		bootOrder, err := m.GetBootOrder()
		if err != nil {
			return fmt.Errorf("failed to get boot order: %w", err)
		}

		// Find current position of this entry
		currentPos := -1
		for i, name := range bootOrder {
			if name == bootName {
				currentPos = i
				break
			}
		}

		// If position has changed and entry is in boot order
		if currentPos != entry.Position && currentPos != -1 {
			// Remove from current position
			bootOrder = append(bootOrder[:currentPos], bootOrder[currentPos+1:]...)

			// Insert at new position
			newBootOrder := make([]string, 0, len(bootOrder))
			inserted := false

			for i := 0; i <= len(bootOrder); i++ {
				if i == entry.Position {
					newBootOrder = append(newBootOrder, bootName)
					inserted = true
				}

				if i < len(bootOrder) {
					newBootOrder = append(newBootOrder, bootOrder[i])
				}
			}

			// If position was beyond the end, append at the end
			if !inserted {
				newBootOrder = append(newBootOrder, bootName)
			}

			if err := m.SetBootOrder(newBootOrder); err != nil {
				return fmt.Errorf("failed to update boot order: %w", err)
			}
		}
	}

	return nil
}

// DeleteBootEntry deletes a boot entry.
func (m *EDK2Manager) DeleteBootEntry(id string) error {
	bootName := "Boot" + id

	// Find the boot entry
	index := -1
	for i, v := range m.varList {
		if v.Name == bootName {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("boot entry %s not found", bootName)
	}

	// Remove from var list
	m.varList = append(m.varList[:index], m.varList[index+1:]...)

	// Remove from boot order
	bootOrder, err := m.GetBootOrder()
	if err == nil {
		newBootOrder := make([]string, 0, len(bootOrder))
		for _, name := range bootOrder {
			if name != bootName {
				newBootOrder = append(newBootOrder, name)
			}
		}

		if len(newBootOrder) < len(bootOrder) {
			if err := m.SetBootOrder(newBootOrder); err != nil {
				return fmt.Errorf("failed to update boot order: %w", err)
			}
		}
	}

	return nil
}

// SetBootNext sets the boot entry to use on next boot.
func (m *EDK2Manager) SetBootNext(index uint16) error {
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, index)

	bootNextVar := m.varList.FindFirst("BootNext")
	if bootNextVar == nil {
		// Create new BootNext variable if it doesn't exist
		bootNextVar = &efi.EfiVar{
			Name:       "BootNext",
			Guid:       efi.EfiGlobalVariableGuid,
			Attributes: efi.EfiAttrBootserviceAccess | efi.EfiAttrRuntimeAccess | efi.EfiAttrNonVolatile,
			Data:       data,
		}
		m.varList = append(m.varList, bootNextVar)
	} else {
		bootNextVar.Data = data
	}

	return nil
}

// GetBootNext retrieves the boot entry to use on next boot.
func (m *EDK2Manager) GetBootNext() (uint16, error) {
	bootNextVar := m.varList.FindFirst("BootNext")
	if bootNextVar == nil {
		return 0, fmt.Errorf("BootNext variable not found")
	}

	if len(bootNextVar.Data) < 2 {
		return 0, fmt.Errorf("invalid BootNext data length")
	}

	return binary.LittleEndian.Uint16(bootNextVar.Data), nil
}

// GetNetworkSettings retrieves network settings from firmware variables.
func (m *EDK2Manager) GetNetworkSettings() (types.NetworkSettings, error) {
	var settings types.NetworkSettings

	// Get MAC address
	macAddr, err := m.GetMacAddress()
	if err == nil {
		settings.MacAddress = macAddr.String()
	}

	// Find UEFI network configuration variables
	for _, v := range m.varList {
		if v.Guid == efi.EfiNetworkInterfaceIdGuid {
			if strings.HasPrefix(v.Name, "IPv4") {
				// Parse IPv4 settings
				if strings.Contains(v.Name, "DHCPEnabled") && len(v.Data) >= 1 {
					settings.EnableDHCP = v.Data[0] != 0
				} else if strings.Contains(v.Name, "Address") && len(v.Data) >= 4 {
					settings.IPAddress = fmt.Sprintf("%d.%d.%d.%d",
						v.Data[0], v.Data[1], v.Data[2], v.Data[3])
				} else if strings.Contains(v.Name, "SubnetMask") && len(v.Data) >= 4 {
					settings.SubnetMask = fmt.Sprintf("%d.%d.%d.%d",
						v.Data[0], v.Data[1], v.Data[2], v.Data[3])
				} else if strings.Contains(v.Name, "Gateway") && len(v.Data) >= 4 {
					settings.Gateway = fmt.Sprintf("%d.%d.%d.%d",
						v.Data[0], v.Data[1], v.Data[2], v.Data[3])
				} else if strings.Contains(v.Name, "DNSServers") && len(v.Data) >= 4 {
					// Each DNS server is 4 bytes
					for i := 0; i < len(v.Data); i += 4 {
						if i+3 < len(v.Data) {
							dns := fmt.Sprintf("%d.%d.%d.%d",
								v.Data[i], v.Data[i+1], v.Data[i+2], v.Data[i+3])
							settings.DNSServers = append(settings.DNSServers, dns)
						}
					}
				}
			} else if strings.HasPrefix(v.Name, "IPv6") {
				// Basic IPv6 support for now
				settings.EnableIPv6 = true
			} else if strings.Contains(v.Name, "VLAN") {
				// Parse VLAN settings
				if strings.Contains(v.Name, "Enabled") && len(v.Data) >= 1 {
					settings.VLANEnabled = v.Data[0] != 0
				} else if strings.Contains(v.Name, "ID") && len(v.Data) >= 2 {
					vlanID := binary.LittleEndian.Uint16(v.Data)
					settings.VLANID = fmt.Sprintf("%d", vlanID)
				}
			}
		}
	}

	return settings, nil
}

// SetNetworkSettings applies network settings to firmware variables.
func (m *EDK2Manager) SetNetworkSettings(settings types.NetworkSettings) error {
	// Set MAC address if provided
	if settings.MacAddress != "" {
		mac, err := net.ParseMAC(settings.MacAddress)
		if err != nil {
			return fmt.Errorf("invalid MAC address: %w", err)
		}

		if err := m.SetMacAddress(mac); err != nil {
			return fmt.Errorf("failed to set MAC address: %w", err)
		}
	}

	// Set IPv4 DHCP flag
	if err := m.setOrCreateVariable("IPv4DHCPEnabled", efi.EfiNetworkInterfaceIdGuid,
		[]byte{boolToByte(settings.EnableDHCP)}); err != nil {
		return fmt.Errorf("failed to set DHCP flag: %w", err)
	}

	// If static IP is being used, set IP address values
	if !settings.EnableDHCP {
		// Set IP address
		if settings.IPAddress != "" {
			ipBytes, err := parseIPv4String(settings.IPAddress)
			if err != nil {
				return fmt.Errorf("invalid IP address: %w", err)
			}
			if err := m.setOrCreateVariable("IPv4Address", efi.EfiNetworkInterfaceIdGuid, ipBytes); err != nil {
				return fmt.Errorf("failed to set IP address: %w", err)
			}
		}

		// Set subnet mask
		if settings.SubnetMask != "" {
			maskBytes, err := parseIPv4String(settings.SubnetMask)
			if err != nil {
				return fmt.Errorf("invalid subnet mask: %w", err)
			}
			if err := m.setOrCreateVariable("IPv4SubnetMask", efi.EfiNetworkInterfaceIdGuid, maskBytes); err != nil {
				return fmt.Errorf("failed to set subnet mask: %w", err)
			}
		}

		// Set gateway
		if settings.Gateway != "" {
			gatewayBytes, err := parseIPv4String(settings.Gateway)
			if err != nil {
				return fmt.Errorf("invalid gateway: %w", err)
			}
			if err := m.setOrCreateVariable("IPv4Gateway", efi.EfiNetworkInterfaceIdGuid, gatewayBytes); err != nil {
				return fmt.Errorf("failed to set gateway: %w", err)
			}
		}
	}

	// Set DNS servers
	if len(settings.DNSServers) > 0 {
		dnsBytes := make([]byte, len(settings.DNSServers)*4)
		for i, dnsServer := range settings.DNSServers {
			ipBytes, err := parseIPv4String(dnsServer)
			if err != nil {
				return fmt.Errorf("invalid DNS server address: %w", err)
			}
			copy(dnsBytes[i*4:], ipBytes)
		}
		if err := m.setOrCreateVariable("IPv4DNSServers", efi.EfiNetworkInterfaceIdGuid, dnsBytes); err != nil {
			return fmt.Errorf("failed to set DNS servers: %w", err)
		}
	}

	// Set IPv6 flag
	if err := m.setOrCreateVariable("IPv6Enabled", efi.EfiNetworkInterfaceIdGuid,
		[]byte{boolToByte(settings.EnableIPv6)}); err != nil {
		return fmt.Errorf("failed to set IPv6 flag: %w", err)
	}

	// Set VLAN settings
	if err := m.setOrCreateVariable("VLANEnabled", efi.EfiNetworkInterfaceIdGuid,
		[]byte{boolToByte(settings.VLANEnabled)}); err != nil {
		return fmt.Errorf("failed to set VLAN flag: %w", err)
	}

	if settings.VLANEnabled && settings.VLANID != "" {
		vlanID, err := strconv.ParseUint(settings.VLANID, 10, 16)
		if err != nil {
			return fmt.Errorf("invalid VLAN ID: %w", err)
		}

		vlanBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(vlanBytes, uint16(vlanID))

		if err := m.setOrCreateVariable("VLANID", efi.EfiNetworkInterfaceIdGuid, vlanBytes); err != nil {
			return fmt.Errorf("failed to set VLAN ID: %w", err)
		}
	}

	return nil
}

// boolToByte converts a boolean to a byte (0 or 1).
func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// parseIPv4String converts an IPv4 address string to a 4-byte array.
func parseIPv4String(ipStr string) ([]byte, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address")
	}

	ip = ip.To4()
	if ip == nil {
		return nil, fmt.Errorf("not an IPv4 address")
	}

	return []byte(ip), nil
}

// setOrCreateVariable sets an existing variable or creates a new one if it doesn't exist.
func (m *EDK2Manager) setOrCreateVariable(name string, guid efi.Guid, data []byte) error {
	v := m.varList.FindVar(name, guid)
	if v != nil {
		v.Data = data
	} else {
		newVar := &efi.EfiVar{
			Name:       name,
			Guid:       guid,
			Attributes: efi.EfiAttrBootserviceAccess | efi.EfiAttrRuntimeAccess | efi.EfiAttrNonVolatile,
			Data:       data,
		}
		m.varList = append(m.varList, newVar)
	}
	return nil
}

// GetMacAddress retrieves the MAC address from firmware variables.
func (m *EDK2Manager) GetMacAddress() (net.HardwareAddr, error) {
	macVar := m.varList.FindFirst("MacAddress")
	if macVar == nil {
		// Check for alternate variable names
		macVar = m.varList.FindFirst("PermanentMacAddress")
		if macVar == nil {
			return nil, fmt.Errorf("MAC address variable not found")
		}
	}

	// MAC should be at least 6 bytes
	if len(macVar.Data) < 6 {
		return nil, fmt.Errorf("invalid MAC address data length")
	}

	return net.HardwareAddr(macVar.Data[:6]), nil
}

// SetMacAddress sets the MAC address in firmware variables.
func (m *EDK2Manager) SetMacAddress(mac net.HardwareAddr) error {
	if len(mac) < 6 {
		return fmt.Errorf("invalid MAC address length")
	}

	// Standard variable name for MAC address
	if err := m.setOrCreateVariable("MacAddress", efi.EfiGlobalVariableGuid, mac); err != nil {
		return fmt.Errorf("failed to set MAC address: %w", err)
	}

	// Also set the permanent MAC address variable which some firmware may use
	if err := m.setOrCreateVariable("PermanentMacAddress", efi.EfiGlobalVariableGuid, mac); err != nil {
		return fmt.Errorf("failed to set permanent MAC address: %w", err)
	}

	return nil
}

// GetVariable retrieves a firmware variable by name.
func (m *EDK2Manager) GetVariable(name string) (*efi.EfiVar, error) {
	// Try to find in common GUIDs
	v := m.varList.FindFirst(name)
	if v == nil {
		return nil, fmt.Errorf("variable %s not found", name)
	}
	return v, nil
}

// SetVariable sets a firmware variable.
func (m *EDK2Manager) SetVariable(name string, value *efi.EfiVar) error {
	if value == nil {
		return fmt.Errorf("variable value cannot be nil")
	}

	// Find existing variable
	existingVar := m.varList.FindVar(name, value.Guid)
	if existingVar != nil {
		// Update existing variable
		existingVar.Data = value.Data
		existingVar.Attributes = value.Attributes
	} else {
		// Add new variable
		m.varList = append(m.varList, value)
	}

	return nil
}

// ListVariables lists all firmware variables.
func (m *EDK2Manager) ListVariables() (map[string]*efi.EfiVar, error) {
	variables := make(map[string]*efi.EfiVar)

	for _, v := range m.varList {
		variables[v.Name] = v
	}

	return variables, nil
}

// EnablePXEBoot enables or disables PXE boot.
func (m *EDK2Manager) EnablePXEBoot(enable bool) error {
	// This is firmware-specific - for Raspberry Pi EDK2, we typically
	// need to check and possibly add a PXE boot entry

	// Search for existing PXE boot entries
	bootEntries, err := m.GetBootEntries()
	if err != nil {
		return fmt.Errorf("failed to get boot entries: %w", err)
	}

	pxeBootFound := false
	var pxeBootID string

	for _, entry := range bootEntries {
		// Check if the description or device path indicates a PXE boot entry
		if strings.Contains(strings.ToLower(entry.Name), "pxe") ||
			strings.Contains(strings.ToLower(entry.DevPath), "pxe") {
			pxeBootFound = true
			pxeBootID = entry.ID
			break
		}
	}

	if enable {
		if !pxeBootFound {
			// Create a new PXE boot entry
			// This is a simplified example - actual PXE path may need to be built
			// based on specific network controller info
			newEntry := types.BootEntry{
				ID:       getNextAvailableBootID(bootEntries),
				Name:     "Network PXE Boot",
				DevPath:  "PciRoot(0)/Pci(2,0)/MAC()/IPv4()/Pxe()",
				Enabled:  true,
				Position: 0, // Add to the beginning of boot order
			}

			if err := m.AddBootEntry(newEntry); err != nil {
				return fmt.Errorf("failed to add PXE boot entry: %w", err)
			}
		} else {
			// Make sure the PXE entry is enabled
			for _, entry := range bootEntries {
				if entry.ID == pxeBootID && !entry.Enabled {
					entry.Enabled = true
					if err := m.UpdateBootEntry(pxeBootID, entry); err != nil {
						return fmt.Errorf("failed to enable PXE boot entry: %w", err)
					}
					break
				}
			}
		}
	} else if pxeBootFound {
		// Disable the PXE boot entry
		for _, entry := range bootEntries {
			if entry.ID == pxeBootID && entry.Enabled {
				entry.Enabled = false
				if err := m.UpdateBootEntry(pxeBootID, entry); err != nil {
					return fmt.Errorf("failed to disable PXE boot entry: %w", err)
				}
				break
			}
		}
	}

	return nil
}

// getNextAvailableBootID finds the next available boot ID.
func getNextAvailableBootID(entries []types.BootEntry) string {
	// Find the highest existing ID
	maxID := uint16(0)
	for _, entry := range entries {
		id, err := strconv.ParseUint(entry.ID, 16, 16)
		if err == nil && uint16(id) > maxID {
			maxID = uint16(id)
		}
	}

	// Return the next ID as a 4-digit hex string
	return fmt.Sprintf("%04X", maxID+1)
}

// EnableHTTPBoot enables or disables HTTP boot.
func (m *EDK2Manager) EnableHTTPBoot(enable bool) error {
	// Similar to PXE boot, check for and manage HTTP boot entries
	bootEntries, err := m.GetBootEntries()
	if err != nil {
		return fmt.Errorf("failed to get boot entries: %w", err)
	}

	httpBootFound := false
	var httpBootID string

	for _, entry := range bootEntries {
		// Check if the description or device path indicates an HTTP boot entry
		if strings.Contains(strings.ToLower(entry.Name), "http") ||
			strings.Contains(strings.ToLower(entry.DevPath), "http") {
			httpBootFound = true
			httpBootID = entry.ID
			break
		}
	}

	if enable {
		if !httpBootFound {
			// Create a new HTTP boot entry
			newEntry := types.BootEntry{
				ID:       getNextAvailableBootID(bootEntries),
				Name:     "Network HTTP Boot",
				DevPath:  "PciRoot(0)/Pci(2,0)/MAC()/IPv4()/URI()",
				Enabled:  true,
				Position: 0, // Add to the beginning of boot order
			}

			if err := m.AddBootEntry(newEntry); err != nil {
				return fmt.Errorf("failed to add HTTP boot entry: %w", err)
			}
		} else {
			// Make sure the HTTP entry is enabled
			for _, entry := range bootEntries {
				if entry.ID == httpBootID && !entry.Enabled {
					entry.Enabled = true
					if err := m.UpdateBootEntry(httpBootID, entry); err != nil {
						return fmt.Errorf("failed to enable HTTP boot entry: %w", err)
					}
					break
				}
			}
		}
	} else if httpBootFound {
		// Disable the HTTP boot entry
		for _, entry := range bootEntries {
			if entry.ID == httpBootID && entry.Enabled {
				entry.Enabled = false
				if err := m.UpdateBootEntry(httpBootID, entry); err != nil {
					return fmt.Errorf("failed to disable HTTP boot entry: %w", err)
				}
				break
			}
		}
	}

	return nil
}

// SetFirmwareTimeoutSeconds sets the boot menu timeout in seconds.
func (m *EDK2Manager) SetFirmwareTimeoutSeconds(seconds int) error {
	if seconds < 0 {
		return fmt.Errorf("timeout value cannot be negative")
	}

	// Convert seconds to platform timer ticks
	// UEFI spec uses 100ns units
	ticks := uint64(seconds) * 10000000

	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, ticks)

	return m.setOrCreateVariable("Timeout", efi.EfiGlobalVariableGuid, data)
}

// SetConsoleConfig sets the console device and baud rate.
func (m *EDK2Manager) SetConsoleConfig(consoleName string, baudRate int) error {
	if consoleName == "" {
		return fmt.Errorf("console name cannot be empty")
	}

	if baudRate <= 0 {
		return fmt.Errorf("baud rate must be positive")
	}

	// Convert console name to UCS-16
	consoleData := efi.StringToUcs16(consoleName)

	if err := m.setOrCreateVariable("ConsoleName", efi.EfiGlobalVariableGuid, consoleData); err != nil {
		return fmt.Errorf("failed to set console name: %w", err)
	}

	// Set baud rate as a 32-bit value
	baudData := make([]byte, 4)
	binary.LittleEndian.PutUint32(baudData, uint32(baudRate))

	if err := m.setOrCreateVariable("ConsoleBaudRate", efi.EfiGlobalVariableGuid, baudData); err != nil {
		return fmt.Errorf("failed to set console baud rate: %w", err)
	}

	return nil
}

// GetSystemInfo retrieves system information.
func (m *EDK2Manager) GetSystemInfo() (types.SystemInfo, error) {
	info := make(types.SystemInfo)

	// Add firmware path
	info["FirmwarePath"] = m.firmwarePath

	// Extract firmware version from EFI variables
	fwVersion, err := m.GetFirmwareVersion()
	if err == nil {
		info["FirmwareVersion"] = fwVersion
	} else {
		info["FirmwareVersion"] = "Unknown"
	}

	// Extract system UUID if available
	systemUUIDVar := m.varList.FindFirst("SystemUUID")
	if systemUUIDVar != nil && len(systemUUIDVar.Data) >= 16 {
		uuid, err := efi.FormatGuid(systemUUIDVar.Data)
		if err == nil {
			info["SystemUUID"] = uuid
		}
	}

	// Get firmware timestamp
	if fwStat, err := os.Stat(m.firmwarePath); err == nil {
		info["LastModified"] = fwStat.ModTime().Format(time.RFC3339)
	}

	// Get firmware size
	fwSize := int64(0)
	if fwStat, err := os.Stat(m.firmwarePath); err == nil {
		fwSize = fwStat.Size()
		info["Size"] = fmt.Sprintf("%d", fwSize)
	}

	// Platform information from variables if available
	platformNameVar := m.varList.FindFirst("PlatformName")
	if platformNameVar != nil {
		info["PlatformName"] = efi.Ucs16ToString(platformNameVar.Data)
	}

	return info, nil
}

// GetFirmwareVersion retrieves the firmware version.
func (m *EDK2Manager) GetFirmwareVersion() (string, error) {
	// Try to find the version from a variable first
	versionVar := m.varList.FindFirst("FirmwareVersion")
	if versionVar != nil {
		if len(versionVar.Data) > 0 {
			// If the data is UCS-16 encoded
			if len(versionVar.Data) >= 2 {
				return efi.Ucs16ToString(versionVar.Data), nil
			}
			// Else assume it's a string
			return string(versionVar.Data), nil
		}
	}

	// If not found in variables, check if it's embedded in the firmware file name
	base := filepath.Base(m.firmwarePath)
	parts := strings.Split(base, "-")
	if len(parts) > 1 {
		// Attempt to extract version if filename follows pattern like "firmware-v1.2.3.bin"
		for _, part := range parts {
			if strings.HasPrefix(part, "v") && len(part) > 1 {
				return part, nil
			}
		}
	}

	// If we still can't determine version, return an error
	return "", fmt.Errorf("firmware version not found")
}

// UpdateFirmware updates the firmware with new data.
func (m *EDK2Manager) UpdateFirmware(firmwareData []byte) error {
	if len(firmwareData) == 0 {
		return fmt.Errorf("firmware data is empty")
	}

	// Create backup of the current firmware
	backupPath := m.firmwarePath + ".bak." + time.Now().Format("20060102150405")
	if err := func() error {
		srcFile, err := os.Open(m.firmwarePath)
		if err != nil {
			return fmt.Errorf("failed to open source firmware: %w", err)
		}
		defer srcFile.Close()

		dstFile, err := os.Create(backupPath)
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	}(); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Write the new firmware
	if err := os.WriteFile(m.firmwarePath+".new", firmwareData, 0644); err != nil {
		return fmt.Errorf("failed to write new firmware: %w", err)
	}

	// Validate the new firmware before applying it
	testVarStore := varstore.NewEdk2VarStore(m.firmwarePath + ".new")
	if testVarStore == nil {
		os.Remove(m.firmwarePath + ".new")
		return fmt.Errorf("new firmware is invalid")
	}

	_, err := testVarStore.GetVarList()
	if err != nil {
		os.Remove(m.firmwarePath + ".new")
		return fmt.Errorf("new firmware has invalid variable store: %w", err)
	}

	// Save the changes we've made to the variable store before replacing the firmware
	if err := m.SaveChanges(); err != nil {
		return fmt.Errorf("failed to save current changes: %w", err)
	}

	// Replace the existing firmware with the new one
	if err := os.Rename(m.firmwarePath+".new", m.firmwarePath); err != nil {
		return fmt.Errorf("failed to replace firmware: %w", err)
	}

	// Reload the firmware
	m.varStore = varstore.NewEdk2VarStore(m.firmwarePath)
	if m.varStore == nil {
		// Try to restore from backup if the new firmware is problematic
		os.Rename(backupPath, m.firmwarePath)
		return fmt.Errorf("failed to load new firmware")
	}

	varList, err := m.varStore.GetVarList()
	if err != nil {
		// Try to restore from backup if the new firmware is problematic
		os.Rename(backupPath, m.firmwarePath)
		return fmt.Errorf("failed to get variable list from new firmware: %w", err)
	}

	m.varList = varList
	return nil
}

// SaveChanges writes the firmware variables back to the firmware file.
func (m *EDK2Manager) SaveChanges() error {
	if m.varStore == nil {
		return fmt.Errorf("varstore is not initialized")
	}

	if err := m.varStore.WriteVarStore(m.firmwarePath, m.varList); err != nil {
		return fmt.Errorf("failed to write variable store: %w", err)
	}

	return nil
}

// RevertChanges discards any changes made to the firmware variables.
func (m *EDK2Manager) RevertChanges() error {
	// Reload the variables from the firmware file
	if m.varStore == nil {
		return fmt.Errorf("varstore is not initialized")
	}

	varList, err := m.varStore.GetVarList()
	if err != nil {
		return fmt.Errorf("failed to reload variable list: %w", err)
	}

	m.varList = varList
	return nil
}

// ResetToDefaults resets the firmware to default settings.
func (m *EDK2Manager) ResetToDefaults() error {
	// For EDK2, we'll reset specific variables to defaults

	// Reset boot order to empty
	bootOrderVar := m.varList.FindFirst("BootOrder")
	if bootOrderVar != nil {
		bootOrderVar.Data = []byte{}
	}

	// Reset timeout to default
	timeoutVar := m.varList.FindFirst("Timeout")
	if timeoutVar != nil {
		// Default timeout: 3 seconds (30 million 100ns units)
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, 30000000)
		timeoutVar.Data = data
	}

	// Reset network settings: enable DHCP by default
	dhcpVar := m.varList.FindVar("IPv4DHCPEnabled", efi.EfiNetworkInterfaceIdGuid)
	if dhcpVar != nil {
		dhcpVar.Data = []byte{1} // Enable DHCP
	}

	// Save the changes
	return m.SaveChanges()
}
