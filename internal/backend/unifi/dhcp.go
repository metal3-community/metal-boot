package unifi

import (
	"context"
	"net"
	"net/netip"

	"github.com/metal3-community/metal-boot/internal/dhcp/data"
	"go.opentelemetry.io/otel"
)

// GetByMac is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Remote) GetByMac(
	ctx context.Context,
	mac net.HardwareAddr,
) (*data.DHCP, *data.Netboot, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.GetByMac")
	defer span.End()

	dhcp := data.DHCP{}

	netboot := &data.Netboot{
		AllowNetboot: true,
		OSIE:         data.OSIE{},
	}

	dhcp.MACAddress = mac

	if client, err := w.getClient(ctx, mac); err == nil {
		w.clientToDHCP(ctx, client, &dhcp)
	} else {
		w.Log.Error(err, "failed to get active client by mac")
	}

	return &dhcp, netboot, nil
}

// GetByIP is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Remote) GetByIP(
	ctx context.Context,
	ip net.IP,
) (*data.DHCP, *data.Netboot, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.GetByIP")
	defer span.End()

	var ipAddr netip.Addr

	if addr, ok := netip.AddrFromSlice(ip); !ok {
		addr, err := netip.ParseAddr(ip.String())
		if err != nil {
		} else {
			ipAddr = addr
		}
	} else {
		ipAddr = addr
	}

	dhcp := data.DHCP{
		IPAddress: ipAddr,
	}

	netboot := &data.Netboot{
		AllowNetboot: true,
		OSIE:         data.OSIE{},
	}

	if client, err := w.getClientByIP(ctx, ip); err == nil {
		w.clientToDHCP(ctx, client, &dhcp)
	}

	if dhcp.MACAddress.String() == "" {
		return nil, nil, &NotFoundError{}
	}

	return &dhcp, netboot, nil
}
