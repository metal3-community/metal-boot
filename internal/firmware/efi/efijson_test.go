package efi

import (
	"reflect"
	"testing"
	"time"
)

func TestJSONEncoder_MarshalEfiVar(t *testing.T) {
	type args struct {
		v *EfiVar
	}
	tests := []struct {
		name string
		e    *jsonEncoder
		args args
		want efiVarJSON
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &jsonEncoder{}
			if got := e.MarshalEfiVar(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("JSONEncoder.MarshalEfiVar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONEncoder_MarshalEfiVarList(t *testing.T) {
	type args struct {
		list EfiVarList
	}
	tests := []struct {
		name string
		e    *jsonEncoder
		args args
		want efiVarListJSON
	}{
		{
			name: "MarshalEfiVarList",
			e:    &jsonEncoder{},
			args: args{
				list: EfiVarList{
					"test": &EfiVar{
						Name:  NewUCS16String("test"),
						Guid:  EFI_GLOBAL_VARIABLE_GUID,
						Attr:  0,
						Data:  []byte("test"),
						Count: 0,
						Time:  nil,
					},
				},
			},
			want: efiVarListJSON{
				Version: 2,
				Variables: []efiVarJSON{
					{
						Name: "test",
						GUID: EFI_GLOBAL_VARIABLE,
						Attr: 0,
						Data: "74657374",
						Time: "",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &jsonEncoder{}
			if got := e.MarshalEfiVarList(tt.args.list); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("JSONEncoder.MarshalEfiVarList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_MarshalJSON(t *testing.T) {
	type fields struct {
		Name  string
		Guid  string
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guid, e := GUIDFromString(tt.fields.Guid)
			if e != nil {
				t.Errorf("GUIDFromString() error = %v", e)
				return
			}
			v := &EfiVar{
				Name:  NewUCS16String(tt.fields.Name),
				Guid:  guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
			}
			got, err := v.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVar.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EfiVar.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVarList_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		list    EfiVarList
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.list.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EfiVarList.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_UnmarshalJSON(t *testing.T) {
	type fields struct {
		Name  string
		Guid  string
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
	}
	type args struct {
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
			guid, e := GUIDFromString(tt.fields.Guid)
			if e != nil {
				t.Errorf("GUIDFromString() error = %v", e)
				return
			}
			v := &EfiVar{
				Name:  NewUCS16String(tt.fields.Name),
				Guid:  guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
			}
			if err := v.UnmarshalJSON(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("EfiVar.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEfiVarList_UnmarshalJSON(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		list    *EfiVarList
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.list.UnmarshalJSON(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("EfiVarList.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecodeEfiJSON(t *testing.T) {
	type args struct {
		data []byte
		v    *efiVarJSON
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DecodeEfiJSON(tt.args.data, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("DecodeEfiJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
