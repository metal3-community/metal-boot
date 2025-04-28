// Package manager provides implementations for firmware management interfaces.
package manager

import (
	"net"
	"reflect"
	"testing"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/bmcpi/pibmc/internal/firmware/types"
	"github.com/bmcpi/pibmc/internal/firmware/varstore"
	"github.com/go-logr/logr"
)

func TestNewEDK2Manager(t *testing.T) {
	type args struct {
		firmwarePath string
		logger       logr.Logger
	}
	tests := []struct {
		name    string
		args    args
		want    *EDK2Manager
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEDK2Manager(tt.args.firmwarePath, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEDK2Manager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEDK2Manager() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEDK2Manager_GetBootOrder(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			got, err := m.GetBootOrder()
			if (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.GetBootOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EDK2Manager.GetBootOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEDK2Manager_SetBootOrder(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		bootOrder []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.SetBootOrder(tt.args.bootOrder); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.SetBootOrder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEDK2Manager_GetBootEntries(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    []types.BootEntry
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			got, err := m.GetBootEntries()
			if (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.GetBootEntries() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EDK2Manager.GetBootEntries() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEDK2Manager_AddBootEntry(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		entry types.BootEntry
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.AddBootEntry(tt.args.entry); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.AddBootEntry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_parseHexString(t *testing.T) {
	type args struct {
		hexStr string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHexString(tt.args.hexStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHexString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseHexString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEDK2Manager_UpdateBootEntry(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		id    string
		entry types.BootEntry
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.UpdateBootEntry(tt.args.id, tt.args.entry); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.UpdateBootEntry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEDK2Manager_DeleteBootEntry(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.DeleteBootEntry(tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.DeleteBootEntry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEDK2Manager_SetBootNext(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		index uint16
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.SetBootNext(tt.args.index); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.SetBootNext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEDK2Manager_GetBootNext(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    uint16
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			got, err := m.GetBootNext()
			if (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.GetBootNext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EDK2Manager.GetBootNext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEDK2Manager_GetNetworkSettings(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    types.NetworkSettings
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			got, err := m.GetNetworkSettings()
			if (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.GetNetworkSettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EDK2Manager.GetNetworkSettings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEDK2Manager_SetNetworkSettings(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		settings types.NetworkSettings
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.SetNetworkSettings(tt.args.settings); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.SetNetworkSettings() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_boolToByte(t *testing.T) {
	type args struct {
		b bool
	}
	tests := []struct {
		name string
		args args
		want byte
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := boolToByte(tt.args.b); got != tt.want {
				t.Errorf("boolToByte() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseIPv4String(t *testing.T) {
	type args struct {
		ipStr string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIPv4String(tt.args.ipStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIPv4String() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseIPv4String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEDK2Manager_setOrCreateVariable(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		name string
		guid efi.Guid
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.setOrCreateVariable(tt.args.name, tt.args.guid, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.setOrCreateVariable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEDK2Manager_GetMacAddress(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    net.HardwareAddr
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			got, err := m.GetMacAddress()
			if (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.GetMacAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EDK2Manager.GetMacAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEDK2Manager_SetMacAddress(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		mac net.HardwareAddr
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.SetMacAddress(tt.args.mac); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.SetMacAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEDK2Manager_GetVariable(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *efi.EfiVar
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			got, err := m.GetVariable(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.GetVariable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EDK2Manager.GetVariable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEDK2Manager_SetVariable(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		name  string
		value *efi.EfiVar
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.SetVariable(tt.args.name, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.SetVariable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEDK2Manager_ListVariables(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    map[string]*efi.EfiVar
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			got, err := m.ListVariables()
			if (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.ListVariables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EDK2Manager.ListVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEDK2Manager_EnablePXEBoot(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		enable bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.EnablePXEBoot(tt.args.enable); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.EnablePXEBoot() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getNextAvailableBootID(t *testing.T) {
	type args struct {
		entries []types.BootEntry
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNextAvailableBootID(tt.args.entries); got != tt.want {
				t.Errorf("getNextAvailableBootID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEDK2Manager_EnableHTTPBoot(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		enable bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.EnableHTTPBoot(tt.args.enable); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.EnableHTTPBoot() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEDK2Manager_SetFirmwareTimeoutSeconds(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		seconds int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.SetFirmwareTimeoutSeconds(tt.args.seconds); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.SetFirmwareTimeoutSeconds() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEDK2Manager_SetConsoleConfig(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		consoleName string
		baudRate    int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.SetConsoleConfig(tt.args.consoleName, tt.args.baudRate); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.SetConsoleConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEDK2Manager_GetSystemInfo(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    types.SystemInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			got, err := m.GetSystemInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.GetSystemInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EDK2Manager.GetSystemInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEDK2Manager_GetFirmwareVersion(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			got, err := m.GetFirmwareVersion()
			if (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.GetFirmwareVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EDK2Manager.GetFirmwareVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEDK2Manager_UpdateFirmware(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	type args struct {
		firmwareData []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.UpdateFirmware(tt.args.firmwareData); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.UpdateFirmware() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEDK2Manager_SaveChanges(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.SaveChanges(); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.SaveChanges() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEDK2Manager_RevertChanges(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.RevertChanges(); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.RevertChanges() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEDK2Manager_ResetToDefaults(t *testing.T) {
	type fields struct {
		firmwarePath string
		varStore     *varstore.Edk2VarStore
		varList      efi.EfiVarList
		logger       logr.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EDK2Manager{
				firmwarePath: tt.fields.firmwarePath,
				varStore:     tt.fields.varStore,
				varList:      tt.fields.varList,
				logger:       tt.fields.logger,
			}
			if err := m.ResetToDefaults(); (err != nil) != tt.wantErr {
				t.Errorf("EDK2Manager.ResetToDefaults() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
