package firmware

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bmcpi/pibmc/internal/firmware/edk2"
	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/bmcpi/pibmc/internal/firmware/varstore"
	"github.com/go-logr/logr"
)

// EDK2Manager implements the FirmwareManager interface for Raspberry Pi EDK2 firmware
type EDK2Manager struct {
	firmwarePath string
	varStore     *varstore.Edk2VarStore
	varList      efi.EfiVarList
	logger       logr.Logger
}

// NewEDK2Manager creates a new EDK2Manager for the given firmware file
func NewEDK2Manager(firmwarePath string, logger logr.Logger) (*EDK2Manager, error) {
	manager := &EDK2Manager{
		firmwarePath: firmwarePath,
		logger:       logger.WithName("edk2-manager"),
	}

	if _, err := os.Stat(firmwarePath); os.IsNotExist(err) {

		firmwareRoot := filepath.Dir(firmwarePath)

		if err := os.MkdirAll(firmwareRoot, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create firmware directory: %w", err)
		}

		for k, f := range edk2.Files {
			kf := filepath.Join(firmwareRoot, k)
			kfr := filepath.Dir(kf)

			if kfr != firmwareRoot {
				if err := os.MkdirAll(kfr, 0o755); err != nil {
					return nil, fmt.Errorf("failed to create firmware directory: %w", err)
				}
			}

			if err := os.WriteFile(kf, f, 0o644); err != nil {
				return nil, fmt.Errorf("failed to create firmware file: %w", err)
			}
		}
	} else {
		return nil, fmt.Errorf("firmware file already exists: %s", firmwarePath)
	}

	// Initialize the variable store
	manager.varStore = varstore.NewEdk2VarStore(firmwarePath)
	manager.varStore.Logger = logger.WithName("edk2-varstore")

	// Load the variable list
	var err error
	manager.varList, err = manager.varStore.GetVarList()
	if err != nil {
		return nil, fmt.Errorf("failed to get variable list: %w", err)
	}

	return manager, nil
}

// GetBootOrder retrieves the boot order as a list of entry IDs
func (m *EDK2Manager) GetBootOrder() ([]string, error) {
	bootOrderVar, found := m.varList[efi.BootOrderName]
	if !found {
		return []string{}, nil
	}

	bootSequence, err := bootOrderVar.GetBootOrder()
	if err != nil {
		return nil, fmt.Errorf("failed to parse boot order: %w", err)
	}

	result := make([]string, len(bootSequence))
	for i, id := range bootSequence {
		result[i] = fmt.Sprintf("%04X", id)
	}

	return result, nil
}

func (m *EDK2Manager) SetBootNext(index uint16) error {
	return m.varList.SetBootNext(index)
}

func (m *EDK2Manager) GetBootNext() (uint16, error) {
	bootNextVar, found := m.varList[efi.BootNextName]
	if !found {
		return 0, nil
	}
	return bootNextVar.GetBootNext()
}

// SetBootOrder sets the boot order from a list of entry IDs
func (m *EDK2Manager) SetBootOrder(order []string) error {
	bootSequence := make([]uint16, len(order))

	for i, id := range order {
		// Remove "Boot" prefix if present
		id = strings.TrimPrefix(id, "Boot")

		// Parse the hex entry ID
		entryID, err := strconv.ParseUint(id, 16, 16)
		if err != nil {
			return fmt.Errorf("invalid boot entry ID '%s': %w", id, err)
		}

		bootSequence[i] = uint16(entryID)
	}

	// Get or create the BootOrder variable
	bootOrderVar, found := m.varList[efi.BootOrderName]
	if !found {
		bootOrderVar = &efi.EfiVar{
			Name: efi.NewUCS16String(efi.BootOrderName),
			Guid: efi.StringToGUID(efi.EFI_GLOBAL_VARIABLE),
			Attr: efi.EFI_VARIABLE_NON_VOLATILE |
				efi.EFI_VARIABLE_BOOTSERVICE_ACCESS |
				efi.EFI_VARIABLE_RUNTIME_ACCESS,
		}
		m.varList[efi.BootOrderName] = bootOrderVar
	}

	// Set the new boot order
	bootOrderVar.SetBootOrder(bootSequence)

	return nil
}

// GetBootEntries returns all boot entries from the firmware
func (m *EDK2Manager) GetBootEntries() ([]BootEntry, error) {
	bootEntries, err := m.varList.ListBootEntries()
	if err != nil {
		return nil, fmt.Errorf("failed to list boot entries: %w", err)
	}

	// Convert to the public BootEntry type
	result := make([]BootEntry, 0, len(bootEntries))
	for id, entry := range bootEntries {
		// Skip empty entries
		if entry == nil {
			continue
		}

		position := 0
		enabled := (entry.Attr & efi.LOAD_OPTION_ACTIVE) != 0

		// Get position from boot order
		bootOrderVar, found := m.varList[efi.BootOrderName]
		if found {
			bootSequence, err := bootOrderVar.GetBootOrder()
			if err == nil {
				for i, bootID := range bootSequence {
					if bootID == uint16(id) {
						position = i
						break
					}
				}
			}
		}

		bootEntry := BootEntry{
			ID:       fmt.Sprintf("%04X", id),
			Name:     entry.Title.String(),
			DevPath:  entry.DevicePath.String(),
			Enabled:  enabled,
			Position: position,
		}

		result = append(result, bootEntry)
	}

	return result, nil
}

// AddBootEntry adds a new boot entry to the firmware
func (m *EDK2Manager) AddBootEntry(entry BootEntry) error {
	// Find the next available boot entry ID
	maxID := uint16(0)
	for k := range m.varList {
		if strings.HasPrefix(k, efi.BootPrefix) && len(k) == 8 {
			idStr := k[4:] // Extract the ID portion
			id, err := strconv.ParseUint(idStr, 16, 16)
			if err == nil && uint16(id) > maxID {
				maxID = uint16(id)
			}
		}
	}
	nextID := maxID + 1

	// Create the boot entry name
	bootEntryName := fmt.Sprintf("%s%04X", efi.BootPrefix, nextID)

	// Create or update the boot entry variable
	bootEntryVar := &efi.EfiVar{
		Name: efi.NewUCS16String(bootEntryName),
		Guid: efi.StringToGUID(efi.EFI_GLOBAL_VARIABLE),
		Attr: efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS | efi.EFI_VARIABLE_RUNTIME_ACCESS,
	}

	// Set attributes based on enabled status
	attr := uint32(0)
	if entry.Enabled {
		attr |= efi.LOAD_OPTION_ACTIVE
	}

	// Set the boot entry with the specified title and device path
	err := bootEntryVar.SetBootEntry(attr, entry.Name, entry.DevPath, []byte(entry.OptData))
	if err != nil {
		return fmt.Errorf("failed to set boot entry: %w", err)
	}

	// Add the entry to the variable list
	m.varList[bootEntryName] = bootEntryVar

	// Update the boot order if position is specified
	if entry.Position >= 0 {
		bootOrder, err := m.GetBootOrder()
		if err != nil {
			return fmt.Errorf("failed to get boot order: %w", err)
		}

		// Convert the new ID to a string format matching the boot order
		newEntryID := fmt.Sprintf("%04X", nextID)

		// Insert the new entry at the specified position
		if entry.Position >= len(bootOrder) {
			bootOrder = append(bootOrder, newEntryID)
		} else {
			bootOrder = append(bootOrder[:entry.Position], append([]string{newEntryID}, bootOrder[entry.Position:]...)...)
		}

		// Update the boot order
		if err := m.SetBootOrder(bootOrder); err != nil {
			return fmt.Errorf("failed to update boot order: %w", err)
		}
	}

	return nil
}

// UpdateBootEntry updates an existing boot entry in the firmware
func (m *EDK2Manager) UpdateBootEntry(id string, entry BootEntry) error {
	// Add "Boot" prefix if not present
	if !strings.HasPrefix(id, efi.BootPrefix) {
		id = efi.BootPrefix + id
	}

	// Check if the entry exists
	bootEntryVar, found := m.varList[id]
	if !found {
		return fmt.Errorf("boot entry not found: %s", id)
	}

	// Get the current boot entry
	currentEntry, err := bootEntryVar.GetBootEntry()
	if err != nil {
		return fmt.Errorf("failed to parse boot entry: %w", err)
	}

	// Set attributes based on enabled status
	attr := currentEntry.Attr
	if entry.Enabled {
		attr |= efi.LOAD_OPTION_ACTIVE
	} else {
		attr &= ^uint32(efi.LOAD_OPTION_ACTIVE)
	}

	// Update the boot entry
	err = bootEntryVar.SetBootEntry(attr, entry.Name, entry.DevPath, currentEntry.OptData)
	if err != nil {
		return fmt.Errorf("failed to update boot entry: %w", err)
	}

	// Update the boot order if position is specified
	if entry.Position >= 0 {
		// Extract numeric ID from the boot entry
		idStr := strings.TrimPrefix(id, efi.BootPrefix)
		bootEntryID, err := strconv.ParseUint(idStr, 16, 16)
		if err != nil {
			return fmt.Errorf("invalid boot entry ID: %w", err)
		}

		bootOrder, err := m.GetBootOrder()
		if err != nil {
			return fmt.Errorf("failed to get boot order: %w", err)
		}

		// Find and remove the entry from the current boot order
		entryIndex := -1
		entryIDStr := fmt.Sprintf("%04X", bootEntryID)
		for i, orderID := range bootOrder {
			if orderID == entryIDStr {
				entryIndex = i
				break
			}
		}

		if entryIndex >= 0 {
			bootOrder = append(bootOrder[:entryIndex], bootOrder[entryIndex+1:]...)
		}

		// Insert the entry at the new position
		if entry.Position >= len(bootOrder) {
			bootOrder = append(bootOrder, entryIDStr)
		} else {
			bootOrder = append(bootOrder[:entry.Position], append([]string{entryIDStr}, bootOrder[entry.Position:]...)...)
		}

		// Update the boot order
		if err := m.SetBootOrder(bootOrder); err != nil {
			return fmt.Errorf("failed to update boot order: %w", err)
		}
	}

	return nil
}

// DeleteBootEntry deletes a boot entry from the firmware
func (m *EDK2Manager) DeleteBootEntry(id string) error {
	// Add "Boot" prefix if not present
	if !strings.HasPrefix(id, efi.BootPrefix) {
		id = efi.BootPrefix + id
	}

	// Check if the entry exists
	_, found := m.varList[id]
	if !found {
		return fmt.Errorf("boot entry not found: %s", id)
	}

	// Remove the entry from the boot order
	bootOrder, err := m.GetBootOrder()
	if err != nil {
		return fmt.Errorf("failed to get boot order: %w", err)
	}

	// Extract numeric ID from the boot entry
	idStr := strings.TrimPrefix(id, efi.BootPrefix)

	// Remove the entry from the boot order
	newBootOrder := make([]string, 0, len(bootOrder))
	for _, orderID := range bootOrder {
		if orderID != idStr {
			newBootOrder = append(newBootOrder, orderID)
		}
	}

	// Update the boot order
	if err := m.SetBootOrder(newBootOrder); err != nil {
		return fmt.Errorf("failed to update boot order: %w", err)
	}

	// Delete the entry from the variable list
	delete(m.varList, id)

	return nil
}

// GetNetworkSettings returns the current network settings
func (m *EDK2Manager) GetNetworkSettings() (NetworkSettings, error) {
	settings := NetworkSettings{
		EnableDHCP: true, // Default to DHCP enabled
	}

	// Get MAC address
	macAddr, err := m.GetMacAddress()
	if err == nil && macAddr != nil {
		settings.MacAddress = macAddr.String()
	}

	// Get IPv6 enabled setting
	ipv6Var, found := m.varList["IPv6Support"]
	if found {
		ipv6Enabled, err := ipv6Var.GetUint32()
		if err == nil {
			settings.EnableIPv6 = ipv6Enabled != 0
		}
	}

	// Get VLAN settings
	vlanVar, found := m.varList["VLANEnable"]
	if found {
		vlanEnabled, err := vlanVar.GetUint32()
		if err == nil {
			settings.VLANEnabled = vlanEnabled != 0
		}
	}

	vlanIDVar, found := m.varList["VLANID"]
	if found {
		vlanID, err := vlanIDVar.GetUint32()
		if err == nil {
			settings.VLANID = fmt.Sprintf("%d", vlanID)
		}
	}

	return settings, nil
}

// SetNetworkSettings sets the network settings
func (m *EDK2Manager) SetNetworkSettings(settings NetworkSettings) error {
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

	// Set IPv6 support
	ipv6Var := m.getOrCreateVar("IPv6Support", efi.EFI_GLOBAL_VARIABLE)
	ipv6Var.SetUint32(boolToUint32(settings.EnableIPv6))

	// Set VLAN settings
	vlanVar := m.getOrCreateVar("VLANEnable", efi.EFI_GLOBAL_VARIABLE)
	vlanVar.SetUint32(boolToUint32(settings.VLANEnabled))

	if settings.VLANEnabled && settings.VLANID != "" {
		vlanID, err := strconv.ParseUint(settings.VLANID, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid VLAN ID: %w", err)
		}

		vlanIDVar := m.getOrCreateVar("VLANID", efi.EFI_GLOBAL_VARIABLE)
		vlanIDVar.SetUint32(uint32(vlanID))
	}

	return nil
}

// GetMacAddress retrieves the MAC address from the firmware
func (m *EDK2Manager) GetMacAddress() (net.HardwareAddr, error) {
	// Check for dedicated MAC address variable first
	macVar, found := m.varList["MacAddress"]
	if found {
		macStr := macVar.Name.String()
		return net.ParseMAC(macStr)
	}

	// Look for MAC address in boot entries
	entries, err := m.GetBootEntries()
	if err != nil {
		return nil, fmt.Errorf("failed to get boot entries: %w", err)
	}

	for _, entry := range entries {
		if strings.Contains(entry.Name, "MAC:") {
			macIndex := strings.Index(entry.Name, "MAC:")
			if macIndex >= 0 {
				macStr := entry.Name[macIndex+4:]
				macEnd := strings.Index(macStr, ")")
				if macEnd >= 0 {
					macStr = macStr[:macEnd]
				}

				// Try to parse the MAC address
				mac, err := net.ParseMAC(strings.ReplaceAll(macStr, ":", ""))
				if err == nil {
					return mac, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("MAC address not found")
}

func (m *EDK2Manager) SetDefaultBootEntries() error {
	entries, err := m.GetBootEntries()
	if err != nil {
		return fmt.Errorf("failed to get boot entries: %w", err)
	}
	if len(entries) > 0 {
		return nil
	}

	defaultBootEntries := []BootEntry{
		{
			Name:    "UiApp",
			DevPath: "FvName(9a15aa37-d555-4a4e-b541-86391ff68164)/FvFileName(462caa21-7614-4503-836e-8ab6f4662331)",
		},
		{
			Name:    "SD/MMC on Arasan SDHCI",
			DevPath: "VendorHW(100c2cfa-b586-4198-9b4c-1683d195b1da)",
			OptData: "4eac0881119f594d850ee21a522c59b2",
		},
		{
			Name:    "UEFI SupTronics X862 202101000087",
			DevPath: "ACPI(hid=0xa0841d0,uid=0x0)/PCI(dev=00:0)/PCI(dev=00:0)/USB(port=2)",
			OptData: "4eac0881119f594d850ee21a522c59b2",
		},
		{
			Name:    "UEFI PXEv4 (MAC:000000000000)",
			DevPath: "MAC()/IPv4()",
			OptData: "4eac0881119f594d850ee21a522c59b2",
		},
		{
			Name:    "UEFI PXEv6 (MAC:000000000000)",
			DevPath: "MAC()/IPv6()",
			OptData: "4eac0881119f594d850ee21a522c59b2",
		},
		{
			Name:    "UEFI HTTPv4 (MAC:000000000000)",
			DevPath: "MAC()/IPv4()/URI()",
			OptData: "4eac0881119f594d850ee21a522c59b2",
		},
		{
			Name:    "UEFI HTTPv6 (MAC:000000000000)",
			DevPath: "MAC()/IPv6()/URI()",
			OptData: "4eac0881119f594d850ee21a522c59b2",
		},
		{
			Name:    "UEFI Shell",
			DevPath: "FvName(9a15aa37-d555-4a4e-b541-86391ff68164)/FvFileName(7c04a583-9e3e-4f1c-ad65-e05268d0b4d1)",
		},
	}

	for _, entry := range defaultBootEntries {
		if err := m.AddBootEntry(entry); err != nil {
			return fmt.Errorf("failed to add default boot entry: %w", err)
		}
	}
	return nil
}

// SetMacAddress sets the MAC address in the firmware
func (m *EDK2Manager) SetMacAddress(mac net.HardwareAddr) error {
	if mac == nil {
		return fmt.Errorf("MAC address is nil")
	}

	// Format MAC address without colons
	macStr := strings.ToUpper(strings.ReplaceAll(mac.String(), ":", ""))

	clientId := m.getOrCreateVar("ClientId", efi.EfiDhcp6ServiceBindingProtocol)
	clientIdStr := fmt.Sprintf("120000041531c000000000000000%s", strings.ToLower(macStr))
	clientId.SetString(clientIdStr)

	ndl := m.getOrCreateVar("_NDL", "e622443c-284e-4b47-a984-fd66b482dac0")
	ndl.Attr = efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS
	uniqueID := macStr[len(macStr)/2:]
	ndl.SetString(
		fmt.Sprintf(
			"030b2500d83add%s0000000000000000000000000000000000000000000000000000017fff0400",
			strings.ToLower(uniqueID),
		),
	)

	// Set the dedicated MAC address variable
	_ = m.getOrCreateVar(macStr, efi.EfiIp6ConfigProtocol)

	err := m.SetDefaultBootEntries()
	if err != nil {
		return fmt.Errorf("failed to set default boot entries: %w", err)
	}

	// Update MAC address in boot entries
	entries, err := m.GetBootEntries()
	if err != nil {
		return fmt.Errorf("failed to get boot entries: %w", err)
	}

	for _, entry := range entries {
		if strings.Contains(entry.Name, "MAC:") {
			// Replace the MAC address in the entry name
			newName := replaceMAC(entry.Name, macStr)
			if newName != entry.Name {
				entry.Name = newName
				if err := m.UpdateBootEntry(entry.ID, entry); err != nil {
					return fmt.Errorf("failed to update boot entry %s: %w", entry.ID, err)
				}
			}
		}
	}

	return nil
}

// GetVariable retrieves a variable by name
func (m *EDK2Manager) GetVariable(name string) (*efi.EfiVar, error) {
	v, found := m.varList[name]
	if !found {
		return nil, fmt.Errorf("variable not found: %s", name)
	}
	return v, nil
}

// SetVariable sets a variable
func (m *EDK2Manager) SetVariable(name string, value *efi.EfiVar) error {
	if value == nil {
		return fmt.Errorf("variable is nil")
	}
	m.varList[name] = value
	return nil
}

// ListVariables returns all variables in the firmware
func (m *EDK2Manager) ListVariables() (map[string]*efi.EfiVar, error) {
	return m.varList, nil
}

// EnablePXEBoot enables or disables PXE boot
func (m *EDK2Manager) EnablePXEBoot(enable bool) error {
	// Get all boot entries
	entries, err := m.GetBootEntries()
	if err != nil {
		return fmt.Errorf("failed to get boot entries: %w", err)
	}

	// Find PXE boot entries
	pxeEntries := make([]BootEntry, 0)
	for _, entry := range entries {
		if strings.Contains(entry.Name, "PXE") {
			entry.Enabled = enable
			pxeEntries = append(pxeEntries, entry)
		}
	}

	// Update PXE boot entries
	for _, entry := range pxeEntries {
		if err := m.UpdateBootEntry(entry.ID, entry); err != nil {
			return fmt.Errorf("failed to update PXE boot entry %s: %w", entry.ID, err)
		}
	}

	// If we need to enable PXE and no entries were found, create one
	if enable && len(pxeEntries) == 0 {
		mac, err := m.GetMacAddress()
		if err != nil {
			mac = net.HardwareAddr{0, 0, 0, 0, 0, 0}
		}

		macStr := strings.ToUpper(strings.ReplaceAll(mac.String(), ":", ""))

		// Create IPv4 PXE entry
		pxeEntry := BootEntry{
			Name:     fmt.Sprintf("UEFI PXEv4 (MAC:%s)", macStr),
			DevPath:  "MAC()/IPv4()",
			Enabled:  true,
			Position: 0, // Set as first boot option
		}

		if err := m.AddBootEntry(pxeEntry); err != nil {
			return fmt.Errorf("failed to add PXE boot entry: %w", err)
		}
	}

	return nil
}

// EnableHTTPBoot enables or disables HTTP boot
func (m *EDK2Manager) EnableHTTPBoot(enable bool) error {
	// Get all boot entries
	entries, err := m.GetBootEntries()
	if err != nil {
		return fmt.Errorf("failed to get boot entries: %w", err)
	}

	// Find HTTP boot entries
	httpEntries := make([]BootEntry, 0)
	for _, entry := range entries {
		if strings.Contains(entry.Name, "HTTP") {
			entry.Enabled = enable
			httpEntries = append(httpEntries, entry)
		}
	}

	// Update HTTP boot entries
	for _, entry := range httpEntries {
		if err := m.UpdateBootEntry(entry.ID, entry); err != nil {
			return fmt.Errorf("failed to update HTTP boot entry %s: %w", entry.ID, err)
		}
	}

	// If we need to enable HTTP boot and no entries were found, create one
	if enable && len(httpEntries) == 0 {
		mac, err := m.GetMacAddress()
		if err != nil {
			mac = net.HardwareAddr{0, 0, 0, 0, 0, 0}
		}

		macStr := strings.ToUpper(strings.ReplaceAll(mac.String(), ":", ""))

		// Create IPv4 HTTP entry
		httpEntry := BootEntry{
			Name:     fmt.Sprintf("UEFI HTTPv4 (MAC:%s)", macStr),
			DevPath:  "MAC()/IPv4()/URI()",
			Enabled:  true,
			Position: 1, // Set as second boot option
		}

		if err := m.AddBootEntry(httpEntry); err != nil {
			return fmt.Errorf("failed to add HTTP boot entry: %w", err)
		}
	}

	return nil
}

// SetFirmwareTimeoutSeconds sets the boot menu timeout in seconds
func (m *EDK2Manager) SetFirmwareTimeoutSeconds(seconds int) error {
	// The timeout is stored as a 16-bit value in the Timeout variable
	timeoutVar := m.getOrCreateVar("Timeout", efi.EFI_GLOBAL_VARIABLE)

	// Convert seconds to the format expected by the firmware
	data := []byte{byte(seconds & 0xFF), byte((seconds >> 8) & 0xFF)}
	timeoutVar.Data = data

	return nil
}

// SetConsoleConfig sets the console configuration
func (m *EDK2Manager) SetConsoleConfig(consoleName string, baudRate int) error {
	// Update the console preference variable
	consoleVar := m.getOrCreateVar("ConsolePref", "2d2358b4-e96c-484d-b2dd-7c2edfc7d56f")

	// Set console preference based on name
	var prefValue uint32
	switch strings.ToLower(consoleName) {
	case "serial":
		prefValue = 1
	case "graphics":
		prefValue = 2
	default:
		prefValue = 0 // Auto
	}

	consoleVar.SetUint32(prefValue)

	// Update baud rate if serial console is selected
	if prefValue == 1 && baudRate > 0 {
		baudVar := m.getOrCreateVar("SerialBaudRate", "cd7cc258-31db-22e6-9f22-63b0b8eed6b5")
		baudVar.SetUint32(uint32(baudRate))
	}

	return nil
}

// GetSystemInfo returns information about the system
func (m *EDK2Manager) GetSystemInfo() (map[string]string, error) {
	info := make(map[string]string)

	// Add firmware version
	version, err := m.GetFirmwareVersion()
	if err == nil {
		info["FirmwareVersion"] = version
	}

	// Try to get asset tag
	assetVar, found := m.varList["AssetTag"]
	if found {
		info["AssetTag"] = string(assetVar.Data)
	}

	// Get CPU settings
	cpuVar, found := m.varList["CpuClock"]
	if found {
		cpuVal, err := cpuVar.GetUint32()
		if err == nil {
			info["CpuClock"] = fmt.Sprintf("%d", cpuVal)
		}
	}

	// Add RAM information
	ramVar, found := m.varList["RamMoreThan3GB"]
	if found {
		ramVal, err := ramVar.GetUint32()
		if err == nil {
			if ramVal != 0 {
				info["RAM"] = "More than 3GB"
			} else {
				info["RAM"] = "3GB or less"
			}
		}
	}

	// Add system table mode
	sysTableVar, found := m.varList["SystemTableMode"]
	if found {
		sysTableVal, err := sysTableVar.GetUint32()
		if err == nil {
			info["SystemTableMode"] = fmt.Sprintf("%d", sysTableVal)
		}
	}

	return info, nil
}

// UpdateFirmware updates the firmware with the provided data
func (m *EDK2Manager) UpdateFirmware(firmwareData []byte) error {
	// Backup the original firmware
	backupPath := m.firmwarePath + ".backup"
	if err := copyFile(m.firmwarePath, backupPath); err != nil {
		return fmt.Errorf("failed to backup firmware: %w", err)
	}

	// Create a temporary file with the new firmware
	tempPath := m.firmwarePath + ".new"
	if err := writeFile(tempPath, firmwareData); err != nil {
		return fmt.Errorf("failed to write new firmware: %w", err)
	}

	// Validate the new firmware
	tempVarStore := varstore.NewEdk2VarStore(tempPath)
	tempVarStore.Logger = m.logger
	_, err := tempVarStore.GetVarList()
	if err != nil {
		// Clean up the temporary file
		removeFile(tempPath)
		return fmt.Errorf("invalid firmware data: %w", err)
	}

	m.logger.Info("firmware updated successfully", "path", tempPath)

	return nil
}

// GetFirmwareVersion returns the firmware version
func (m *EDK2Manager) GetFirmwareVersion() (string, error) {
	// Try to extract version from embedded firmware info
	var version string

	// Get the data from the FirmwareRevision variable if it exists
	revVar, found := m.varList["FirmwareRevision"]
	if found {
		version = string(revVar.Data)
	}

	// If no version found, use the firmware file modification time
	if version == "" {
		fileInfo, err := getFileInfo(m.firmwarePath)
		if err == nil {
			modTime := fileInfo.ModTime()
			version = fmt.Sprintf("Unknown (Modified: %s)", modTime.Format(time.RFC3339))
		} else {
			version = "Unknown"
		}
	}

	return version, nil
}

// SaveChanges writes the modified variables back to the firmware file
func (m *EDK2Manager) SaveChanges() error {
	// Create a new file with the updated variables
	newPath := m.firmwarePath + ".new"

	if err := m.varStore.WriteVarStore(newPath, m.varList); err != nil {
		return fmt.Errorf("failed to write variable store: %w", err)
	}

	m.logger.Info("firmware saved successfully", "path", newPath)

	return nil
}

// RevertChanges discards all changes
func (m *EDK2Manager) RevertChanges() error {
	// Reload the variables from the file
	var err error
	m.varList, err = m.varStore.GetVarList()
	if err != nil {
		return fmt.Errorf("failed to reload variable list: %w", err)
	}

	return nil
}

// ResetToDefaults resets the firmware to default settings
func (m *EDK2Manager) ResetToDefaults() error {
	// Reset the boot timeout
	timeoutVar := m.getOrCreateVar("Timeout", efi.EFI_GLOBAL_VARIABLE)
	timeoutVar.Data = []byte{0x05, 0x00} // 5 seconds

	// Reset console preference
	consoleVar := m.getOrCreateVar("ConsolePref", "2d2358b4-e96c-484d-b2dd-7c2edfc7d56f")
	consoleVar.SetUint32(0) // Auto

	// Reset the boot order to defaults
	defaultBootOrder := []string{"0000", "0001"} // UiApp, SD/MMC
	if err := m.SetBootOrder(defaultBootOrder); err != nil {
		return fmt.Errorf("failed to reset boot order: %w", err)
	}

	// Reset network settings
	ipv6Var := m.getOrCreateVar("IPv6Support", efi.EFI_GLOBAL_VARIABLE)
	ipv6Var.SetUint32(0) // Disable IPv6

	vlanVar := m.getOrCreateVar("VLANEnable", efi.EFI_GLOBAL_VARIABLE)
	vlanVar.SetUint32(0) // Disable VLAN

	return nil
}

// Helper functions

// getOrCreateVar gets an existing variable or creates a new one with the specified name and GUID
func (m *EDK2Manager) getOrCreateVar(name, guidStr string) *efi.EfiVar {
	v, found := m.varList[name]
	if found {
		return v
	}

	// Create a new variable
	v = &efi.EfiVar{
		Name: efi.NewUCS16String(name),
		Guid: efi.StringToGUID(guidStr),
		Attr: efi.EFI_VARIABLE_NON_VOLATILE |
			efi.EFI_VARIABLE_BOOTSERVICE_ACCESS |
			efi.EFI_VARIABLE_RUNTIME_ACCESS,
	}
	m.varList[name] = v

	return v
}

// boolToUint32 converts a boolean to a uint32 (0 or 1)
func boolToUint32(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// replaceMAC replaces a MAC address in a string
func replaceMAC(s, newMAC string) string {
	// Find the MAC address in the string
	macIndex := strings.Index(s, "MAC:")
	if macIndex < 0 {
		return s
	}

	// Extract the MAC portion
	macStart := macIndex + 4
	macEnd := strings.Index(s[macStart:], ")")
	if macEnd < 0 {
		return s
	}

	macEnd += macStart

	// Replace the MAC address
	return s[:macStart] + newMAC + s[macEnd:]
}

// File utility functions
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", src, err)
	}
	return os.WriteFile(dst, data, 0o644)
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func removeFile(path string) error {
	return os.Remove(path)
}

func getFileInfo(path string) (os.FileInfo, error) {
	return os.Stat(path)
}
