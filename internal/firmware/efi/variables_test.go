package efi

import (
	"reflect"
	"testing"
	"time"
)

func TestNewEfiVar(t *testing.T) {
	type args struct {
		name  any
		guid  *string
		attr  uint32
		data  []byte
		count int
	}
	tests := []struct {
		name    string
		args    args
		want    *EfiVar
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEfiVar(
				tt.args.name,
				tt.args.guid,
				tt.args.attr,
				tt.args.data,
				tt.args.count,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEfiVar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEfiVar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_ParseTime(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
	}
	type args struct {
		data   []byte
		offset int
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			if err := v.ParseTime(tt.args.data, tt.args.offset); (err != nil) != tt.wantErr {
				t.Errorf("EfiVar.ParseTime() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEfiVar_BytesTime(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			if got := v.BytesTime(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EfiVar.BytesTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_updateTime(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
	}
	type args struct {
		ts *time.Time
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			v.updateTime(tt.args.ts)
		})
	}
}

func TestEfiVar_SetBool(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
	}
	type args struct {
		value bool
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			v.SetBool(tt.args.value)
		})
	}
}

func TestEfiVar_SetString(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
	}
	type args struct {
		value string
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			v.SetString(tt.args.value)
		})
	}
}

func TestEfiVar_SetUint32(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
	}
	type args struct {
		value uint32
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			v.SetUint32(tt.args.value)
		})
	}
}

func TestEfiVar_GetUint32(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
	}
	tests := []struct {
		name    string
		fields  fields
		want    uint32
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			got, err := v.GetUint32()
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVar.GetUint32() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EfiVar.GetUint32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_GetBootEntry(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
	}
	tests := []struct {
		name    string
		fields  fields
		want    *BootEntry
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			got, err := v.GetBootEntry()
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVar.GetBootEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EfiVar.GetBootEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_SetBootEntry(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
	}
	type args struct {
		attr    uint32
		title   string
		path    string
		optdata []byte
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			if err := v.SetBootEntry(tt.args.attr, tt.args.title, tt.args.path, tt.args.optdata); (err != nil) != tt.wantErr {
				t.Errorf("EfiVar.SetBootEntry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEfiVar_GetBootNext(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			got, err := v.GetBootNext()
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVar.GetBootNext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EfiVar.GetBootNext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_SetBootNext(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
	}
	type args struct {
		index uint16
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			v.SetBootNext(tt.args.index)
		})
	}
}

func TestEfiVar_GetBootOrder(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
	}
	tests := []struct {
		name    string
		fields  fields
		want    []uint16
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			got, err := v.GetBootOrder()
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVar.GetBootOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EfiVar.GetBootOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_SetBootOrder(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
	}
	type args struct {
		order []uint16
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			v.SetBootOrder(tt.args.order)
		})
	}
}

func TestEfiVar_AppendBootOrder(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
	}
	type args struct {
		index uint16
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			v.AppendBootOrder(tt.args.index)
		})
	}
}

func TestEfiVar_SetFromFile(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
	}
	type args struct {
		filename string
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			if err := v.SetFromFile(tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("EfiVar.SetFromFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEfiVar_FmtBool(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			if got := v.FmtBool(); got != tt.want {
				t.Errorf("EfiVar.FmtBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_FmtAscii(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			if got := v.FmtAscii(); got != tt.want {
				t.Errorf("EfiVar.FmtAscii() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_FmtBootEntry(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			got, err := v.FmtBootEntry()
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVar.FmtBootEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EfiVar.FmtBootEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_FmtBootList(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			if got := v.FmtBootList(); got != tt.want {
				t.Errorf("EfiVar.FmtBootList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_FmtDevPath(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			got, err := v.FmtDevPath()
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVar.FmtDevPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EfiVar.FmtDevPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_FmtData(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			got, err := v.FmtData()
			if (err != nil) != tt.wantErr {
				t.Errorf("EfiVar.FmtData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EfiVar.FmtData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEfiVar_String(t *testing.T) {
	type fields struct {
		Name  *UCS16String
		Guid  GUID
		Attr  uint32
		Data  []byte
		Count int
		Time  *time.Time
		PkIdx int
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
			v := &EfiVar{
				Name:  tt.fields.Name,
				Guid:  tt.fields.Guid,
				Attr:  tt.fields.Attr,
				Data:  tt.fields.Data,
				Count: tt.fields.Count,
				Time:  tt.fields.Time,
				PkIdx: tt.fields.PkIdx,
			}
			if got := v.String(); got != tt.want {
				t.Errorf("EfiVar.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
