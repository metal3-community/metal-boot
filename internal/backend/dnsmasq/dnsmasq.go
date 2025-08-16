// Package dnsmasq provides a DNSMasq-compatible backend for DHCP lease and configuration management.
package dnsmasq

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/metal3-community/metal-boot/internal/backend/dnsmasq/lease"
	"github.com/metal3-community/metal-boot/internal/dhcp/data"
	"github.com/metal3-community/metal-boot/internal/util"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

const tracerName = "github.com/metal3-community/metal-boot/backend/dnsmasq"

// Errors used by the dnsmasq backend.
var (
	errRecordNotFound = fmt.Errorf("record not found")
)

// Backend implements the BackendReader and BackendWriter interfaces using DNSMasq-compatible
// lease and configuration files.
type Backend struct {
	mu            sync.RWMutex
	leaseManager  *lease.LeaseManager
	configManager *ConfigManager
	log           logr.Logger

	// Configuration
	rootDir    string
	tftpServer string
	httpServer string

	// Automatic lease assignment
	autoAssignEnabled bool
	ipPoolStart       net.IP
	ipPoolEnd         net.IP
	defaultLeaseTime  uint32
	defaultGateway    string
	defaultSubnet     string
	defaultDNS        []string
	defaultDomain     string
}

// Config holds configuration for the DNSMasq backend.
type Config struct {
	RootDir    string
	TFTPServer string
	HTTPServer string

	// Automatic lease assignment configuration
	AutoAssignEnabled bool
	IPPoolStart       string
	IPPoolEnd         string
	DefaultLeaseTime  uint32
	DefaultGateway    string
	DefaultSubnet     string
	DefaultDNS        []string
	DefaultDomain     string
}

// NewBackend creates a new DNSMasq backend.
func NewBackend(log logr.Logger, config Config) (*Backend, error) {
	leaseManager, err := lease.NewLeaseManager(log, filepath.Join(config.RootDir, "dnsmasq.leases"))
	if err != nil {
		return nil, fmt.Errorf("failed to create lease manager: %w", err)
	}

	configManager, err := NewConfigManager(log, config.RootDir)
	if err != nil {
		leaseManager.Close() // Clean up on error
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	backend := &Backend{
		leaseManager:  leaseManager,
		configManager: configManager,
		log:           log,
		rootDir:       config.RootDir,
		tftpServer:    config.TFTPServer,
		httpServer:    config.HTTPServer,

		// Auto assignment settings
		autoAssignEnabled: config.AutoAssignEnabled,
		defaultLeaseTime:  config.DefaultLeaseTime,
		defaultGateway:    config.DefaultGateway,
		defaultSubnet:     config.DefaultSubnet,
		defaultDNS:        config.DefaultDNS,
		defaultDomain:     config.DefaultDomain,
	}

	// Parse IP pool range if auto assignment is enabled
	if config.AutoAssignEnabled {
		if config.IPPoolStart != "" {
			ipStart := net.ParseIP(config.IPPoolStart)
			if ipStart == nil {
				leaseManager.Close()
				return nil, fmt.Errorf("invalid IP pool start address: %s", config.IPPoolStart)
			}
			backend.ipPoolStart = ipStart
		}

		if config.IPPoolEnd != "" {
			ipEnd := net.ParseIP(config.IPPoolEnd)
			if ipEnd == nil {
				leaseManager.Close()
				return nil, fmt.Errorf("invalid IP pool end address: %s", config.IPPoolEnd)
			}
			backend.ipPoolEnd = ipEnd
		}

		// Set default lease time if not specified
		if backend.defaultLeaseTime == 0 {
			backend.defaultLeaseTime = 604800 // 1 week default
		}
	}

	// Load existing data
	if err := backend.loadData(); err != nil {
		leaseManager.Close()  // Clean up on error
		configManager.Close() // Clean up on error
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
) (*data.DHCP, *data.Netboot, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.dnsmasq.GetByMac")
	defer span.End()

	b.mu.RLock()
	lease, exists := b.leaseManager.GetLease(mac)
	b.mu.RUnlock()

	if !exists && b.autoAssignEnabled {
		// Automatically assign a lease for unknown MAC addresses
		b.log.Info("MAC address not found, auto-assigning lease", "mac", mac.String())

		assignedIP, err := b.assignIPForMAC(mac)
		if err != nil {
			err = fmt.Errorf("failed to auto-assign IP for MAC %s: %w", mac.String(), err)
			span.SetStatus(codes.Error, err.Error())
			return nil, nil, err
		}

		// Create and store the new lease
		hostname := fmt.Sprintf("auto-%s", mac.String())
		b.mu.Lock()
		b.leaseManager.AddLease(mac, assignedIP, hostname, b.defaultLeaseTime)
		clientID := fmt.Sprintf(mac.String())

		// Set up netboot configuration for auto-assigned devices with architecture-specific boot file
		bootFile := "snp.efi" // Default for arm64
		// bootFile := "ipxe.efi" // Default for x86_64
		// if util.IsRaspberryPI(mac) {
		// 	bootFile = "snp.efi" // ARM64 Raspberry Pi
		// }
		b.configManager.AddNetbootOptionsWithBootFile(mac, b.tftpServer, b.httpServer, bootFile)

		b.mu.Unlock()

		// Save the lease immediately
		if err := b.save(); err != nil {
			b.log.Error(
				err,
				"failed to save auto-assigned lease",
				"mac",
				mac.String(),
				"ip",
				assignedIP.String(),
			)
		}

		// Get the newly created lease
		b.mu.RLock()
		lease, exists = b.leaseManager.GetLease(mac)
		b.mu.RUnlock()
	}

	if !exists {
		err := fmt.Errorf("%w: %s", errRecordNotFound, mac.String())
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, err
	}

	// Convert lease to DHCP data
	dhcpData, err := b.leaseToDHCP(lease)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, err
	}

	// Get netboot options from config
	netbootData := b.getNetbootData(mac)

	span.SetAttributes(dhcpData.EncodeToAttributes()...)
	span.SetAttributes(netbootData.EncodeToAttributes()...)
	span.SetStatus(codes.Ok, "")

	return dhcpData, netbootData, nil
}

// GetByIP implements BackendReader.GetByIP.
func (b *Backend) GetByIP(
	ctx context.Context,
	ip net.IP,
) (*data.DHCP, *data.Netboot, error) {
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
				return nil, nil, err
			}

			netbootData := b.getNetbootData(lease.MAC)

			span.SetAttributes(dhcpData.EncodeToAttributes()...)
			span.SetAttributes(netbootData.EncodeToAttributes()...)
			span.SetStatus(codes.Ok, "")

			return dhcpData, netbootData, nil
		}
	}

	err := fmt.Errorf("%w: %s", errRecordNotFound, ip.String())
	span.SetStatus(codes.Error, err.Error())
	return nil, nil, err
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
		// Use architecture-specific boot file
		bootFile := "ipxe.efi" // Default for x86_64
		if util.IsRaspberryPI(mac) {
			bootFile = "snp.efi" // ARM64 Raspberry Pi
		}
		b.configManager.AddNetbootOptionsWithBootFile(mac, b.tftpServer, b.httpServer, bootFile)
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
func (b *Backend) leaseToDHCP(lease *lease.Lease) (*data.DHCP, error) {
	ipAddr, err := netip.ParseAddr(lease.IP.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse IP address: %w", err)
	}

	dhcp := &data.DHCP{
		MACAddress: lease.MAC,
		IPAddress:  ipAddr,
		Hostname:   lease.Hostname,
		LeaseTime:  uint32(lease.Expiry - time.Now().Unix()),
		ClientID:   lease.ClientID,
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

	// Start the config file watcher in a goroutine
	go b.configManager.Start(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	b.log.Info("stopping dnsmasq backend")
}

// Close closes all file watchers and cleans up resources.
func (b *Backend) Close() error {
	var errs []error

	if err := b.leaseManager.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close lease manager: %w", err))
	}

	if err := b.configManager.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close config manager: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing backend: %v", errs)
	}

	return nil
}

// assignIPForMAC assigns an IP address from the pool for a given MAC address.
// It uses a deterministic hash-based approach to ensure the same MAC gets the same IP.
func (b *Backend) assignIPForMAC(mac net.HardwareAddr) (net.IP, error) {
	if !b.autoAssignEnabled || b.ipPoolStart == nil || b.ipPoolEnd == nil {
		return nil, fmt.Errorf("automatic IP assignment not configured")
	}

	// Convert IP addresses to integers for calculation
	startInt := ipToInt(b.ipPoolStart)
	endInt := ipToInt(b.ipPoolEnd)

	if startInt > endInt {
		return nil, fmt.Errorf(
			"invalid IP pool range: start %s > end %s",
			b.ipPoolStart.String(),
			b.ipPoolEnd.String(),
		)
	}

	poolSize := endInt - startInt + 1

	// Use MD5 hash of MAC address to get a deterministic offset
	macBytes := []byte(mac.String())
	hasher := md5.New()
	hasher.Write(macBytes)
	hash := hasher.Sum(nil)

	// Use first 4 bytes of hash as offset seed
	offsetSeed := binary.BigEndian.Uint32(hash[:4])

	// Calculate offset within the pool
	offset := offsetSeed % poolSize
	assignedInt := startInt + offset

	// Check if this IP is already assigned to a different MAC
	activeLeases := b.leaseManager.GetActiveLeases()

	// Try the calculated IP first, then search sequentially if occupied
	for i := uint32(0); i < poolSize; i++ {
		currentInt := assignedInt + i
		if currentInt > endInt {
			currentInt = startInt + (currentInt - endInt - 1) // wrap around
		}

		testIP := intToIP(currentInt)

		// Check if this IP is available (not assigned to different MAC and not declined)
		occupied := false
		for _, lease := range activeLeases {
			if lease.IP.Equal(testIP) && lease.MAC.String() != mac.String() {
				occupied = true
				break
			}
		}

		// Also check if this IP is currently declined
		if !occupied && b.leaseManager.IsIPDeclined(testIP.String()) {
			occupied = true
		}

		if !occupied {
			return testIP, nil
		}
	}

	// If we get here, the pool is full
	return nil, fmt.Errorf("IP pool exhausted: no available IPs in range %s-%s",
		b.ipPoolStart.String(), b.ipPoolEnd.String())
}

// ipToInt converts an IPv4 address to a uint32.
func ipToInt(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip)
}

// intToIP converts a uint32 to an IPv4 address.
func intToIP(i uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, i)
	return ip
}

// NewConfigFromDnsmasqConfig creates a Backend Config from a DnsmasqConfig.
func NewConfigFromDnsmasqConfig(dnsmasqConfig any) (Config, error) {
	// Use type assertion to access the fields
	// This allows the backend to be independent of the main config package
	config := Config{}

	// Use reflection or type switches to extract values
	// For now, we'll define a simple interface that the caller must satisfy
	if cfg, ok := dnsmasqConfig.(map[string]any); ok {
		if rootDir, exists := cfg["root_directory"]; exists {
			if s, ok := rootDir.(string); ok {
				config.RootDir = s
			}
		}
		if tftpServer, exists := cfg["tftp_server"]; exists {
			if s, ok := tftpServer.(string); ok {
				config.TFTPServer = s
			}
		}
		if httpServer, exists := cfg["http_server"]; exists {
			if s, ok := httpServer.(string); ok {
				config.HTTPServer = s
			}
		}
		if autoAssign, exists := cfg["auto_assign_enabled"]; exists {
			if b, ok := autoAssign.(bool); ok {
				config.AutoAssignEnabled = b
			}
		}
		if poolStart, exists := cfg["ip_pool_start"]; exists {
			if s, ok := poolStart.(string); ok {
				config.IPPoolStart = s
			}
		}
		if poolEnd, exists := cfg["ip_pool_end"]; exists {
			if s, ok := poolEnd.(string); ok {
				config.IPPoolEnd = s
			}
		}
		if leaseTime, exists := cfg["default_lease_time"]; exists {
			if i, ok := leaseTime.(uint32); ok {
				config.DefaultLeaseTime = i
			} else if i, ok := leaseTime.(int); ok {
				config.DefaultLeaseTime = uint32(i)
			}
		}
		if gateway, exists := cfg["default_gateway"]; exists {
			if s, ok := gateway.(string); ok {
				config.DefaultGateway = s
			}
		}
		if subnet, exists := cfg["default_subnet"]; exists {
			if s, ok := subnet.(string); ok {
				config.DefaultSubnet = s
			}
		}
		if dns, exists := cfg["default_dns"]; exists {
			if slice, ok := dns.([]string); ok {
				config.DefaultDNS = slice
			} else if slice, ok := dns.([]any); ok {
				config.DefaultDNS = make([]string, len(slice))
				for i, v := range slice {
					if s, ok := v.(string); ok {
						config.DefaultDNS[i] = s
					}
				}
			}
		}
		if domain, exists := cfg["default_domain"]; exists {
			if s, ok := domain.(string); ok {
				config.DefaultDomain = s
			}
		}
	}

	return config, nil
}
