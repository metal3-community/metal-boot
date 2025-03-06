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

	"github.com/bmcpi/pibmc/internal/dhcp/data"
	"github.com/bmcpi/pibmc/internal/dhcp/handler"
	"github.com/bmcpi/pibmc/internal/firmware/edk2"
	"github.com/go-logr/logr"

	"github.com/pin/tftp/v3"
	"github.com/tinkerbell/ipxedust/binary"
)

type Server struct {
	Logger        logr.Logger
	RootDirectory string
	Patch         string
	Log           logr.Logger
}

type Handler struct {
	ctx           context.Context
	RootDirectory string
	Patch         string
	Log           logr.Logger

	backend handler.BackendReader
}

func (h Handler) OnSuccess(stats tftp.TransferStats) {
	h.Log.Info("transfer complete", "stats", stats)
}

func (h Handler) OnFailure(stats tftp.TransferStats, err error) {
	h.Log.Error(err, "transfer failed", "stats", stats)
}

// ListenAndServe sets up the listener on the given address and serves TFTP requests.
func (r *Server) ListenAndServe(ctx context.Context, addr netip.AddrPort, backend handler.BackendReader) error {
	tftpHandler := &Handler{
		ctx:           ctx,
		RootDirectory: r.RootDirectory,
		Patch:         r.Patch,
		Log:           r.Logger,
		backend:       backend,
	}

	s := tftp.NewServer(tftpHandler.HandleRead, tftpHandler.HandleWrite)

	s.SetHook(tftpHandler)

	a, err := net.ResolveUDPAddr("udp", addr.String())
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", a)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		r.Logger.Info("shutting down tftp server")
		s.Shutdown()
	}()
	if err := Serve(ctx, conn, s); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		r.Logger.Error(err, "listen and serve http")
		return err
	}

	return nil
}

// Serve serves TFTP requests using the given conn and server.
func Serve(_ context.Context, conn net.PacketConn, s *tftp.Server) error {
	return s.Serve(conn)
}

func (h *Handler) getDhcpInfo(ctx context.Context, f any) (*data.DHCP, *data.Netboot, *data.Power, error) {
	outgoingTransfer, ok := f.(tftp.OutgoingTransfer)
	if !ok {
		err := fmt.Errorf("invalid type: %w", os.ErrInvalid)
		h.Log.Error(err, "invalid type", "type", fmt.Sprintf("%T", f))
	}

	remoteAddr := outgoingTransfer.RemoteAddr()
	h.Log.Info("client", "remoteAddr", remoteAddr, "event", "getdhcpinfo")

	dhcpInfo, netboot, _, err := h.backend.GetByIP(h.ctx, remoteAddr.IP)
	if err != nil {
		h.Log.Error(err, "failed to get dhcp info", "remoteAddr", remoteAddr)
	}

	if dhcpInfo == nil {
		err := fmt.Errorf("failed to get dhcp info: %w", os.ErrNotExist)
		h.Log.Error(err, "failed to get dhcp info", "remoteAddr", remoteAddr)
	}

	if netboot == nil {
		err := fmt.Errorf("failed to get netboot info: %w", os.ErrNotExist)
		h.Log.Error(err, "failed to get netboot info", "remoteAddr", remoteAddr)
	}

	return dhcpInfo, netboot, nil, nil
}

// HandleRead handlers TFTP GET requests. The function signature satisfies the tftp.Server.readHandler parameter type.
func (h *Handler) HandleRead(fullfilepath string, rf io.ReaderFrom) error {
	dhcpInfo, netboot, _, err := h.getDhcpInfo(h.ctx, rf)
	if err != nil {
		return err
	}

	patch := h.Patch
	if netboot != nil {
		if len(netboot.IPXEScript) <= len(magicString) {
			patch = netboot.IPXEScript
		}
	}

	content, ok := binary.Files[filepath.Base(fullfilepath)]
	if ok {
		return h.HandleIpxeRead(fullfilepath, rf, content, patch)
	}

	root, err := OpenRoot(h.RootDirectory)
	if err != nil {
		h.Log.Error(err, "opening root directory failed", "rootDirectory", h.RootDirectory)
		return fmt.Errorf("opening root directory %s: %w", h.RootDirectory, err)
	}
	defer root.Close()

	parts := strings.Split(fullfilepath, "/")
	filename := parts[len(parts)-1]
	filedir := strings.Join(parts[:len(parts)-1], "/")
	prefix := parts[0]

	hasSerial := regexp.MustCompile(`^\d{2}[a-z]\d{5}$`).MatchString(prefix)
	if hasSerial {
		if dhcpInfo == nil {
			err := fmt.Errorf("serial detected, but no dhcp info: %w", os.ErrNotExist)
			h.Log.Error(err, "serial detected, but no dhcp info", "serial", prefix)
		} else {
			newMac := dhcpInfo.MACAddress.String()

			fullfilepath = strings.ReplaceAll(fullfilepath, prefix, newMac)

			parts = strings.Split(fullfilepath, "/")
			filename = parts[len(parts)-1]
			filedir = strings.Join(parts[:len(parts)-1], "/")
			prefix = parts[0]

		}
	}

	hasMac := false
	if _, err := net.ParseMAC(prefix); err == nil {
		hasMac = true
	}

	if hasMac {
		rootpath := filename
		if len(parts) > 2 {
			rootpath = strings.Join(parts[1:], "/")
		}

		childExists := false
		if !root.Exists(filedir) {
			h.Log.Info("creating directories for %s", rootpath)
			// If the mac address directory does not exist, create it.
			err := root.MkdirAll(filedir, 0755)
			if err != nil {
				h.Log.Error(err, "creating directory failed", "directory", filedir)
				return fmt.Errorf("creating %s: %w", filedir, err)
			}
		} else {
			childExists = root.Exists(fullfilepath)
		}

		if !childExists {
			rootExists := root.Exists(rootpath)

			if rootExists {
				// If the file exists in the new path, but not in the old path, use the new path.
				// This is to support the old path for backwards compatibility.
				newF, err := root.Create(fullfilepath)
				if err != nil {
					h.Log.Error(err, "creating file failed", "filename", filename)
					return fmt.Errorf("creating %s: %w", filename, err)
				}
				defer newF.Close()
				oldF, err := root.Open(rootpath)
				if err != nil {
					h.Log.Error(err, "opening file failed", "filename", rootpath)
					return fmt.Errorf("opening %s: %w", rootpath, err)
				}
				defer oldF.Close()
				_, err = io.Copy(newF, oldF)
				if err != nil {
					h.Log.Error(err, "copying file failed", "filename", rootpath)
					return fmt.Errorf("copying %s to %s: %w", rootpath, filename, err)
				}
			} else if content, ok := edk2.Files[rootpath]; ok {
				if err := h.createFile(root, fullfilepath, content); err != nil {
					return err
				}
			}
		}
	}

	isPxe := false
	if strings.Contains(prefix, "pxelinux.cfg") {
		isPxe = true
	}

	// TODO: Add support for PXE booting - make it a toggle feature
	if isPxe && false {

		pxeConfig := `
		KERNEL undionly.kpxe dhcp
		`

		ct := bytes.NewReader([]byte(pxeConfig))
		b, err := rf.ReadFrom(ct)
		if err != nil {
			h.Log.Error(err, "file serve failed", "fullfilepath", fullfilepath, "b", b, "contentSize", len(content))
			return err
		} else {
			h.Log.Info("file served", "bytesSent", b, "contentSize", len(content))
			return nil
		}
	}

	var parsedfilepath string
	if hasSerial {
		parsedfilepath = strings.Join(parts[:], "/")
	} else {
		parsedfilepath = strings.Join(parts, "/")
	}

	if _, err := root.Stat(fullfilepath); err == nil {
		// file exists
		file, err := root.Open(fullfilepath)
		if err != nil {
			errMsg := fmt.Sprintf("opening %s: %s", fullfilepath, err.Error())
			h.Log.Error(err, "file open failed")
			return errors.New(errMsg)
		}
		n, err := rf.ReadFrom(file)
		if err != nil {
			errMsg := fmt.Sprintf("reading %s: %s", fullfilepath, err.Error())
			h.Log.Error(err, "file read failed")
			return errors.New(errMsg)
		}
		h.Log.Info("bytes sent", n)
		return nil

	} else if content, ok := edk2.Files[parsedfilepath]; ok {
		ct := bytes.NewReader(content)
		b, err := rf.ReadFrom(ct)
		if err != nil {
			h.Log.Error(err, "file serve failed", "fullfilepath", fullfilepath, "b", b, "contentSize", len(content))
			return err
		}
		h.Log.Info("file served", "bytesSent", b, "contentSize", len(content))
	} else {
		errMsg := fmt.Sprintf("error checking if file exists: %s", fullfilepath)
		h.Log.Error(err, errMsg)
		return errors.New(errMsg)
	}

	// content, ok := binary.Files[filepath.Base(shortfile)]
	// if !ok {
	// 	err := fmt.Errorf("file [%v] unknown: %w", filepath.Base(shortfile), os.ErrNotExist)
	// 	h.Log.Error(err, "file unknown")
	// 	span.SetStatus(codes.Error, err.Error())
	// 	return err
	// }

	// content, err = binary.Patch(content, t.Patch)
	// if err != nil {
	// 	h.Log.Error(err, "failed to patch binary")
	// 	span.SetStatus(codes.Error, err.Error())
	// 	return err
	// }

	// ct := bytes.NewReader(content)
	// b, err := rf.ReadFrom(ct)
	// if err != nil {
	// 	h.Log.Error(err, "file serve failed", "b", b, "contentSize", len(content))
	// 	span.SetStatus(codes.Error, err.Error())

	// 	return err
	// }
	// h.Log.Info("file served", "bytesSent", b, "contentSize", len(content))
	// span.SetStatus(codes.Ok, filename)

	return nil
}

func (h *Handler) createFile(root *Root, filename string, content []byte) error {
	// If the file does not exist in the new path, but exists in the edk2.Files map, use the map.
	newF, err := root.Create(filename)
	if err != nil {
		h.Log.Error(err, "creating file failed", "filename", filename)
		return fmt.Errorf("creating %s: %w", filename, err)
	}
	defer newF.Close()
	_, err = newF.Write(content)
	if err != nil {
		h.Log.Error(err, "writing file failed", "filename", filename)
		return fmt.Errorf("writing %s: %w", filename, err)
	}

	return nil
}

var magicString = []byte(`#a8b7e61f1075c37a793f2f92cee89f7bba00c4a8d7842ce3d40b5889032d8881
#ddd16a4fc4926ecefdfb6941e33c44ed3647133638f5e84021ea44d3152e7f97`)

func (h *Handler) HandleIpxeRead(filename string, rf io.ReaderFrom, content []byte, patch string) error {
	if patch == "" {
		patch = h.Patch
	}
	// if true {
	// 	patch += fmt.Sprintf("\n  %s\n  %s", "echo -n 'ipxe booting...'", "sanboot --no-describe --drive 0x80")
	// }
	content, err := binary.Patch(content, []byte(patch))
	if err != nil {
		h.Log.Error(err, "failed to patch binary")
		return err
	}

	ct := bytes.NewReader(content)
	b, err := rf.ReadFrom(ct)
	if err != nil {
		h.Log.Error(err, "file serve failed", "b", b, "contentSize", len(content))
		return err
	}
	h.Log.Info("file served", "bytesSent", b, "contentSize", len(content))

	return nil
}

// HandleWrite handles TFTP PUT requests. It will always return an error. This library does not support PUT.
func (h *Handler) HandleWrite(fullfilepath string, wt io.WriterTo) error {

	dhcpInfo, _, _, err := h.getDhcpInfo(h.ctx, wt)
	if err != nil {
		return err
	}

	parts := strings.Split(fullfilepath, "/")
	prefix := parts[0]

	hasSerial := regexp.MustCompile(`^\d{2}[a-z]\d{5}$`).MatchString(prefix)
	if hasSerial {
		if dhcpInfo == nil {
			err := fmt.Errorf("serial detected, but no dhcp info: %w", os.ErrNotExist)
			h.Log.Error(err, "serial detected, but no dhcp info", "serial", prefix)
		} else {
			newMac := dhcpInfo.MACAddress.String()

			fullfilepath = strings.ReplaceAll(fullfilepath, prefix, newMac)

			parts = strings.Split(fullfilepath, "/")
			fullfilepath = parts[len(parts)-1]
		}
	}

	root, err := os.OpenRoot(h.RootDirectory)
	if err != nil {
		h.Log.Error(err, "opening root directory failed", "rootDirectory", h.RootDirectory)
		return fmt.Errorf("opening root directory %s: %w", h.RootDirectory, err)
	}
	defer root.Close()

	file, err := root.OpenFile(fullfilepath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		h.Log.Error(err, "opening file failed", "filename", fullfilepath)
		return nil
	}
	n, err := wt.WriteTo(file)
	if err != nil {
		h.Log.Error(err, "writing file failed", "filename", fullfilepath)
		return nil
	}
	h.Log.Info("bytes received", n)
	return nil
}
