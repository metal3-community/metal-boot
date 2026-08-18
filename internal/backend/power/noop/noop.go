// Package noop provides a BackendPower implementation that performs no power
// management. It is used when metal-boot runs without an out-of-band power
// controller; power operations report that the feature is unconfigured rather
// than failing at startup.
package noop

import (
	"context"
	"errors"
	"net"

	"github.com/metal3-community/metal-boot/internal/backend"
	"github.com/metal3-community/metal-boot/internal/dhcp/data"
)

// ErrUnsupported is returned by every power operation.
var ErrUnsupported = errors.New("power management is not configured")

// Backend is a no-op power backend.
type Backend struct{}

// New returns a no-op BackendPower.
func New() backend.BackendPower { return &Backend{} }

func (Backend) GetPower(context.Context, net.HardwareAddr) (*data.PowerState, error) {
	return nil, ErrUnsupported
}

func (Backend) SetPower(context.Context, net.HardwareAddr, data.PowerState) error {
	return ErrUnsupported
}

func (Backend) PowerCycle(context.Context, net.HardwareAddr) error {
	return ErrUnsupported
}
