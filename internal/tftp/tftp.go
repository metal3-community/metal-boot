// Package tftp provides metal-boot's TFTP boot handler. The serving framework
// (ServeMux, Config.Serve, middleware) lives in the server/handler/middleware
// subpackages, ported from Tinkerbell's pkg/tftp; this file is the boot
// handler that serves iPXE binaries and filesystem files, registered as a
// handler.Handler on that ServeMux.
package tftp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/metal3-community/metal-boot/internal/backend"
	"github.com/metal3-community/metal-boot/internal/dhcp/data"
	"github.com/metal3-community/metal-boot/internal/ipxe/binary"
	"github.com/pin/tftp/v3"
)

// Handler serves iPXE binaries and filesystem files over TFTP. It implements
// handler.Handler (ServeTFTP).
type Handler struct {
	ctx           context.Context
	RootDirectory string
	Patch         string
	Log           logr.Logger
	backend       backend.BackendReader
}

// NewHandler creates a TFTP boot handler. backend may be nil; when set it is
// used, best effort, to look up a per-node iPXE script by client IP.
func NewHandler(
	ctx context.Context,
	log logr.Logger,
	be backend.BackendReader,
	rootDirectory string,
	patch string,
) (*Handler, error) {
	return &Handler{
		ctx:           ctx,
		RootDirectory: rootDirectory,
		Patch:         patch,
		Log:           log,
		backend:       be,
	}, nil
}

// ServeTFTP implements handler.Handler.
func (h *Handler) ServeTFTP(filename string, rf io.ReaderFrom) error {
	return h.HandleRead(filename, rf)
}

// HandleRead handles TFTP GET requests.
func (h *Handler) HandleRead(fullfilepath string, rf io.ReaderFrom) (err error) {
	defer func() {
		if r := recover(); r != nil {
			h.Log.Error(
				fmt.Errorf("panic in HandleRead: %v", r),
				"recovered from panic", "path", fullfilepath,
			)
			err = fmt.Errorf("internal error during TFTP read")
		}
	}()

	if rf == nil {
		return fmt.Errorf("nil ReaderFrom parameter")
	}

	filename := filepath.Base(fullfilepath)

	if filename == "autoexec.ipxe" {
		_, err = rf.ReadFrom(bytes.NewReader([]byte("#!ipxe\n\n")))
		return err
	}

	// Serve iPXE binaries, optionally patched with a per-node iPXE script.
	if content, ok := binary.Files[filename]; ok {
		patch := h.Patch
		if netboot := h.netbootFor(rf); netboot != nil && len(netboot.IPXEScript) > 1 {
			patch = netboot.IPXEScript
		}
		return h.serveIPXE(rf, content, patch)
	}

	// Serve from the filesystem.
	root, err := NewRoot(h.RootDirectory)
	if err != nil {
		return fmt.Errorf("failed to open root directory: %w", err)
	}
	defer root.Close()

	if file, err := root.Open(fullfilepath); err == nil {
		defer file.Close()
		_, err := rf.ReadFrom(file)
		return err
	}

	h.Log.Info("file not found", "path", fullfilepath)
	return os.ErrNotExist
}

// netbootFor best-effort looks up per-node netboot info by client IP. It
// returns nil when there is no backend or the lookup fails.
func (h *Handler) netbootFor(rf io.ReaderFrom) *data.Netboot {
	if h.backend == nil {
		return nil
	}
	ip, err := getRemoteIP(rf)
	if err != nil || ip == nil {
		return nil
	}
	_, netboot, err := h.backend.GetByIP(h.ctx, ip)
	if err != nil {
		h.Log.V(1).Info("no per-node netboot info", "ip", ip, "error", err)
		return nil
	}
	return netboot
}

func (h *Handler) serveIPXE(rf io.ReaderFrom, content []byte, patch string) error {
	if patch == "" {
		patch = h.Patch
	}
	patched, err := binary.Patch(content, []byte(patch))
	if err != nil {
		return fmt.Errorf("failed to patch iPXE binary: %w", err)
	}
	_, err = rf.ReadFrom(bytes.NewReader(patched))
	return err
}

func getRemoteIP(r any) (net.IP, error) {
	if r == nil {
		return nil, fmt.Errorf("transfer object is nil")
	}

	var remoteAddr net.Addr
	switch v := r.(type) {
	case tftp.OutgoingTransfer:
		addr := v.RemoteAddr()
		remoteAddr = &addr
	case tftp.IncomingTransfer:
		addr := v.RemoteAddr()
		remoteAddr = &addr
	default:
		return nil, fmt.Errorf("invalid TFTP transfer type: %T", r)
	}

	if remoteAddr == nil {
		return nil, fmt.Errorf("remote address is nil")
	}

	udpAddr, ok := remoteAddr.(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("address is not a UDP address: %T", remoteAddr)
	}

	if udpAddr.IP == nil {
		return nil, fmt.Errorf("UDP address IP is nil")
	}

	return udpAddr.IP, nil
}
