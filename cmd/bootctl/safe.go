package main

import (
	"errors"
	"os"
	"syscall"

	"github.com/spf13/afero"
	"go.uber.org/multierr"
	"golang.org/x/sys/unix"
)

const (
	FS_IMMUTABLE_FL = 0x00000010
)

type flags uint32

func (a flags) IsSet(attrs flags) bool  { return a&attrs != 0 }
func (a flags) Clear(attrs flags) flags { return a & ^attrs }
func (a flags) Set(attrs flags) flags   { return a | attrs }

func getFlags(fd uintptr) (flags, error) {
	attrs, err := unix.IoctlGetInt(int(fd), unix.FS_IOC_GETFLAGS)
	return flags(attrs), err
}

func setFlags(fd uintptr, attr flags) error {
	return unix.IoctlSetPointerInt(int(fd), unix.FS_IOC_SETFLAGS, int(attr))
}

func resolveOsFile(f afero.File) (o *os.File, ok bool) {
	// Unwrap afero.BasePathFile instances.
	for {
		if baseFile, ok := f.(*afero.BasePathFile); ok {
			f = baseFile.File
			continue
		}
		break
	}

	o, ok = f.(*os.File)
	return
}

func withInnerFileDescriptor(f *os.File, cb func(fd uintptr) error) (err error) {
	rawConn, err := f.SyscallConn()
	if err != nil {
		return err
	}

	err2 := rawConn.Control(func(fd uintptr) {
		err2 := cb(fd)
		// Report anything else other than unsupported error.
		if !errors.Is(err2, syscall.ENOTTY) {
			err = multierr.Append(err, err2)
		}
	})
	err = multierr.Append(err, err2)
	return
}

type safeguard struct {
	*os.File
	fl flags
}

func (g *safeguard) disable() (wasProtected bool, err error) {
	if g != nil {
		err = withInnerFileDescriptor(g.File, func(fd uintptr) (err error) {
			wasProtected = g.fl.IsSet(FS_IMMUTABLE_FL)
			if !wasProtected {
				return nil
			}
			g.fl = g.fl.Clear(FS_IMMUTABLE_FL)
			return setFlags(fd, g.fl)
		})
	}
	return
}

func (g *safeguard) enable() error {
	return withInnerFileDescriptor(g.File, func(fd uintptr) error {
		g.fl = g.fl.Set(FS_IMMUTABLE_FL)
		return setFlags(fd, g.fl)
	})
}
