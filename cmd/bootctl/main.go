// Copyright (c) 2022 individual contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// <https://www.apache.org/licenses/LICENSE-2.0>
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/0x5a17ed/uefi/efi/efiguid"
	"github.com/0x5a17ed/uefi/efi/efivario"
	"github.com/0x5a17ed/uefi/efi/efivars"

	"github.com/spf13/afero"
)

func openSafeguard(fs afero.Fs, fpath string) (p *safeguard, err error) {
	f, err := fs.OpenFile(fpath, os.O_RDONLY, 0644)
	if err != nil {
		switch {
		case errors.Is(err, afero.ErrFileNotFound):
			fallthrough
		case errors.Is(err, syscall.ENOENT):
			return nil, nil
		default:
			return nil, err
		}
	}

	osFile, ok := resolveOsFile(f)
	if !ok {
		// The protection operation is not implemented by the
		// underlying filesystem and thus can't be performed.
		return nil, f.Close()
	}

	p = &safeguard{File: osFile}
	err = withInnerFileDescriptor(osFile, func(fd uintptr) (err error) {
		p.fl, err = getFlags(fd)
		return
	})
	return
}

const (
	DefaultEfiPath = "/Users/atkini01/rpi4"
)

func NewContext(path string) efivario.Context {
	return efivario.NewFileSystemContext(afero.NewBasePathFs(afero.NewOsFs(), path))
}

func main() {

	c := NewContext(DefaultEfiPath)

	_, a, err := efivario.ReadAll(c, "RPI_EFI.fd", efiguid.MustFromString("eb704011-1402-11d3-8e77-00a0c969723b"))
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(a)

	if err := efivars.BootNext.Set(c, 1); err != nil {
		fmt.Println(err)
	}
}
