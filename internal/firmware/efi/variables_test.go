package efi_test

import (
	"testing"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEfiVarInitialization(t *testing.T) {
	// Test creating a new EFI variable
	testVar := efi.NewEfiVar("TestVar", efi.EfiGlobalVariable, []byte{0x01, 0x02, 0x03, 0x04})

	assert.Equal(t, "TestVar", testVar.Name)
	assert.Equal(t, efi.EfiGlobalVariable, testVar.GuidStr)
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, testVar.Data)
	assert.Equal(t, uint32(4), testVar.DataSize)
	assert.Equal(t, efi.EFI_VARIABLE_NON_VOLATILE|efi.EFI_VARIABLE_BOOTSERVICE_ACCESS, testVar.Attrs)
}

func TestEfiVarWithAttributes(t *testing.T) {
	// Test creating a variable with specific attributes
	attrs := efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_RUNTIME_ACCESS
	testVar := efi.NewEfiVarWithAttrs("TestVar", efi.EfiGlobalVariable, []byte{0x01, 0x02, 0x03, 0x04}, attrs)

	assert.Equal(t, "TestVar", testVar.Name)
	assert.Equal(t, efi.EfiGlobalVariable, testVar.GuidStr)
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, testVar.Data)
	assert.Equal(t, uint32(4), testVar.DataSize)
	assert.Equal(t, attrs, testVar.Attrs)
}

func TestEfiVarDefaultAttributes(t *testing.T) {
	// Test the default attributes for common EFI variables

	// SecureBoot should have specific attributes
	secureBootVar := efi.NewEfiVar("SecureBoot", efi.EfiGlobalVariable, []byte{0x01})
	assert.Equal(t, efi.EFI_VARIABLE_BOOTSERVICE_ACCESS|efi.EFI_VARIABLE_RUNTIME_ACCESS, secureBootVar.Attrs)

	// Custom variable should have default attributes
	customVar := efi.NewEfiVar("CustomVar", efi.EfiGlobalVariable, []byte{0x01})
	assert.Equal(t, efi.EFI_VARIABLE_NON_VOLATILE|efi.EFI_VARIABLE_BOOTSERVICE_ACCESS, customVar.Attrs)
}

func TestEfiVarCopy(t *testing.T) {
	// Test copying an EFI variable
	original := efi.NewEfiVar("TestVar", efi.EfiGlobalVariable, []byte{0x01, 0x02, 0x03, 0x04})

	// Create a copy
	copy := original.Copy()

	// Verify the copy is independent
	assert.Equal(t, original.Name, copy.Name)
	assert.Equal(t, original.GuidStr, copy.GuidStr)
	assert.Equal(t, original.Data, copy.Data)
	assert.Equal(t, original.DataSize, copy.DataSize)
	assert.Equal(t, original.Attrs, copy.Attrs)

	// Modify the copy and verify original is unchanged
	copy.Name = "ModifiedVar"
	copy.Data = []byte{0xFF, 0xFF}
	copy.DataSize = 2
	copy.Attrs = efi.EFI_VARIABLE_RUNTIME_ACCESS

	assert.Equal(t, "TestVar", original.Name)
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, original.Data)
	assert.Equal(t, uint32(4), original.DataSize)
	assert.Equal(t, efi.EFI_VARIABLE_NON_VOLATILE|efi.EFI_VARIABLE_BOOTSERVICE_ACCESS, original.Attrs)
}

func TestEfiVarBootOrder(t *testing.T) {
	// Test creating and parsing BootOrder variable
	bootOrder := []uint16{1, 2, 3, 4}
	bootOrderData := efi.CreateBootOrderData(bootOrder)

	// Verify the data is correct
	assert.Equal(t, []byte{0x01, 0x00, 0x02, 0x00, 0x03, 0x00, 0x04, 0x00}, bootOrderData)

	// Create the variable
	bootOrderVar := efi.NewEfiVar("BootOrder", efi.EfiGlobalVariable, bootOrderData)

	// Parse the variable
	parsedBootOrder, err := efi.ParseBootOrder(bootOrderVar.Data)
	require.NoError(t, err)
	assert.Equal(t, bootOrder, parsedBootOrder)
}

func TestEfiVarBootOrderErrors(t *testing.T) {
	// Test parsing BootOrder with invalid data

	// Test with nil data
	_, err := efi.ParseBootOrder(nil)
	assert.Error(t, err)

	// Test with empty data
	_, err = efi.ParseBootOrder([]byte{})
	assert.Error(t, err)

	// Test with incomplete data
	_, err = efi.ParseBootOrder([]byte{0x01})
	assert.Error(t, err)

	// Test with odd-length data
	_, err = efi.ParseBootOrder([]byte{0x01, 0x00, 0x02})
	assert.Error(t, err)
}

func TestEfiVarBootEntry(t *testing.T) {
	// Test creating and parsing a Boot#### variable

	// Create a sample boot entry
	bootEntry := &efi.BootEntry{
		Description:  "Test Boot Entry",
		DevicePath:   []byte{0x01, 0x02, 0x03, 0x04},
		OptionalData: []byte{0xAA, 0xBB},
		Active:       true,
		Category:     efi.LOAD_OPTION_CATEGORY_BOOT,
	}

	// Convert to bytes
	bootEntryData, err := bootEntry.ToBytes()
	require.NoError(t, err)

	// Create the variable
	bootEntryVar := efi.NewEfiVar("Boot0001", efi.EfiGlobalVariable, bootEntryData)

	// Parse the variable
	parsedBootEntry, err := efi.ParseBootEntry(bootEntryVar.Data)
	require.NoError(t, err)

	// Verify the parsed data
	assert.Equal(t, bootEntry.Description, parsedBootEntry.Description)
	assert.Equal(t, bootEntry.DevicePath, parsedBootEntry.DevicePath)
	assert.Equal(t, bootEntry.OptionalData, parsedBootEntry.OptionalData)
	assert.Equal(t, bootEntry.Active, parsedBootEntry.Active)
	assert.Equal(t, bootEntry.Category, parsedBootEntry.Category)
}

func TestEfiVarAttributes(t *testing.T) {
	// Test EFI variable attribute operations

	// Create a variable with attributes
	attrs := efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS
	testVar := efi.NewEfiVarWithAttrs("TestVar", efi.EfiGlobalVariable, []byte{0x01}, attrs)

	// Test attribute checks
	assert.True(t, testVar.HasAttr(efi.EFI_VARIABLE_NON_VOLATILE))
	assert.True(t, testVar.HasAttr(efi.EFI_VARIABLE_BOOTSERVICE_ACCESS))
	assert.False(t, testVar.HasAttr(efi.EFI_VARIABLE_RUNTIME_ACCESS))

	// Test adding an attribute
	testVar.AddAttr(efi.EFI_VARIABLE_RUNTIME_ACCESS)
	assert.True(t, testVar.HasAttr(efi.EFI_VARIABLE_RUNTIME_ACCESS))

	// Test removing an attribute
	testVar.RemoveAttr(efi.EFI_VARIABLE_BOOTSERVICE_ACCESS)
	assert.False(t, testVar.HasAttr(efi.EFI_VARIABLE_BOOTSERVICE_ACCESS))

	// Test full attributes value
	assert.Equal(t, efi.EFI_VARIABLE_NON_VOLATILE|efi.EFI_VARIABLE_RUNTIME_ACCESS, testVar.Attrs)
}

func TestEfiVarSecurityAttributes(t *testing.T) {
	// Test security-related attributes

	// Create a secure variable with time-based authentication
	attrs := efi.EFI_VARIABLE_NON_VOLATILE | efi.EFI_VARIABLE_BOOTSERVICE_ACCESS | efi.EFI_VARIABLE_TIME_BASED_AUTHENTICATED_WRITE_ACCESS
	secureVar := efi.NewEfiVarWithAttrs("SecureVar", efi.EfiGlobalVariable, []byte{0x01}, attrs)

	// Verify the security attributes
	assert.True(t, secureVar.HasAttr(efi.EFI_VARIABLE_TIME_BASED_AUTHENTICATED_WRITE_ACCESS))
	assert.False(t, secureVar.HasAttr(efi.EFI_VARIABLE_AUTHENTICATED_WRITE_ACCESS)) // Deprecated attribute

	// Test with deprecated authenticated write access
	secureVar.AddAttr(efi.EFI_VARIABLE_AUTHENTICATED_WRITE_ACCESS)
	assert.True(t, secureVar.HasAttr(efi.EFI_VARIABLE_AUTHENTICATED_WRITE_ACCESS))

	// Test with hardware error record
	secureVar.AddAttr(efi.EFI_VARIABLE_HARDWARE_ERROR_RECORD)
	assert.True(t, secureVar.HasAttr(efi.EFI_VARIABLE_HARDWARE_ERROR_RECORD))
}

func TestEfiVarData(t *testing.T) {
	// Test data operations

	// Create a variable with data
	initialData := []byte{0x01, 0x02, 0x03, 0x04}
	testVar := efi.NewEfiVar("TestVar", efi.EfiGlobalVariable, initialData)

	// Verify initial data
	assert.Equal(t, initialData, testVar.Data)
	assert.Equal(t, uint32(4), testVar.DataSize)

	// Update data
	newData := []byte{0xAA, 0xBB, 0xCC}
	testVar.SetData(newData)

	// Verify updated data
	assert.Equal(t, newData, testVar.Data)
	assert.Equal(t, uint32(3), testVar.DataSize)

	// Test with empty data
	testVar.SetData([]byte{})
	assert.Empty(t, testVar.Data)
	assert.Equal(t, uint32(0), testVar.DataSize)

	// Test with nil data
	testVar.SetData(nil)
	assert.Empty(t, testVar.Data)
	assert.Equal(t, uint32(0), testVar.DataSize)
}

func TestEfiVarGuids(t *testing.T) {
	// Test GUID handling

	// Create a variable with a known GUID
	testVar := efi.NewEfiVar("TestVar", efi.EfiGlobalVariable, []byte{0x01})
	assert.Equal(t, efi.EfiGlobalVariable, testVar.GuidStr)

	// Update the GUID
	testVar.GuidStr = efi.EfiImageSecurityDatabaseGUID
	assert.Equal(t, efi.EfiImageSecurityDatabaseGUID, testVar.GuidStr)

	// Verify GUID bytes
	guidBytes, err := efi.GUIDStringToBytes(testVar.GuidStr)
	require.NoError(t, err)

	expectedGuidBytes, err := efi.GUIDStringToBytes(efi.EfiImageSecurityDatabaseGUID)
	require.NoError(t, err)

	assert.Equal(t, expectedGuidBytes, guidBytes)
}
