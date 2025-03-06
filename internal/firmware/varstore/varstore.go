package varstore

import (
	"log"
	"os"
	"path"
	"strings"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/bmcpi/pibmc/internal/util"
)

// EfiVariableStore represents an EDK2 EFI variable store
type EfiVariableStore struct {
	Filename string
	VarList  efi.EfiVarList
}

type EfiVariablesStore struct {
	RootPath string
	Stores   map[string]*EfiVariableStore
}

// NewEfiVariableStore creates a new Edk2VarStore from a file
func NewEfiVariableStore(rootPath string, macAddresses []string) (*EfiVariablesStore, error) {

	store := &EfiVariablesStore{
		RootPath: rootPath,
		Stores:   make(map[string]*EfiVariableStore),
	}

	for _, mac := range macAddresses {
		filename := path.Join(rootPath, mac, "RPI_EFI.fd")
		store.Stores[mac] = &EfiVariableStore{
			Filename: filename,
		}
		store.Stores[mac].ReadFile()
	}

	return store, nil
}

// ReadFile reads the raw EDK2 varstore from file
func (vs *EfiVariableStore) ReadFile() error {
	log.Printf("Reading raw edk2 varstore from %s", vs.Filename)

	fwVarsJsonFile := vs.getFwVarsJsonFile()

	_, err := VirtFwVars("-i", vs.Filename, "--output-json", fwVarsJsonFile)

	// _, err := runVirtFwVars("-i", vs.Filename, "--output-json", fwFile)
	if err != nil {
		return err
	}

	if util.Exists(fwVarsJsonFile) {
		b, err := os.ReadFile(fwVarsJsonFile)
		if err != nil {
			return err
		}

		if vs.VarList == nil {
			vs.VarList = make(efi.EfiVarList)
		}

		err = vs.VarList.UnmarshalJSON(b)
		if err != nil {
			return err
		}
	}

	return nil
}

func (vs *EfiVariableStore) getFwVarsJsonFile() string {
	return strings.Join([]string{path.Dir(vs.Filename), "fw-vars.json"}, string(os.PathSeparator))
}

// GetVarList gets the list of variables
func (vs *EfiVariableStore) GetVarList() efi.EfiVarList {
	return vs.VarList
}

// WriteVarStore writes the varstore to a file
func (vs *EfiVariableStore) WriteVarStore(filename string) error {

	log.Printf("Writing raw edk2 varstore to %s", filename)

	fwFile := vs.getFwVarsJsonFile()

	f, err := os.Open(fwFile)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := vs.VarList.MarshalJSON()
	if err != nil {
		return err
	}

	err = os.WriteFile(fwFile, b, 0755)
	if err != nil {
		return err
	}

	return SaveVirtFwVars(vs.Filename, fwFile)
}

// GetBootOrder retrieves the BootOrder variable
func (vs *EfiVariableStore) GetBootOrder() ([]uint16, error) {
	return vs.VarList.GetBootOrder()
}

// SetBootOrder sets the BootOrder variable
func (vs *EfiVariableStore) SetBootOrder(bootOrder []uint16) error {
	return vs.VarList.SetBootOrder(bootOrder)
}

// GetBootEntry retrieves a boot entry by its ID
func (vs *EfiVariableStore) GetBootEntry(id uint16) (*efi.BootEntry, error) {
	return vs.VarList.GetBootEntry(id)
}

// SetBootEntry sets a boot entry
func (vs *EfiVariableStore) SetBootEntry(id uint16, entry *efi.BootEntry) error {
	return vs.VarList.SetBootEntry(id, entry.Title.String(), entry.DevicePath.String(), entry.OptData)
}

// DeleteBootEntry deletes a boot entry
func (vs *EfiVariableStore) DeleteBootEntry(id uint16) error {
	return vs.VarList.DeleteBootEntry(id)
}

// ListBootEntries lists all boot entries
func (vs *EfiVariableStore) ListBootEntries() (map[uint16]*efi.BootEntry, error) {
	return vs.VarList.ListBootEntries()
}

// GetOrderedBootEntries returns boot entries in boot order
func (vs *EfiVariableStore) GetOrderedBootEntries() ([]*efi.BootEntry, error) {
	// Get boot order
	bootOrder, err := vs.GetBootOrder()
	if err != nil {
		return nil, err
	}

	// Get all boot entries
	allEntries, err := vs.ListBootEntries()
	if err != nil {
		return nil, err
	}

	// Create ordered list
	ordered := make([]*efi.BootEntry, 0, len(bootOrder))

	for _, id := range bootOrder {
		if entry, ok := allEntries[id]; ok {
			ordered = append(ordered, entry)
		}
	}

	return ordered, nil
}
