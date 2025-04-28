// Package types contains common firmware related types and structures.
package types

import (
	"path"
)

// NetworkSettings contains network-related UEFI settings.
type NetworkSettings struct {
	MacAddress  string
	IPAddress   string
	SubnetMask  string
	Gateway     string
	DNSServers  []string
	EnableIPv6  bool
	EnableDHCP  bool
	VLANEnabled bool
	VLANID      string
}

// BootEntry represents a single UEFI boot entry.
type BootEntry struct {
	ID       string
	Name     string
	DevPath  string
	Enabled  bool
	OptData  string
	Position int
}

// FirmwareSource defines a source for firmware files.
type FirmwareSource struct {
	Path string
	URL  string
}

// IsArchive checks if the URL points to an archive file.
func (f *FirmwareSource) IsArchive() bool {
	return IsArchiveExt(path.Ext(f.URL))
}

// IsArchiveExt checks if a file extension is for an archive format.
func IsArchiveExt(ext string) bool {
	return ext == ".zip" || ext == ".tar" || ext == ".tgz" || ext == ".tar.gz"
}

// SystemInfo contains firmware and system information.
type SystemInfo map[string]string
