// Package server provides an http.ServeMux-style TFTP server, ported from
// github.com/tinkerbell/tinkerbell/pkg/tftp/server.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/metal3-community/metal-boot/internal/tftp/middleware"
	"github.com/pin/tftp/v3"
)

const (
	// DefaultTimeout is the default TFTP transfer timeout.
	DefaultTimeout = 5 * time.Second
	// DefaultBlockSize is the default TFTP block size.
	DefaultBlockSize = 512
)

// errWriteNotAllowed is returned for every TFTP write request; the server is read-only.
var errWriteNotAllowed = fmt.Errorf("access_violation: %w", os.ErrPermission)

// Config is the configuration for the TFTP server.
type Config struct {
	// Anticipate is the number of blocks to send ahead of ACKs.
	Anticipate uint
	// BlockSize is the TFTP block size in bytes.
	BlockSize int
	// EnableSinglePort enables single-port mode for TFTP.
	EnableSinglePort bool
	// Timeout is the TFTP transfer timeout.
	Timeout time.Duration
}

func (c *Config) setDefaults() {
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	if c.BlockSize <= 0 {
		c.BlockSize = DefaultBlockSize
	}
}

// Serve starts the TFTP server on the given address with the provided ServeMux
// and blocks until ctx is cancelled or the server fails.
func (c *Config) Serve(ctx context.Context, log logr.Logger, addr string, mux *ServeMux) error {
	if mux == nil {
		return errors.New("tftp: nil ServeMux")
	}
	c.setDefaults()

	server := tftp.NewServer(mux.ServeTFTP, handleWrite(log))
	server.SetTimeout(c.Timeout)
	server.SetBlockSize(c.BlockSize)
	server.SetAnticipate(c.Anticipate)
	server.SetHook(middleware.NewTransferHook(log))

	if c.EnableSinglePort {
		server.EnableSinglePort()
	}

	log.Info("starting tftp server", "addr", addr)
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe(addr)
	}()

	select {
	case <-ctx.Done():
		server.Shutdown()
		// ListenAndServe returns nil once Shutdown completes; wait for it so
		// the socket is released before we return.
		<-serveErr
		return nil
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("tftp server error: %w", err)
		}
		return nil
	}
}

// handleWrite returns a TFTP write handler that always rejects writes.
func handleWrite(log logr.Logger) func(string, io.WriterTo) error {
	return func(filename string, _ io.WriterTo) error {
		log.Error(errWriteNotAllowed, "tftp write request rejected", "filename", filename)
		return errWriteNotAllowed
	}
}
