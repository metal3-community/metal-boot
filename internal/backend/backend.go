// Package backend holds the interface that backends implement, handlers take in, and the top level dhcp package passes to handlers.
package backend

import (
	"context"
	"net"

	"github.com/metal3-community/metal-boot/internal/dhcp/data"
)

// BackendReader is the interface for getting data from a backend.
//
// Backends implement this interface to provide DHCP and Netboot data to the handlers.
type BackendReader interface {
	// Read data (from a backend) based on a mac address
	// and return DHCP headers and options, including netboot info.
	GetByMac(context.Context, net.HardwareAddr) (*data.DHCP, *data.Netboot, error)
	GetByIP(context.Context, net.IP) (*data.DHCP, *data.Netboot, error)
	GetKeys(context.Context) ([]net.HardwareAddr, error)
}

type BackendWriter interface {
	// Write data (to a backend) based on a mac address
	// and return DHCP headers and options, including netboot info.
	Put(
		ctx context.Context,
		mac net.HardwareAddr,
		d *data.DHCP,
		n *data.Netboot,
	) error
}

type BackendPower interface {
	GetPower(context.Context, net.HardwareAddr) (*data.PowerState, error)
	SetPower(ctx context.Context, mac net.HardwareAddr, state data.PowerState) error
	// Cycle power on a device.
	PowerCycle(ctx context.Context, mac net.HardwareAddr) error
}

type BackendSyncer interface {
	// Sync the backend with the file.
	Sync(ctx context.Context) error
}
