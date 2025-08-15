// Package file watches a file for changes and updates the in memory DHCP data.
package remote

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"slices"
	"time"

	"github.com/bmcpi/pibmc/internal/backend"
	"github.com/bmcpi/pibmc/internal/config"
	"github.com/bmcpi/pibmc/internal/dhcp/data"
	"github.com/bmcpi/pibmc/internal/util"
	"github.com/go-logr/logr"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"go.opentelemetry.io/otel"
)

const tracerName = "github.com/bmcpi/pibmc/backend/remote"

// Remote represents the backend for watching a file for changes and updating the in memory DHCP data.
type Remote struct {
	// Log is the logger to be used in the File backend.
	Log logr.Logger

	config *config.Config

	client *unifi.Client

	jar *cookiejar.Jar

	power map[string]data.PowerState
}

// NewRemote creates a new file watcher.
func NewRemote(
	ctx context.Context,
	l logr.Logger,
	config *config.Config,
) (backend.BackendPower, error) {
	client := unifi.Client{}

	if err := client.SetBaseURL(config.Unifi.Endpoint); err != nil {
		panic(fmt.Sprintf("failed to set base url: %s", err))
	}

	client.SetAPIKey(config.Unifi.APIKey)

	httpClient := &http.Client{}
	httpClient.Transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.Unifi.Insecure,
		},
	}

	jar, _ := cookiejar.New(nil)
	httpClient.Jar = jar

	if err := client.SetHTTPClient(httpClient); err != nil {
		panic(fmt.Sprintf("failed to set http client: %s", err))
	}

	if err := client.Login(ctx, config.Unifi.Username, config.Unifi.Password); err != nil {
		panic(fmt.Sprintf("failed to login: %s", err))
	}

	backend := &Remote{
		Log:    l,
		client: &client,
		config: config,
		jar:    jar,
	}

	return backend, nil
}

func (w *Remote) getClient(ctx context.Context, mac net.HardwareAddr) (*unifi.ClientInfo, error) {
	client, err := w.client.GetClientLocal(
		ctx,
		w.config.Unifi.Site,
		mac.String(),
	)
	if err != nil {
		var notFoundErr *unifi.NotFoundError
		if errors.As(err, &notFoundErr) {
			return nil, &NotFoundError{}
		}
		return nil, err
	}
	return client, nil
}

func (w *Remote) getDevice(ctx context.Context, mac net.HardwareAddr) (*unifi.Device, error) {
	client, err := w.getClient(ctx, mac)
	if err != nil {
		return nil, err
	}
	var deviceMac string
	if client.UplinkMac != "" {
		deviceMac = client.UplinkMac
	} else if client.LastUplinkMac != "" {
		deviceMac = client.LastUplinkMac
	} else {
		return nil, fmt.Errorf("no uplink mac found for client %s", mac.String())
	}

	device, err := w.client.GetDeviceByMAC(ctx, w.config.Unifi.Site, deviceMac)
	if err != nil {
		return nil, err
	}

	return device, nil
}

func (w *Remote) getPortIdx(
	mac net.HardwareAddr,
	device *unifi.Device,
) (int, error) {
	i := slices.IndexFunc(device.PortTable, func(i unifi.DevicePortTable) bool {
		return i.LastConnection.MAC == mac.String()
	})
	if i == -1 {
		return -1, fmt.Errorf("no port found for mac %s", mac.String())
	}
	return device.PortTable[i].PortIdx, nil
}

func (w *Remote) getPortTable(
	ctx context.Context,
	mac net.HardwareAddr,
	device *unifi.Device,
) (*unifi.DevicePortTable, error) {
	i := slices.IndexFunc(device.PortTable, func(i unifi.DevicePortTable) bool {
		return i.LastConnection.MAC == mac.String()
	})
	if i == -1 {
		return nil, fmt.Errorf("no port found for mac %s", mac.String())
	}
	return &device.PortTable[i], nil
}

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

type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "no client found"
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
