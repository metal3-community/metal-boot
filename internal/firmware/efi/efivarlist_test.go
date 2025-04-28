package efi

import (
	"reflect"
	"testing"
)

func TestNewEfiVarList(t *testing.T) {
	tests := []struct {
		name string
		want EfiVarList
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEfiVarList(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEfiVarList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVarList_Create(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		l       EfiVarList
		args    args
		want    *EfiVar
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.l.Create(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EfiVarList.Create() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVarList_Delete(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		l    EfiVarList
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Delete(tt.args.name)
		})
	}
}

func TestEfiVarList_SetBool(t *testing.T) {
	type args struct {
		name  string
		value bool
	}
	tests := []struct {
		name    string
		l       EfiVarList
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.l.SetBool(tt.args.name, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.SetBool() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEfiVarList_SetUint32(t *testing.T) {
	type args struct {
		name  string
		value uint32
	}
	tests := []struct {
		name    string
		l       EfiVarList
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.l.SetUint32(tt.args.name, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.SetUint32() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEfiVarList_SetBootEntry(t *testing.T) {
	type args struct {
		index   uint16
		title   string
		path    string
		optdata []byte
	}
	tests := []struct {
		name    string
		l       EfiVarList
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.l.SetBootEntry(tt.args.index, tt.args.title, tt.args.path, tt.args.optdata); (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.SetBootEntry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEfiVarList_AddBootEntry(t *testing.T) {
	type args struct {
		title   string
		path    string
		optdata []byte
	}
	tests := []struct {
		name    string
		l       EfiVarList
		args    args
		want    uint16
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.l.AddBootEntry(tt.args.title, tt.args.path, tt.args.optdata)
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.AddBootEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EfiVarList.AddBootEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVarList_GetBootNext(t *testing.T) {
	tests := []struct {
		name    string
		l       EfiVarList
		want    uint16
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.l.GetBootNext()
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.GetBootNext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EfiVarList.GetBootNext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVarList_SetBootNext(t *testing.T) {
	type args struct {
		index uint16
	}
	tests := []struct {
		name    string
		l       EfiVarList
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.l.SetBootNext(tt.args.index); (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.SetBootNext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEfiVarList_SetBootOrder(t *testing.T) {
	type args struct {
		order []uint16
	}
	tests := []struct {
		name    string
		l       EfiVarList
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.l.SetBootOrder(tt.args.order); (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.SetBootOrder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEfiVarList_AppendBootOrder(t *testing.T) {
	type args struct {
		index uint16
	}
	tests := []struct {
		name    string
		l       EfiVarList
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.l.AppendBootOrder(tt.args.index); (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.AppendBootOrder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEfiVarList_GetBootOrder(t *testing.T) {
	tests := []struct {
		name    string
		l       EfiVarList
		want    []uint16
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.l.GetBootOrder()
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.GetBootOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EfiVarList.GetBootOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVarList_SetFromFile(t *testing.T) {
	type args struct {
		name     string
		filename string
	}
	tests := []struct {
		name    string
		l       EfiVarList
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.l.SetFromFile(tt.args.name, tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.SetFromFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEfiVarList_GetBootEntry(t *testing.T) {
	type args struct {
		index uint16
	}
	tests := []struct {
		name    string
		l       EfiVarList
		args    args
		want    *BootEntry
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.l.GetBootEntry(tt.args.index)
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.GetBootEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EfiVarList.GetBootEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVarList_ListBootEntries(t *testing.T) {
	tests := []struct {
		name    string
		l       EfiVarList
		want    map[uint16]*BootEntry
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.l.ListBootEntries()
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.ListBootEntries() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EfiVarList.ListBootEntries() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVarList_DeleteBootEntry(t *testing.T) {
	type args struct {
		index uint16
	}
	tests := []struct {
		name    string
		l       EfiVarList
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.l.DeleteBootEntry(tt.args.index); (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.DeleteBootEntry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEfiVarList_FindFirst(t *testing.T) {
	type args struct {
		predicate func(name string, efiVar *EfiVar) bool
	}
	tests := []struct {
		name  string
		l     EfiVarList
		args  args
		want  *EfiVar
		want1 string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.l.FindFirst(tt.args.predicate)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EfiVarList.FindFirst() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("EfiVarList.FindFirst() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestEfiVarList_Variables(t *testing.T) {
	tests := []struct {
		name string
		l    EfiVarList
		want []*EfiVar
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.Variables(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EfiVarList.Variables() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVarList_FindByPrefix(t *testing.T) {
	type args struct {
		prefix string
	}
	tests := []struct {
		name string
		l    EfiVarList
		args args
		want []*EfiVar
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.FindByPrefix(tt.args.prefix); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EfiVarList.FindByPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}
