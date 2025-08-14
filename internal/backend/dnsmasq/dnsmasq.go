// Package dnsmasq provides a DNSMasq-compatible backend for DHCP lease and configuration management.
package dnsmasq

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/bmcpi/pibmc/internal/dhcp/data"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

const tracerName = "github.com/bmcpi/pibmc/backend/dnsmasq"

// Errors used by the dnsmasq backend.
var (
	errRecordNotFound = fmt.Errorf("record not found")
)

// Backend implements the BackendReader and BackendWriter interfaces using DNSMasq-compatible
// lease and configuration files.
type Backend struct {
	mu            sync.RWMutex
	leaseManager  *LeaseManager
	configManager *ConfigManager
	log           logr.Logger

	// Configuration
	rootDir    string
	tftpServer string
	httpServer string
}

// Config holds configuration for the DNSMasq backend.
type Config struct {
	RootDir    string
	TFTPServer string
	HTTPServer string
}

// NewBackend creates a new DNSMasq backend.
func NewBackend(log logr.Logger, config Config) (*Backend, error) {
	leaseManager, err := NewLeaseManager(log, filepath.Join(config.RootDir, "dnsmasq.leases"))
	if err != nil {
		return nil, fmt.Errorf("failed to create lease manager: %w", err)
	}

	configManager := NewConfigManager(config.RootDir)

	backend := &Backend{
		leaseManager:  leaseManager,
		configManager: configManager,
		log:           log,
		rootDir:       config.RootDir,
		tftpServer:    config.TFTPServer,
		httpServer:    config.HTTPServer,
	}

	// Load existing data
	if err := backend.loadData(); err != nil {
		leaseManager.Close() // Clean up on error
		return nil, fmt.Errorf("failed to load existing data: %w", err)
	}

	return backend, nil
}

// loadData loads existing lease and configuration data.
func (b *Backend) loadData() error {
	if err := b.leaseManager.LoadLeases(); err != nil {
		return fmt.Errorf("failed to load leases: %w", err)
	}

	if err := b.configManager.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	return nil
}

// GetByMac implements BackendReader.GetByMac.
func (b *Backend) GetByMac(
	ctx context.Context,
	mac net.HardwareAddr,
) (*data.DHCP, *data.Netboot, *data.Power, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.dnsmasq.GetByMac")
	defer span.End()

	b.mu.RLock()
	defer b.mu.RUnlock()

	lease, exists := b.leaseManager.GetLease(mac)
	if !exists {
		err := fmt.Errorf("%w: %s", errRecordNotFound, mac.String())
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}

	// Convert lease to DHCP data
	dhcpData, err := b.leaseToDHCP(lease)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, nil, err
	}

	// Get netboot options from config
	netbootData := b.getNetbootData(mac)

	// Power data is not supported in DNSMasq format
	powerData := &data.Power{}

	span.SetAttributes(dhcpData.EncodeToAttributes()...)
	span.SetAttributes(netbootData.EncodeToAttributes()...)
	span.SetStatus(codes.Ok, "")

	return dhcpData, netbootData, powerData, nil
}

// GetByIP implements BackendReader.GetByIP.
func (b *Backend) GetByIP(
	ctx context.Context,
	ip net.IP,
) (*data.DHCP, *data.Netboot, *data.Power, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.dnsmasq.GetByIP")
	defer span.End()

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Find lease by IP
	leases := b.leaseManager.GetActiveLeases()
	for _, lease := range leases {
		if lease.IP.Equal(ip) {
			dhcpData, err := b.leaseToDHCP(lease)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				return nil, nil, nil, err
			}

			netbootData := b.getNetbootData(lease.MAC)
			powerData := &data.Power{}

			span.SetAttributes(dhcpData.EncodeToAttributes()...)
			span.SetAttributes(netbootData.EncodeToAttributes()...)
			span.SetStatus(codes.Ok, "")

			return dhcpData, netbootData, powerData, nil
		}
	}

	err := fmt.Errorf("%w: %s", errRecordNotFound, ip.String())
	span.SetStatus(codes.Error, err.Error())
	return nil, nil, nil, err
}

// GetKeys implements BackendReader.GetKeys.
func (b *Backend) GetKeys(ctx context.Context) ([]net.HardwareAddr, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.dnsmasq.GetKeys")
	defer span.End()

	b.mu.RLock()
	defer b.mu.RUnlock()

	leases := b.leaseManager.GetActiveLeases()
	keys := make([]net.HardwareAddr, 0, len(leases))

	for _, lease := range leases {
		keys = append(keys, lease.MAC)
	}

	span.SetStatus(codes.Ok, "")
	return keys, nil
}

// Put implements BackendWriter.Put.
func (b *Backend) Put(
	ctx context.Context,
	mac net.HardwareAddr,
	d *data.DHCP,
	n *data.Netboot,
	p *data.Power,
) error {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.dnsmasq.Put")
	defer span.End()

	b.mu.Lock()
	defer b.mu.Unlock()

	// Add or update lease
	if d != nil {
		hostname := d.Hostname
		if hostname == "" {
			hostname = "*"
		}

		leaseTime := d.LeaseTime
		if leaseTime == 0 {
			leaseTime = 604800 // Default to one week
		}

		b.leaseManager.AddLease(mac, net.IP(d.IPAddress.AsSlice()), hostname, leaseTime)
	}

	// Update netboot configuration
	if n != nil && n.AllowNetboot {
		b.configManager.AddNetbootOptions(mac, b.tftpServer, b.httpServer)
	} else if n != nil && !n.AllowNetboot {
		// Disable netboot for this MAC
		b.configManager.DisableNetboot(mac)
	}

	// Save changes
	if err := b.save(); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// PowerCycle implements BackendPower.PowerCycle.
func (b *Backend) PowerCycle(ctx context.Context, mac net.HardwareAddr) error {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.dnsmasq.PowerCycle")
	defer span.End()

	// Power cycling is not supported in DNSMasq format
	span.SetStatus(codes.Ok, "power cycling not supported")
	return nil
}

// Sync implements BackendSyncer.Sync.
func (b *Backend) Sync(ctx context.Context) error {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.dnsmasq.Sync")
	defer span.End()

	b.mu.Lock()
	defer b.mu.Unlock()

	// Clean expired leases
	b.leaseManager.CleanExpiredLeases()

	// Reload data from files
	if err := b.loadData(); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// save persists current state to files.
func (b *Backend) save() error {
	if err := b.leaseManager.SaveLeases(); err != nil {
		return fmt.Errorf("failed to save leases: %w", err)
	}

	if err := b.configManager.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// leaseToDHCP converts a Lease to data.DHCP.
func (b *Backend) leaseToDHCP(lease *Lease) (*data.DHCP, error) {
	ipAddr, err := netip.ParseAddr(lease.IP.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse IP address: %w", err)
	}

	dhcp := &data.DHCP{
		MACAddress: lease.MAC,
		IPAddress:  ipAddr,
		Hostname:   lease.Hostname,
		LeaseTime:  uint32(lease.Expiry - time.Now().Unix()),
	}

	return dhcp, nil
}

// getNetbootData gets netboot configuration for a MAC address.
func (b *Backend) getNetbootData(mac net.HardwareAddr) *data.Netboot {
	// Check if netboot is enabled for this MAC
	enabled := b.configManager.IsNetbootEnabled(mac)

	netboot := &data.Netboot{
		AllowNetboot: enabled,
	}

	if enabled {
		// Get options for this MAC
		options := b.configManager.GetOptions(mac.String())

		// Try to construct the iPXE script URL from option 67
		for _, option := range options {
			if option.OptionCode == 67 && option.ConditionalTag == "ipxe" {
				if u, err := url.Parse(option.Value); err == nil {
					netboot.IPXEScriptURL = u
				}
				break
			}
		}
	}

	return netboot
}

// Start starts the file watchers for lease and configuration files.
// This is a blocking method. Use a context cancellation to exit.
func (b *Backend) Start(ctx context.Context) {
	// Start the lease file watcher in a goroutine
	go b.leaseManager.Start(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	b.log.Info("stopping dnsmasq backend")
}

// Close closes all file watchers and cleans up resources.
func (b *Backend) Close() error {
	if err := b.leaseManager.Close(); err != nil {
		return fmt.Errorf("failed to close lease manager: %w", err)
	}
	return nil
}
