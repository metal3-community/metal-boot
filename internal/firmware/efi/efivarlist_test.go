package efi_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEfiVarListBasic(t *testing.T) {
	// Create a test EFI variable list
	varList := efi.NewEfiVarList()
	assert.NotNil(t, varList)
	assert.Empty(t, varList.Variables())

	// Test adding a variable
	testVar := &efi.EfiVar{
		Name:     "TestVar",
		GuidStr:  efi.EfiGlobalVariableGUID,
		Data:     []byte{0x01, 0x02, 0x03, 0x04},
		Attrs:    efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS,
		DataSize: 4,
	}
	err := varList.Add(testVar)
	assert.NoError(t, err)

	// Test getting the variable back
	retrievedVar, err := varList.Get("TestVar")
	assert.NoError(t, err)
	assert.Equal(t, testVar, retrievedVar)

	// Test listing all variables
	vars := varList.Variables()
	assert.Len(t, vars, 1)
	assert.Equal(t, "TestVar", vars[0].Name)

	// Test deleting a variable
	err = varList.Delete("TestVar")
	assert.NoError(t, err)
	assert.Empty(t, varList.Variables())

	// Test deleting a non-existent variable
	err = varList.Delete("NonExistentVar")
	assert.Error(t, err)
}

func TestEfiVarListLoadFromBytes(t *testing.T) {
	// Create a binary representation of EFI variables
	var buf bytes.Buffer

	// Write header size (uint32)
	binary.Write(&buf, binary.LittleEndian, uint32(4))

	// Write variable 1
	nameBytes := append([]byte("BootOrder"), make([]byte, 32-len("BootOrder"))...)
	buf.Write(nameBytes)
	binary.Write(&buf, binary.LittleEndian, uint32(4)) // DataSize
	binary.Write(&buf, binary.LittleEndian, uint32(efi.EFI_VARIABLE_NON_VOLATILE|efi.EFI_VARIABLE_BOOTSERVICE_ACCESS)) // Attrs
	guidBytes := make([]byte, 16)
	copy(guidBytes, []byte(efi.EfiGlobalVariableGUID))
	buf.Write(guidBytes)
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // BootOrder data
	binary.Write(&buf, binary.LittleEndian, uint16(2)) // BootOrder data

	// Write variable 2
	nameBytes = append([]byte("Boot0001"), make([]byte, 32-len("Boot0001"))...)
	buf.Write(nameBytes)
	binary.Write(&buf, binary.LittleEndian, uint32(8)) // DataSize
	binary.Write(&buf, binary.LittleEndian, uint32(efi.EFI_VARIABLE_NON_VOLATILE|efi.EFI_VARIABLE_BOOTSERVICE_ACCESS)) // Attrs
	buf.Write(guidBytes)
	buf.Write([]byte("UEFI OS\x00")) // Boot0001 data

	// Load from bytes
	varList := efi.NewEfiVarList()
	err := varList.LoadFromBytes(buf.Bytes())
	require.NoError(t, err)

	// Test that variables were loaded correctly
	bootOrder, err := varList.Get("BootOrder")
	assert.NoError(t, err)
	assert.Equal(t, "BootOrder", bootOrder.Name)
	assert.Equal(t, efi.EfiGlobalVariableGUID, bootOrder.GuidStr)
	assert.Equal(t, uint32(4), bootOrder.DataSize)
	assert.Equal(t, efi.EFI_VARIABLE_NON_VOLATILE|efi.EFI_VARIABLE_BOOTSERVICE_ACCESS, bootOrder.Attrs)
	assert.Equal(t, []byte{1, 0, 2, 0}, bootOrder.Data)

	boot0001, err := varList.Get("Boot0001")
	assert.NoError(t, err)
	assert.Equal(t, "Boot0001", boot0001.Name)
	assert.Equal(t, efi.EfiGlobalVariableGUID, boot0001.GuidStr)
	assert.Equal(t, uint32(8), boot0001.DataSize)
	assert.Equal(t, efi.EFI_VARIABLE_NON_VOLATILE|efi.EFI_VARIABLE_BOOTSERVICE_ACCESS, boot0001.Attrs)
	assert.Equal(t, []byte("UEFI OS\x00"), boot0001.Data)
}

func TestEfiVarListSaveToBytes(t *testing.T) {
	// Create a variable list with test variables
	varList := efi.NewEfiVarList()

	bootOrderVar := &efi.EfiVar{
		Name:     "BootOrder",
		GuidStr:  efi.EfiGlobalVariableGUID,
		Data:     []byte{1, 0, 2, 0},
		Attrs:    efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS,
		DataSize: 4,
	}

	boot0001Var := &efi.EfiVar{
		Name:     "Boot0001",
		GuidStr:  efi.EfiGlobalVariableGUID,
		Data:     []byte("UEFI OS\x00"),
		Attrs:    efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS,
		DataSize: 8,
	}

	require.NoError(t, varList.Add(bootOrderVar))
	require.NoError(t, varList.Add(boot0001Var))

	// Save to bytes
	data, err := varList.SaveToBytes()
	require.NoError(t, err)

	// Load back from bytes to verify
	newVarList := efi.NewEfiVarList()
	err = newVarList.LoadFromBytes(data)
	require.NoError(t, err)

	// Verify loaded variables match original ones
	loadedBootOrder, err := newVarList.Get("BootOrder")
	assert.NoError(t, err)
	assert.Equal(t, bootOrderVar.Name, loadedBootOrder.Name)
	assert.Equal(t, bootOrderVar.GuidStr, loadedBootOrder.GuidStr)
	assert.Equal(t, bootOrderVar.DataSize, loadedBootOrder.DataSize)
	assert.Equal(t, bootOrderVar.Attrs, loadedBootOrder.Attrs)
	assert.Equal(t, bootOrderVar.Data, loadedBootOrder.Data)

	loadedBoot0001, err := newVarList.Get("Boot0001")
	assert.NoError(t, err)
	assert.Equal(t, boot0001Var.Name, loadedBoot0001.Name)
	assert.Equal(t, boot0001Var.GuidStr, loadedBoot0001.GuidStr)
	assert.Equal(t, boot0001Var.DataSize, loadedBoot0001.DataSize)
	assert.Equal(t, boot0001Var.Attrs, loadedBoot0001.Attrs)
	assert.Equal(t, boot0001Var.Data, loadedBoot0001.Data)
}

func TestEfiVarListErrors(t *testing.T) {
	varList := efi.NewEfiVarList()

	// Test adding a variable with an empty name
	emptyNameVar := &efi.EfiVar{
		Name:     "",
		GuidStr:  efi.EfiGlobalVariableGUID,
		Data:     []byte{0x01},
		DataSize: 1,
	}
	err := varList.Add(emptyNameVar)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "variable name cannot be empty")

	// Test adding a variable with an empty GUID
	emptyGuidVar := &efi.EfiVar{
		Name:     "TestVar",
		GuidStr:  "",
		Data:     []byte{0x01},
		DataSize: 1,
	}
	err = varList.Add(emptyGuidVar)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "variable GUID cannot be empty")

	// Test adding a variable with nil data
	nilDataVar := &efi.EfiVar{
		Name:     "TestVar",
		GuidStr:  efi.EfiGlobalVariableGUID,
		Data:     nil,
		DataSize: 0,
	}
	err = varList.Add(nilDataVar)
	assert.NoError(t, err) // This should be valid, empty variables are allowed

	// Test getting a non-existent variable
	_, err = varList.Get("NonExistentVar")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "variable not found")

	// Test loading from invalid bytes
	err = varList.LoadFromBytes([]byte{0x01, 0x02})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid variable list")
}

func TestEfiVarListUpdate(t *testing.T) {
	varList := efi.NewEfiVarList()

	// Add a test variable
	testVar := &efi.EfiVar{
		Name:     "TestVar",
		GuidStr:  efi.EfiGlobalVariableGUID,
		Data:     []byte{0x01, 0x02},
		Attrs:    efi.EFI_VARIABLE_NON_VOLATILE,
		DataSize: 2,
	}
	err := varList.Add(testVar)
	require.NoError(t, err)

	// Update the variable
	updatedVar := &efi.EfiVar{
		Name:     "TestVar",
		GuidStr:  efi.EfiGlobalVariableGUID,
		Data:     []byte{0x03, 0x04, 0x05},
		Attrs:    efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS,
		DataSize: 3,
	}
	err = varList.Update("TestVar", updatedVar)
	assert.NoError(t, err)

	// Verify the update
	retrievedVar, err := varList.Get("TestVar")
	assert.NoError(t, err)
	assert.Equal(t, updatedVar.Name, retrievedVar.Name)
	assert.Equal(t, updatedVar.GuidStr, retrievedVar.GuidStr)
	assert.Equal(t, updatedVar.Data, retrievedVar.Data)
	assert.Equal(t, updatedVar.Attrs, retrievedVar.Attrs)
	assert.Equal(t, updatedVar.DataSize, retrievedVar.DataSize)

	// Test updating a non-existent variable
	err = varList.Update("NonExistentVar", updatedVar)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "variable not found")
}

func TestEfiVarListFind(t *testing.T) {
	varList := efi.NewEfiVarList()

	// Add test variables
	bootOrderVar := &efi.EfiVar{
		Name:     "BootOrder",
		GuidStr:  efi.EfiGlobalVariableGUID,
		Data:     []byte{1, 0, 2, 0},
		Attrs:    efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS,
		DataSize: 4,
	}

	boot0001Var := &efi.EfiVar{
		Name:     "Boot0001",
		GuidStr:  efi.EfiGlobalVariableGUID,
		Data:     []byte("UEFI OS\x00"),
		Attrs:    efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS,
		DataSize: 8,
	}

	boot0002Var := &efi.EfiVar{
		Name:     "Boot0002",
		GuidStr:  efi.EfiGlobalVariableGUID,
		Data:     []byte("PXE Boot\x00"),
		Attrs:    efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS,
		DataSize: 9,
	}

	customVar := &efi.EfiVar{
		Name:     "CustomVar",
		GuidStr:  "12345678-1234-1234-1234-123456789012",
		Data:     []byte{0x01},
		Attrs:    efi.EFI_VARIABLE_NON_VOLATILE,
		DataSize: 1,
	}

	require.NoError(t, varList.Add(bootOrderVar))
	require.NoError(t, varList.Add(boot0001Var))
	require.NoError(t, varList.Add(boot0002Var))
	require.NoError(t, varList.Add(customVar))

	// Test FindByPrefix
	bootVars := varList.FindByPrefix("Boot")
	assert.Len(t, bootVars, 3) // BootOrder, Boot0001, Boot0002
	
	// Test FindByGUID
	globalVars := varList.FindByGUID(efi.EfiGlobalVariableGUID)
	assert.Len(t, globalVars, 3) // All Boot* variables use the global GUID
	
	customGuidVars := varList.FindByGUID("12345678-1234-1234-1234-123456789012")
	assert.Len(t, customGuidVars, 1) // Only CustomVar uses this GUID
	
	// Test FindByNameAndGUID
	bootOrderByNameAndGuid := varList.FindByNameAndGUID("BootOrder", efi.EfiGlobalVariableGUID)
	assert.Len(t, bootOrderByNameAndGuid, 1)
	assert.Equal(t, "BootOrder", bootOrderByNameAndGuid[0].Name)
	
	// Test with non-existent values
	nonExistentByPrefix := varList.FindByPrefix("NonExistent")
	assert.Empty(t, nonExistentByPrefix)
	
	nonExistentByGuid := varList.FindByGUID("00000000-0000-0000-0000-000000000000")
	assert.Empty(t, nonExistentByGuid)
	
	nonExistentByNameAndGuid := varList.FindByNameAndGUID("NonExistent", efi.EfiGlobalVariableGUID)
	assert.Empty(t, nonExistentByNameAndGuid)
}
