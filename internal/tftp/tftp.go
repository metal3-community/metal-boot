package tftp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmcpi/pibmc/internal/backend"
	"github.com/bmcpi/pibmc/internal/dhcp/data"
	"github.com/bmcpi/uefi-firmware-manager/edk2"
	"github.com/go-logr/logr"
	"github.com/pin/tftp/v3"
	"github.com/tinkerbell/ipxedust/binary"
)

type Server struct {
	Logger        logr.Logger
	RootDirectory string
	Patch         string
}

type Handler struct {
	ctx           context.Context
	RootDirectory string
	Patch         string
	Log           logr.Logger
	backend       backend.BackendReader
}

// ListenAndServe sets up the listener and serves TFTP requests.
func (s *Server) ListenAndServe(
	ctx context.Context,
	addr netip.AddrPort,
	backend backend.BackendReader,
) error {
	handler := &Handler{
		ctx:           ctx,
		RootDirectory: s.RootDirectory,
		Patch:         s.Patch,
		Log:           s.Logger,
		backend:       backend,
	}

	tftpServer := tftp.NewServer(handler.HandleRead, handler.HandleWrite)
	tftpServer.SetHook(handler)

	a, err := net.ResolveUDPAddr("udp", addr.String())
	if err != nil {
		return fmt.Errorf("(tftp) failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", a)
	if err != nil {
		return fmt.Errorf("(tftp) failed to listen on UDP: %w", err)
	}

	go func() {
		<-ctx.Done()
		s.Logger.Info("(tftp) shutting down tftp server")
		tftpServer.Shutdown()
	}()

	if err := tftpServer.Serve(conn); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.Logger.Error(err, "(tftp) TFTP server error")
		return err
	}

	return nil
}

func (h *Handler) OnSuccess(stats tftp.TransferStats) {
	h.Log.Info("(tftp) transfer complete", "remote", stats.RemoteAddr, "path", stats.Filename)
}

func (h *Handler) OnFailure(stats tftp.TransferStats, err error) {
	h.Log.Error(err, "(tftp) transfer failed", "remote", stats.RemoteAddr, "path", stats.Filename)
}

// HandleRead handles TFTP GET requests.
func (h *Handler) HandleRead(fullfilepath string, rf io.ReaderFrom) error {
	dhcpInfo, netboot, err := h.getDHCPInfo(rf)
	if err != nil {
		h.Log.Info("could not get DHCP info, proceeding without it", "error", err)
	}

	filename := filepath.Base(fullfilepath)

	// Serve iPXE binaries if requested
	if content, ok := binary.Files[filename]; ok {
		patch := h.Patch
		if netboot != nil && len(netboot.IPXEScript) > 1 {
			patch = netboot.IPXEScript
		}
		return h.serveIPXE(rf, content, patch)
	}

	// Resolve the file path, potentially swapping a serial for a MAC address
	resolvedPath := h.resolvePath(fullfilepath, dhcpInfo)

	// Serve from the filesystem if the file exists
	root, err := NewRoot(h.RootDirectory)
	if err != nil {
		return fmt.Errorf("failed to open root directory: %w", err)
	}
	defer root.Close()

	if file, err := root.Open(resolvedPath); err == nil {
		defer file.Close()
		_, err := rf.ReadFrom(file)
		return err
	}

	// If not on the filesystem, try serving from embedded EDK2 files
	if content, ok := edk2.Files[resolvedPath]; ok {
		return h.serveContent(rf, content)
	}

	// Handle generic paths for MAC-specific requests
	parts := strings.Split(resolvedPath, "/")
	if len(parts) > 1 {
		if _, err := net.ParseMAC(parts[0]); err == nil {
			genericPath := strings.Join(parts[1:], "/")
			if content, ok := edk2.Files[genericPath]; ok {
				return h.serveContent(rf, content)
			}
		}
	}

	h.Log.Info("file not found", "path", fullfilepath, "resolvedPath", resolvedPath)
	return os.ErrNotExist
}

// HandleWrite handles TFTP PUT requests.
func (h *Handler) HandleWrite(fullfilepath string, wt io.WriterTo) error {
	dhcpInfo, _, err := h.getDHCPInfo(wt)
	if err != nil {
		h.Log.Info("could not get DHCP info, proceeding without it", "error", err)
	}

	resolvedPath := h.resolvePath(fullfilepath, dhcpInfo)

	root, err := NewRoot(h.RootDirectory)
	if err != nil {
		return fmt.Errorf("failed to open root directory: %w", err)
	}
	defer root.Close()

	dir := filepath.Dir(resolvedPath)
	if !root.Exists(dir) {
		if err := root.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	file, err := root.OpenFile(resolvedPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open file for writing %s: %w", resolvedPath, err)
	}
	defer file.Close()

	n, err := wt.WriteTo(file)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", resolvedPath, err)
	}

	h.Log.Info("file written successfully", "path", resolvedPath, "bytes", n)
	return nil
}

func (h *Handler) getDHCPInfo(r any) (*data.DHCP, *data.Netboot, error) {
	ip, err := getRemoteIP(r)
	if err != nil {
		return nil, nil, err
	}

	dhcpInfo, netboot, _, err := h.backend.GetByIP(h.ctx, ip)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get info by IP %s: %w", ip, err)
	}
	if dhcpInfo == nil {
		return nil, nil, fmt.Errorf("no DHCP info found for IP: %s", ip)
	}

	return dhcpInfo, netboot, nil
}

func (h *Handler) resolvePath(fullfilepath string, dhcpInfo *data.DHCP) string {
	parts := strings.Split(fullfilepath, "/")

	if len(parts) < 2 {
		return fullfilepath
	}

	prefix := parts[0]
	filename := parts[len(parts)-1]

	mac := dhcpInfo.MACAddress.String()
	mac = strings.ReplaceAll(mac, ":", "-")

	isSerial, _ := regexp.MatchString(`^\d{2}[a-z]\d{5}$`, prefix)
	if isSerial && dhcpInfo != nil {
		if filename == "RPI_EFI.fd" {
			return strings.Replace(fullfilepath, prefix, mac, 1)
		} else {
			return strings.Join(parts[1:], "/")
		}
	}

	return fullfilepath
}

func (h *Handler) serveIPXE(rf io.ReaderFrom, content []byte, patch string) error {
	if patch == "" {
		patch = h.Patch
	}

	patchedContent, err := binary.Patch(content, []byte(patch))
	if err != nil {
		return fmt.Errorf("failed to patch iPXE binary: %w", err)
	}

	return h.serveContent(rf, patchedContent)
}

func (h *Handler) serveContent(rf io.ReaderFrom, content []byte) error {
	_, err := rf.ReadFrom(bytes.NewReader(content))
	if err != nil {
		h.Log.Error(err, "failed to serve content")
	}
	return err
}

func getRemoteIP(r any) (net.IP, error) {
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

	udpAddr, ok := remoteAddr.(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("address is not a UDP address: %T", remoteAddr)
	}
	return udpAddr.IP, nil
}
