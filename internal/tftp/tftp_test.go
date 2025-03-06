package tftp

import (
	"context"
	"io"
	"net"
	"net/netip"
	"reflect"
	"testing"

	"github.com/bmcpi/pibmc/internal/dhcp/data"
	"github.com/bmcpi/pibmc/internal/dhcp/handler"
	"github.com/go-logr/logr"
	"github.com/pin/tftp/v3"
)

func TestHandler_OnSuccess(t *testing.T) {
	type fields struct {
		ctx           context.Context
		RootDirectory string
		Patch         string
		Log           logr.Logger
		backend       handler.BackendReader
	}
	type args struct {
		stats tftp.TransferStats
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
			h := Handler{
				ctx:           tt.fields.ctx,
				RootDirectory: tt.fields.RootDirectory,
				Patch:         tt.fields.Patch,
				Log:           tt.fields.Log,
				backend:       tt.fields.backend,
			}
			h.OnSuccess(tt.args.stats)
		})
	}
}

func TestHandler_OnFailure(t *testing.T) {
	type fields struct {
		ctx           context.Context
		RootDirectory string
		Patch         string
		Log           logr.Logger
		backend       handler.BackendReader
	}
	type args struct {
		stats tftp.TransferStats
		err   error
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
			h := Handler{
				ctx:           tt.fields.ctx,
				RootDirectory: tt.fields.RootDirectory,
				Patch:         tt.fields.Patch,
				Log:           tt.fields.Log,
				backend:       tt.fields.backend,
			}
			h.OnFailure(tt.args.stats, tt.args.err)
		})
	}
}

func TestServer_ListenAndServe(t *testing.T) {
	type fields struct {
		Logger        logr.Logger
		RootDirectory string
		Patch         string
		Log           logr.Logger
	}
	type args struct {
		ctx     context.Context
		addr    netip.AddrPort
		backend handler.BackendReader
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
			r := &Server{
				Logger:        tt.fields.Logger,
				RootDirectory: tt.fields.RootDirectory,
				Patch:         tt.fields.Patch,
				Log:           tt.fields.Log,
			}
			if err := r.ListenAndServe(tt.args.ctx, tt.args.addr, tt.args.backend); (err != nil) != tt.wantErr {
				t.Errorf("Server.ListenAndServe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServe(t *testing.T) {
	type args struct {
		in0  context.Context
		conn net.PacketConn
		s    *tftp.Server
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
			if err := Serve(tt.args.in0, tt.args.conn, tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("Serve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_getDhcpInfo(t *testing.T) {
	type fields struct {
		ctx           context.Context
		RootDirectory string
		Patch         string
		Log           logr.Logger
		backend       handler.BackendReader
	}
	type args struct {
		ctx context.Context
		rf  io.ReaderFrom
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *data.DHCP
		want1   *data.Netboot
		want2   *data.Power
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				ctx:           tt.fields.ctx,
				RootDirectory: tt.fields.RootDirectory,
				Patch:         tt.fields.Patch,
				Log:           tt.fields.Log,
				backend:       tt.fields.backend,
			}
			got, got1, got2, err := h.getDhcpInfo(tt.args.ctx, tt.args.rf)
			if (err != nil) != tt.wantErr {
				t.Errorf("Handler.getDhcpInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Handler.getDhcpInfo() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Handler.getDhcpInfo() got1 = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("Handler.getDhcpInfo() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func TestHandler_HandleRead(t *testing.T) {
	type fields struct {
		ctx           context.Context
		RootDirectory string
		Patch         string
		Log           logr.Logger
		backend       handler.BackendReader
	}
	type args struct {
		fullfilepath string
		rf           io.ReaderFrom
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
			h := &Handler{
				ctx:           tt.fields.ctx,
				RootDirectory: tt.fields.RootDirectory,
				Patch:         tt.fields.Patch,
				Log:           tt.fields.Log,
				backend:       tt.fields.backend,
			}
			if err := h.HandleRead(tt.args.fullfilepath, tt.args.rf); (err != nil) != tt.wantErr {
				t.Errorf("Handler.HandleRead() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_createFile(t *testing.T) {
	type fields struct {
		ctx           context.Context
		RootDirectory string
		Patch         string
		Log           logr.Logger
		backend       handler.BackendReader
	}
	type args struct {
		root     *Root
		filename string
		content  []byte
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
			h := &Handler{
				ctx:           tt.fields.ctx,
				RootDirectory: tt.fields.RootDirectory,
				Patch:         tt.fields.Patch,
				Log:           tt.fields.Log,
				backend:       tt.fields.backend,
			}
			if err := h.createFile(tt.args.root, tt.args.filename, tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("Handler.createFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_HandleIpxeRead(t *testing.T) {
	type fields struct {
		ctx           context.Context
		RootDirectory string
		Patch         string
		Log           logr.Logger
		backend       handler.BackendReader
	}
	type args struct {
		filename string
		rf       io.ReaderFrom
		content  []byte
		patch    string
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
			h := &Handler{
				ctx:           tt.fields.ctx,
				RootDirectory: tt.fields.RootDirectory,
				Patch:         tt.fields.Patch,
				Log:           tt.fields.Log,
				backend:       tt.fields.backend,
			}
			if err := h.HandleIpxeRead(tt.args.filename, tt.args.rf, tt.args.content, tt.args.patch); (err != nil) != tt.wantErr {
				t.Errorf("Handler.HandleIpxeRead() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_HandleWrite(t *testing.T) {
	type fields struct {
		ctx           context.Context
		RootDirectory string
		Patch         string
		Log           logr.Logger
		backend       handler.BackendReader
	}
	type args struct {
		filename string
		wt       io.WriterTo
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
			h := &Handler{
				ctx:           tt.fields.ctx,
				RootDirectory: tt.fields.RootDirectory,
				Patch:         tt.fields.Patch,
				Log:           tt.fields.Log,
				backend:       tt.fields.backend,
			}
			if err := h.HandleWrite(tt.args.filename, tt.args.wt); (err != nil) != tt.wantErr {
				t.Errorf("Handler.HandleWrite() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
