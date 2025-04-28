// Package file watches a file for changes and updates the in memory DHCP data.
package persist

import (
	"context"
	"net"
	"slices"

	"github.com/bmcpi/pibmc/internal/backend/file"
	"github.com/bmcpi/pibmc/internal/backend/remote"
	"github.com/bmcpi/pibmc/internal/config"
	"github.com/bmcpi/pibmc/internal/dhcp/data"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

const tracerName = "github.com/bmcpi/pibmc/backend/persist"

// Persist represents the backend for watching a file for changes and updating the in memory DHCP data.
type Persist struct {
	// Log is the logger to be used in the File backend.
	Log logr.Logger

	config *config.Config

	fileBackend *file.Watcher

	remoteBackend *remote.Remote

	sync bool
}

// NewPersist creates a new file watcher.
func NewPersist(l logr.Logger, config *config.Config) (*Persist, error) {
	fileBackend, err := file.NewWatcher(l, config.BackendFilePath)
	if err != nil {
		return nil, err
	}

	remoteBackend, err := remote.NewRemote(l, config)
	if err != nil {
		return nil, err
	}

	return &Persist{
		Log:           l,
		config:        config,
		fileBackend:   fileBackend,
		remoteBackend: remoteBackend,
	}, nil
}

// GetByMac is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Persist) GetByMac(
	ctx context.Context,
	mac net.HardwareAddr,
) (*data.DHCP, *data.Netboot, *data.Power, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.persist.GetByMac")
	defer span.End()

	if w.sync {
		dd, nn, pp, err := w.remoteBackend.GetByMac(ctx, mac)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, nil, nil, err
		}

		if err := w.Put(ctx, mac, dd, nn, pp); err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, nil, nil, err
		}

		span.SetAttributes(dd.EncodeToAttributes()...)
		span.SetAttributes(nn.EncodeToAttributes()...)
		span.SetStatus(codes.Ok, "")

		w.sync = false
	}

	d, n, p, err := w.remoteBackend.GetByMac(ctx, mac)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}

	// get data from file, translate it, then pass it into set

	return d, n, p, nil
}

// GetByIP is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Persist) GetByIP(
	ctx context.Context,
	ip net.IP,
) (*data.DHCP, *data.Netboot, *data.Power, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.persist.GetByIP")
	defer span.End()

	d, n, p, err := w.remoteBackend.GetByIP(ctx, ip)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}

	if w.sync {
		dd, nn, pp, err := w.remoteBackend.GetByIP(ctx, ip)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, nil, nil, err
		}

		if dd != d || nn != n || pp != p {
			if err := w.Put(ctx, dd.MACAddress, dd, nn, pp); err != nil {
				span.SetStatus(codes.Error, err.Error())
				return nil, nil, nil, err
			}
		}

		span.SetAttributes(dd.EncodeToAttributes()...)
		span.SetAttributes(nn.EncodeToAttributes()...)
		span.SetStatus(codes.Ok, "")

		w.sync = false
	}

	// get data from file, translate it, then pass it into set

	return d, n, p, nil
}

func (w *Persist) Put(
	ctx context.Context,
	mac net.HardwareAddr,
	d *data.DHCP,
	n *data.Netboot,
	p *data.Power,
) error {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.persist.Put")
	defer span.End()

	if err := w.remoteBackend.Put(ctx, mac, d, n, p); err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetAttributes(d.EncodeToAttributes()...)
		span.SetAttributes(n.EncodeToAttributes()...)
		span.SetStatus(codes.Ok, "")
	}

	if err := w.fileBackend.Put(ctx, mac, d, n, p); err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetAttributes(d.EncodeToAttributes()...)
		span.SetAttributes(n.EncodeToAttributes()...)
		span.SetStatus(codes.Ok, "")
	}

	return nil
}

func (p *Persist) PowerCycle(ctx context.Context, mac net.HardwareAddr) error {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.PowerCycle")
	defer span.End()

	return p.remoteBackend.PowerCycle(ctx, mac)
}

func (w *Persist) GetKeys(ctx context.Context) ([]net.HardwareAddr, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.remote.GetKeys")
	defer span.End()

	keys := []net.HardwareAddr{}

	if k, err := w.fileBackend.GetKeys(ctx); err == nil {
		keys = append(keys, k...)
	}

	if k, err := w.remoteBackend.GetKeys(ctx); err == nil {
		for _, key := range k {
			if slices.ContainsFunc(
				keys,
				func(e net.HardwareAddr) bool { return e.String() == key.String() },
			) {
				continue
			}
			keys = append(keys, key)
		}
	}

	span.SetStatus(codes.Ok, "")
	return keys, nil
}

// Start starts watching a file for changes and updates the in memory data (w.data) on changes.
// Start is a blocking method. Use a context cancellation to exit.
func (w *Persist) Start(ctx context.Context) {
	w.sync = true
	go w.fileBackend.Start(ctx)
	go w.remoteBackend.Start(ctx)
}

func (w *Persist) Sync(ctx context.Context) error {
	w.sync = true

	k, err := w.GetKeys(ctx)
	if err != nil {
		return err
	}

	for _, key := range k {
		_, _, _, err := w.GetByMac(ctx, key)
		if err != nil {
			return err
		}
	}

	return nil
}
