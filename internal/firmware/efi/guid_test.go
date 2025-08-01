package efi

import (
	"reflect"
	"testing"
)

func TestGuidName(t *testing.T) {
	type args struct {
		guid GUID
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
			if got := GuidName(tt.args.guid); got != tt.want {
				t.Errorf("GuidName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGUID_BytesLE(t *testing.T) {
	type fields struct {
		Data1 uint32
		Data2 uint16
		Data3 uint16
		Data4 [8]byte
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := GUID{
				Data1: tt.fields.Data1,
				Data2: tt.fields.Data2,
				Data3: tt.fields.Data3,
				Data4: tt.fields.Data4,
			}
			if got := g.BytesLE(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GUID.BytesLE() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseGUID(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    GUID
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGUID(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseGUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseGuid(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want GUID
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseGuid(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseGuid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewGUID(t *testing.T) {
	type args struct {
		data1 uint32
		data2 uint16
		data3 uint16
		data4 [8]byte
	}
	tests := []struct {
		name string
		args args
		want GUID
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewGUID(tt.args.data1, tt.args.data2, tt.args.data3, tt.args.data4); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("NewGUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGUIDFromString(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    GUID
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GUIDFromString(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("GUIDFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GUIDFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringToGUID(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want GUID
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringToGUID(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringToGUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGUIDFromBytes(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    GUID
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GUIDFromBytes(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("GUIDFromBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GUIDFromBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGUID_Bytes(t *testing.T) {
	type fields struct {
		Data1 uint32
		Data2 uint16
		Data3 uint16
		Data4 [8]byte
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := GUID{
				Data1: tt.fields.Data1,
				Data2: tt.fields.Data2,
				Data3: tt.fields.Data3,
				Data4: tt.fields.Data4,
			}
			if got := g.Bytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GUID.Bytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGUID_String(t *testing.T) {
	type fields struct {
		Data1 uint32
		Data2 uint16
		Data3 uint16
		Data4 [8]byte
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := GUID{
				Data1: tt.fields.Data1,
				Data2: tt.fields.Data2,
				Data3: tt.fields.Data3,
				Data4: tt.fields.Data4,
			}
			if got := g.String(); got != tt.want {
				t.Errorf("GUID.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseBinGUID(t *testing.T) {
	type args struct {
		data   []byte
		offset int
	}
	tests := []struct {
		name string
		args args
		want GUID
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseBinGUID(tt.args.data, tt.args.offset); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseBinGUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGUID_Equal(t *testing.T) {
	type fields struct {
		Data1 uint32
		Data2 uint16
		Data3 uint16
		Data4 [8]byte
	}
	type args struct {
		other GUID
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := GUID{
				Data1: tt.fields.Data1,
				Data2: tt.fields.Data2,
				Data3: tt.fields.Data3,
				Data4: tt.fields.Data4,
			}
			if got := g.Equal(tt.args.other); got != tt.want {
				t.Errorf("GUID.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}
