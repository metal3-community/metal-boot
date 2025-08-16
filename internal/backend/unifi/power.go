package unifi

import (
	"context"
	"fmt"
	"net"
	"slices"

	"github.com/metal3-community/metal-boot/internal/dhcp/data"
	"github.com/metal3-community/metal-boot/internal/util"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"go.opentelemetry.io/otel"
)

// GetByMac is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Remote) GetPower(
	ctx context.Context,
	mac net.HardwareAddr,
) (*data.PowerState, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.GetByMac")
	defer span.End()

	device, err := w.getDevice(ctx, mac)
	if err != nil {
		return nil, err
	}

	pt, err := w.getPortTable(ctx, mac, device)
	if err != nil {
		return nil, err
	}

	currentPower := pt.PoeEnable
	poePower := pt.PoeMode == "auto"

	var power data.PowerState
	if poePower {
		// If POE is enabled, we need to check the power state
		if currentPower {
			power = data.PowerOn
		} else {
			power = data.PoweringOn
		}
	} else {
		if currentPower {
			power = data.PoweringOff
		} else {
			power = data.PowerOff
		}
	}

	return &power, nil
}

func (w *Remote) SetPower(ctx context.Context, mac net.HardwareAddr, state data.PowerState) error {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.SetPower")
	defer span.End()

	device, err := w.getDevice(ctx, mac)
	if err != nil {
		return err
	}

	port, err := w.getPortIdx(mac, device)
	if err != nil {
		return err
	}

	i := slices.IndexFunc(device.PortOverrides, func(i unifi.DevicePortOverrides) bool {
		return i.PortIDX == port
	})
	if i == -1 {
		return fmt.Errorf("no port %d found", port)
	}

	var poeMode string
	if state == data.PowerOn || state == data.PoweringOn {
		poeMode = "auto"
	} else {
		poeMode = "off"
	}

	if device.PortOverrides[i].PoeMode != poeMode {
		device.PortOverrides[i].PoeMode = poeMode

		if dev, err := w.client.UpdateDevice(ctx, w.config.Unifi.Site, device); err != nil {
			return err
		} else {
			i := slices.IndexFunc(dev.PortOverrides, func(i unifi.DevicePortOverrides) bool {
				return i.PortIDX == port
			})
			if i != -1 {
				portOverride := dev.PortOverrides[i]

				w.Log.Info("POE mode changed", "device", dev.ID, "port", port, "mode", portOverride.PoeMode)
			}
		}
	}

	return nil
}

// PowerCycle is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Remote) PowerCycle(ctx context.Context, mac net.HardwareAddr) error {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.PowerCycle")
	defer span.End()

	device, err := w.getDevice(ctx, mac)
	if err != nil {
		return err
	}

	port, err := w.getPortIdx(mac, device)
	if err != nil {
		return err
	}

	if _, err := w.client.ExecuteCmd(ctx, w.config.Unifi.Site, "devmgr", unifi.Cmd{
		Command: "power-cycle",
		Mac:     device.MAC,
		PortIdx: util.Ptr(port),
	}); err != nil {
		w.Log.Error(err, "failed to power cycle")
		return err
	}

	return nil
}
