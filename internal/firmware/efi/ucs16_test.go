package efi

import (
	"reflect"
	"testing"
)

func TestNewUCS16String(t *testing.T) {
	type args struct {
		string []string
	}
	tests := []struct {
		name string
		args args
		want *UCS16String
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewUCS16String(tt.args.string...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUCS16String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUCS16String_ParseBin(t *testing.T) {
	type fields struct {
		data []byte
	}
	type args struct {
		data   []byte
		offset int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &UCS16String{
				data: tt.fields.data,
			}
			s.ParseBin(tt.args.data, tt.args.offset)
		})
	}
}

func TestUCS16String_ParseStr(t *testing.T) {
	type fields struct {
		data []byte
	}
	type args struct {
		str string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &UCS16String{
				data: tt.fields.data,
			}
			s.ParseStr(tt.args.str)
		})
	}
}

func TestUCS16String_Bytes(t *testing.T) {
	type fields struct {
		data []byte
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
			s := &UCS16String{
				data: tt.fields.data,
			}
			if got := s.Bytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UCS16String.Bytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUCS16String_Size(t *testing.T) {
	type fields struct {
		data []byte
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &UCS16String{
				data: tt.fields.data,
			}
			if got := s.Size(); got != tt.want {
				t.Errorf("UCS16String.Size() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUCS16String_String(t *testing.T) {
	type fields struct {
		data []byte
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
			s := &UCS16String{
				data: tt.fields.data,
			}
			if got := s.String(); got != tt.want {
				t.Errorf("UCS16String.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUCS16String_GoString(t *testing.T) {
	type fields struct {
		data []byte
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
			s := &UCS16String{
				data: tt.fields.data,
			}
			if got := s.GoString(); got != tt.want {
				t.Errorf("UCS16String.GoString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromUCS16(t *testing.T) {
	type args struct {
		data   []byte
		offset []int
	}
	tests := []struct {
		name string
		args args
		want *UCS16String
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromUCS16(tt.args.data, tt.args.offset...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromUCS16() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromString(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want *UCS16String
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromString(tt.args.str); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToUCS16(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want *UCS16String
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToUCS16(tt.args.str); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToUCS16() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUcs16ToString(t *testing.T) {
	type args struct {
		s *UCS16String
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
			if got := Ucs16ToString(tt.args.s); got != tt.want {
				t.Errorf("Ucs16ToString() = %v, want %v", got, tt.want)
			}
		})
	}
}
