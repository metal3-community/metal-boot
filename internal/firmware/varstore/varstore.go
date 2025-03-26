package varstore

import "github.com/bmcpi/pibmc/internal/firmware/efi"

type VarStore interface {
	GetVarList() efi.EfiVarList
	WriteVarStore(filename string, varlist efi.EfiVarList) error
}
