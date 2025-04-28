package varstore_test

import (
	"testing"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/bmcpi/pibmc/internal/firmware/varstore"
	"github.com/stretchr/testify/assert"
)

// MockVarStore implements the VarStore interface for testing
type MockVarStore struct {
	varList     efi.EfiVarList
	writeErrors bool
}

func NewMockVarStore(writeErrors bool) *MockVarStore {
	return &MockVarStore{
		varList:     efi.NewEfiVarList(),
		writeErrors: writeErrors,
	}
}

func (m *MockVarStore) GetVarList() (efi.EfiVarList, error) {
	return m.varList, nil
}

func (m *MockVarStore) WriteVarStore(filename string, varlist efi.EfiVarList) error {
	if m.writeErrors {
		return assert.AnError
	}
	m.varList = varlist
	return nil
}

func TestVarStoreInterface(t *testing.T) {
	// Test that our mock implements the interface
	var _ varstore.VarStore = &MockVarStore{}
	
	// Create a mock varstore
	mock := NewMockVarStore(false)
	
	// Create some test variables
	varList := efi.NewEfiVarList()
	
	bootOrderVar := efi.NewEfiVar("BootOrder", efi.EfiGlobalVariableGUID, []byte{0x01, 0x00, 0x02, 0x00})
	assert.NoError(t, varList.Add(bootOrderVar))
	
	// Write to the mock varstore
	err := mock.WriteVarStore("test.bin", varList)
	assert.NoError(t, err)
	
	// Get the variable list back
	readVarList, err := mock.GetVarList()
	assert.NoError(t, err)
	
	// Verify the variables
	readVars := readVarList.Variables()
	assert.Len(t, readVars, 1)
	assert.Equal(t, "BootOrder", readVars[0].Name)
	assert.Equal(t, efi.EfiGlobalVariableGUID, readVars[0].GuidStr)
	assert.Equal(t, []byte{0x01, 0x00, 0x02, 0x00}, readVars[0].Data)
	
	// Test with write errors
	mockWithErrors := NewMockVarStore(true)
	err = mockWithErrors.WriteVarStore("test.bin", varList)
	assert.Error(t, err)
}
