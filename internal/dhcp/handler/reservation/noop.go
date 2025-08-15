// Package noop is a backend handler that does nothing.
package reservation

import (
	"context"
	"errors"
	"net"

	"github.com/bmcpi/pibmc/internal/dhcp/data"
)

// Handler is a noop backend.
type noop struct{}

// GetKeys implements backend.BackendReader.
func (h noop) GetKeys(context.Context) ([]net.HardwareAddr, error) {
	return nil, errors.New("no backend specified, please specify a backend")
}

// GetByMac returns an error.
func (h noop) GetByMac(
	_ context.Context,
	_ net.HardwareAddr,
) (*data.DHCP, *data.Netboot, error) {
	return nil, nil, errors.New("no backend specified, please specify a backend")
}

// GetByIP returns an error.
func (h noop) GetByIP(_ context.Context, _ net.IP) (*data.DHCP, *data.Netboot, error) {
	return nil, nil, errors.New("no backend specified, please specify a backend")
}
