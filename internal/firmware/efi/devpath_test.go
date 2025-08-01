package efi

import (
	"reflect"
	"testing"
)

func TestGuid_String(t *testing.T) {
	type fields struct {
		BytesLe []byte
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
			g, err := GUIDFromBytes(tt.fields.BytesLe)
			if err != nil {
				t.Errorf("GUIDFromBytes() error = %v", err)
				return
			}
			if got := g.String(); got != tt.want {
				t.Errorf("Guid.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_guidsParseStr(t *testing.T) {
	type args struct {
		guidStr string
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
			got, err := ParseGUID(tt.args.guidStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("guidsParseStr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("guidsParseStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_guidsParseBin(t *testing.T) {
	type args struct {
		data   []byte
		offset int
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
			got := ParseBinGUID(tt.args.data, tt.args.offset)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("guidsParseBin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ucs16FromString(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ucs16FromString(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ucs16FromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ucs16FromUcs16(t *testing.T) {
	type args struct {
		data   []byte
		offset int
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
			if got := ucs16FromUcs16(tt.args.data, tt.args.offset); got != tt.want {
				t.Errorf("ucs16FromUcs16() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDevicePathElem(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want *DevicePathElem
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDevicePathElem(tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDevicePathElem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePathElem_set_mac(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			dpe.set_mac()
		})
	}
}

func TestDevicePathElem_set_ipv4(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			dpe.set_ipv4()
		})
	}
}

func TestDevicePathElem_set_ipv6(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			dpe.set_ipv6()
		})
	}
}

func TestDevicePathElem_set_iscsi(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
	}
	type args struct {
		target string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "test_iscsi",
			fields: fields{
				Devtype: DevTypeMessage,
				Subtype: DevSubTypeISCSI,
				Data:    make([]byte, 0),
			},
			args: args{
				target: "iqn.1994-05.com.redhat:example",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			dpe.set_iscsi(tt.args.target)
		})
	}
}

func TestDevicePathElem_set_sata(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
	}
	type args struct {
		port uint16
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			dpe.set_sata(tt.args.port)
		})
	}
}

func TestDevicePathElem_set_usb(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
	}
	type args struct {
		port uint8
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			dpe.set_usb(tt.args.port)
		})
	}
}

func TestDevicePathElem_set_uri(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
	}
	type args struct {
		uri string
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			dpe.set_uri(tt.args.uri)
		})
	}
}

func TestDevicePathElem_set_filepath(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
	}
	type args struct {
		filepath string
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			dpe.set_filepath(tt.args.filepath)
		})
	}
}

func TestDevicePathElem_set_fvname(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
	}
	type args struct {
		guid string
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			dpe.set_fvname(tt.args.guid)
		})
	}
}

func TestDevicePathElem_set_fvfilename(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
	}
	type args struct {
		guid string
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			dpe.set_fvfilename(tt.args.guid)
		})
	}
}

func TestDevicePathElem_set_gpt(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
	}
	type args struct {
		pnr  uint32
		poff uint64
		plen uint64
		guid string
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			dpe.set_gpt(tt.args.pnr, tt.args.poff, tt.args.plen, tt.args.guid)
		})
	}
}

func TestDevicePathElem_fmt_hw(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			if got := dpe.fmt_hw(); got != tt.want {
				t.Errorf("DevicePathElem.fmt_hw() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePathElem_fmt_acpi(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			if got := dpe.fmt_acpi(); got != tt.want {
				t.Errorf("DevicePathElem.fmt_acpi() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePathElem_fmt_msg(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			if got := dpe.fmt_msg(); got != tt.want {
				t.Errorf("DevicePathElem.fmt_msg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePathElem_fmt_media(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			if got := dpe.fmt_media(); got != tt.want {
				t.Errorf("DevicePathElem.fmt_media() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePathElem_size(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			if got := dpe.size(); got != tt.want {
				t.Errorf("DevicePathElem.size() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePathElem_Bytes(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			if got := dpe.Bytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePathElem.Bytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePathElem_String(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			if got := dpe.String(); got != tt.want {
				t.Errorf("DevicePathElem.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePathElem_Equal(t *testing.T) {
	type fields struct {
		Devtype DeviceType
		Subtype DeviceSubType
		Data    []byte
	}
	type args struct {
		other *DevicePathElem
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
			dpe := &DevicePathElem{
				Devtype: tt.fields.Devtype,
				Subtype: tt.fields.Subtype,
				Data:    tt.fields.Data,
			}
			if got := dpe.Equal(tt.args.other); got != tt.want {
				t.Errorf("DevicePathElem.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_VendorHW(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	type args struct {
		guid GUID
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.VendorHW(tt.args.guid); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePath.VendorHW() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_Mac(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	tests := []struct {
		name   string
		fields fields
		want   *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.Mac(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePath.Mac() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_IPv4(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	tests := []struct {
		name   string
		fields fields
		want   *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.IPv4(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePath.IPv4() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_IPv6(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	tests := []struct {
		name   string
		fields fields
		want   *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.IPv6(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePath.IPv6() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_ISCSI(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	type args struct {
		target string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.ISCSI(tt.args.target); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePath.ISCSI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_SATA(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	type args struct {
		port uint16
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.SATA(tt.args.port); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePath.SATA() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_USB(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	type args struct {
		port uint8
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.USB(tt.args.port); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePath.USB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_FvName(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	type args struct {
		guid string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.FvName(tt.args.guid); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePath.FvName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_FVFileName(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	type args struct {
		guid string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.FVFileName(tt.args.guid); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePath.FVFileName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_FilePath(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	type args struct {
		filepath string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.FilePath(tt.args.filepath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePath.FilePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_GptPartition(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	type args struct {
		pnr  uint32
		poff uint64
		plen uint64
		guid string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.GptPartition(tt.args.pnr, tt.args.poff, tt.args.plen, tt.args.guid); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("DevicePath.GptPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_Append(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	type args struct {
		elem *DevicePathElem
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.Append(tt.args.elem); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePath.Append() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDevicePath(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDevicePath(tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDevicePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePathUri(t *testing.T) {
	type args struct {
		uri string
	}
	tests := []struct {
		name string
		args args
		want *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DevicePathUri(tt.args.uri); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePathUri() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePathFilepath(t *testing.T) {
	type args struct {
		filepath string
	}
	tests := []struct {
		name string
		args args
		want *DevicePath
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DevicePathFilepath(tt.args.filepath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePathFilepath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDevicePath(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *DevicePath
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDevicePath(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDevicePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDevicePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_ParseFromString(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	type args struct {
		s string
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
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if err := dp.ParseFromString(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("DevicePath.ParseFromString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDevicePath_Bytes(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
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
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.Bytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevicePath.Bytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_String(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
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
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.String(); got != tt.want {
				t.Errorf("DevicePath.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevicePath_Equal(t *testing.T) {
	type fields struct {
		elems []*DevicePathElem
	}
	type args struct {
		other *DevicePath
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
			dp := &DevicePath{
				elems: tt.fields.elems,
			}
			if got := dp.Equal(tt.args.other); got != tt.want {
				t.Errorf("DevicePath.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDevicePathFromString(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    *DevicePath
		wantErr bool
	}{
		{
			name: "valid_device_path",
			args: args{
				s: "PciRoot(0x0)",
			},
			want: &DevicePath{
				elems: []*DevicePathElem{
					{
						Devtype: DevTypeHardware,
						Subtype: DevSubTypePCI,
						Data:    []byte{0x00, 0x00},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDevicePathFromString(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDevicePathFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDevicePathFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
