// Package dnsmasq provides DNSMasq-compatible DHCP lease file management.
package dnsmasq

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
)

// Lease represents a DHCP lease entry compatible with DNSMasq format.
type Lease struct {
	// Expiry is the lease expiration time as Unix timestamp
	Expiry int64
	// MAC is the client MAC address
	MAC net.HardwareAddr
	// IP is the assigned IP address
	IP net.IP
	// Hostname is the client hostname
	Hostname string
	// ClientID is the DHCP client identifier (optional)
	ClientID string
	// Declined indicates if this IP was declined by a client
	Declined bool
	// DeclineTime is when the IP was declined (Unix timestamp)
	DeclineTime int64
}

// LeaseManager handles DHCP lease file operations in DNSMasq format with file watching.
type LeaseManager struct {
	fileMu sync.RWMutex // protects LeaseFile for reads

	// LeaseFile is the path to the lease file
	LeaseFile string

	// Log is the logger to be used in the LeaseManager
	Log logr.Logger

	dataMu  sync.RWMutex      // protects leases
	leases  map[string]*Lease // leases maps MAC addresses to lease entries
	watcher *fsnotify.Watcher // file system watcher
}

// NewLeaseManager creates a new lease manager with file watching capabilities.
func NewLeaseManager(log logr.Logger, leaseFile string) (*LeaseManager, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	m := &LeaseManager{
		LeaseFile: leaseFile,
		Log:       log,
		leases:    make(map[string]*Lease),
		watcher:   watcher,
	}

	// Load initial data
	if err := m.LoadLeases(); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to load initial lease data: %w", err)
	}

	// Add the lease file to the watcher if it exists
	if _, err := os.Stat(leaseFile); err == nil {
		if err := watcher.Add(leaseFile); err != nil {
			watcher.Close()
			return nil, fmt.Errorf("failed to add lease file to watcher: %w", err)
		}
	} else if !os.IsNotExist(err) {
		watcher.Close()
		return nil, fmt.Errorf("failed to check lease file: %w", err)
	}

	return m, nil
}

// LoadLeases reads and parses the DNSMasq lease file format.
// DNSMasq lease file format:
// <expiry-time> <mac-address> <ip-address> <hostname> <client-id>
// Example: 1692123456 aa:bb:cc:dd:ee:ff 192.168.1.100 hostname 01:aa:bb:cc:dd:ee:ff.
func (m *LeaseManager) LoadLeases() error {
	m.fileMu.RLock()
	file, err := os.Open(m.LeaseFile)
	m.fileMu.RUnlock()
	if err != nil {
		// If file doesn't exist, that's OK - we'll create it when we write
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open lease file %s: %w", m.LeaseFile, err)
	}
	defer file.Close()

	// Create a temporary map to hold new leases
	newLeases := make(map[string]*Lease)

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		lease, err := m.parseLeaseLine(line)
		if err != nil {
			// Log the error but continue parsing other leases
			m.Log.Error(err, "failed to parse lease line", "line", lineNum, "content", line)
			continue
		}

		newLeases[lease.MAC.String()] = lease
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading lease file: %w", err)
	}

	// Update the leases map with the new data
	m.dataMu.Lock()
	m.leases = newLeases
	m.dataMu.Unlock()

	return nil
}

// parseLeaseLine parses a single lease line from the DNSMasq format.
func (m *LeaseManager) parseLeaseLine(line string) (*Lease, error) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return nil, fmt.Errorf("invalid lease line format: %s", line)
	}

	// Parse expiry time
	expiry, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid expiry time: %s", fields[0])
	}

	// Parse MAC address
	mac, err := net.ParseMAC(fields[1])
	if err != nil {
		return nil, fmt.Errorf("invalid MAC address: %s", fields[1])
	}

	// Parse IP address
	ip := net.ParseIP(fields[2])
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", fields[2])
	}

	// Hostname
	hostname := fields[3]

	// Client ID (optional)
	var clientID string
	if len(fields) > 4 {
		clientID = fields[4]
	}

	return &Lease{
		Expiry:   expiry,
		MAC:      mac,
		IP:       ip,
		Hostname: hostname,
		ClientID: clientID,
	}, nil
}

// AddLease adds or updates a lease entry.
func (m *LeaseManager) AddLease(
	mac net.HardwareAddr,
	ip net.IP,
	hostname string,
	leaseTime uint32,
) {
	expiry := time.Now().Add(time.Duration(leaseTime) * time.Second).Unix()

	lease := &Lease{
		Expiry:   expiry,
		MAC:      mac,
		IP:       ip,
		Hostname: hostname,
	}

	m.dataMu.Lock()
	m.leases[mac.String()] = lease
	m.dataMu.Unlock()
}

// GetLease retrieves a lease by MAC address.
func (m *LeaseManager) GetLease(mac net.HardwareAddr) (*Lease, bool) {
	m.dataMu.RLock()
	lease, exists := m.leases[mac.String()]
	m.dataMu.RUnlock()
	return lease, exists
}

// RemoveLease removes a lease by MAC address.
func (m *LeaseManager) RemoveLease(mac net.HardwareAddr) {
	m.dataMu.Lock()
	delete(m.leases, mac.String())
	m.dataMu.Unlock()
}

// MarkIPDeclined marks an IP address as declined by a client.
// The IP will be excluded from assignment for a cooldown period.
func (m *LeaseManager) MarkIPDeclined(ip string) error {
	m.dataMu.Lock()
	defer m.dataMu.Unlock()

	for _, lease := range m.leases {
		if lease.IP.Equal(net.ParseIP(ip)) {
			lease.Declined = true
			lease.DeclineTime = time.Now().Unix()
		}
	}

	return nil
}

// IsIPDeclined checks if an IP address is currently in decline cooldown.
// Returns true if the IP was recently declined and is still in cooldown.
func (m *LeaseManager) IsIPDeclined(ip string) bool {
	m.dataMu.RLock()
	defer m.dataMu.RUnlock()

	now := time.Now().Unix()
	cooldownPeriod := int64(5 * 60) // 5 minutes cooldown

	for _, lease := range m.leases {
		if lease.IP.Equal(net.ParseIP(ip)) && lease.Declined {
			if lease.DeclineTime > 0 && (now-lease.DeclineTime) < cooldownPeriod {
				return true
			}
		}
	}
	return false
}

// ClearDeclinedIPs removes declined status from IPs that have passed cooldown.
func (m *LeaseManager) ClearDeclinedIPs() error {
	m.dataMu.Lock()
	defer m.dataMu.Unlock()

	now := time.Now().Unix()
	cooldownPeriod := int64(5 * 60) // 5 minutes cooldown

	for mac, lease := range m.leases {
		if lease.Declined && lease.DeclineTime > 0 {
			if (now - lease.DeclineTime) >= cooldownPeriod {
				// Clear declined status but keep the lease
				lease.Declined = false
				lease.DeclineTime = 0
				m.Log.Info("cleared declined status", "mac", mac, "ip", lease.IP.String())
			}
		}
	}
	return nil
}

// SaveLeases writes all leases to the lease file in DNSMasq format.
func (m *LeaseManager) SaveLeases() error {
	// Create directory if it doesn't exist
	if dir := filepath.Dir(m.LeaseFile); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create lease directory: %w", err)
		}
	}

	// Write to temporary file first
	tmpFile := m.LeaseFile + ".tmp"
	file, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary lease file: %w", err)
	}
	defer file.Close()

	// Write header comment
	fmt.Fprintf(file, "# DHCP leases file - DNSMasq compatible format\n")
	fmt.Fprintf(file, "# <expiry-time> <mac-address> <ip-address> <hostname> <client-id>\n")

	// Write all leases
	now := time.Now().Unix()
	m.dataMu.RLock()
	for _, lease := range m.leases {
		// Skip expired leases
		if lease.Expiry < now {
			continue
		}

		line := fmt.Sprintf("%d %s %s %s",
			lease.Expiry,
			lease.MAC.String(),
			lease.IP.String(),
			lease.Hostname,
		)

		if lease.ClientID != "" {
			line += " " + lease.ClientID
		}

		fmt.Fprintln(file, line)
	}
	m.dataMu.RUnlock()

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close temporary lease file: %w", err)
	}

	// Check if the file is being watched before replacing
	m.fileMu.RLock()
	fileExists := false
	if _, err := os.Stat(m.LeaseFile); err == nil {
		fileExists = true
	}
	m.fileMu.RUnlock()

	// Atomically replace the original file
	if err := os.Rename(tmpFile, m.LeaseFile); err != nil {
		return fmt.Errorf("failed to replace lease file: %w", err)
	}

	// Add to watcher if this is the first time the file is created
	if !fileExists {
		if err := m.watcher.Add(m.LeaseFile); err != nil {
			m.Log.Error(err, "failed to add lease file to watcher", "file", m.LeaseFile)
		}
	}

	return nil
}

// CleanExpiredLeases removes expired leases from memory.
func (m *LeaseManager) CleanExpiredLeases() {
	now := time.Now().Unix()
	m.dataMu.Lock()
	for mac, lease := range m.leases {
		if lease.Expiry < now {
			delete(m.leases, mac)
		}
	}
	m.dataMu.Unlock()
}

// GetActiveLeases returns all non-expired leases.
func (m *LeaseManager) GetActiveLeases() map[string]*Lease {
	active := make(map[string]*Lease)
	now := time.Now().Unix()

	m.dataMu.RLock()
	for mac, lease := range m.leases {
		if lease.Expiry >= now {
			active[mac] = lease
		}
	}
	m.dataMu.RUnlock()

	return active
}

// Start starts watching the lease file for changes and updates the in-memory data on changes.
// Start is a blocking method. Use a context cancellation to exit.
func (m *LeaseManager) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			m.Log.Info("stopping lease file watcher")
			return
		case event, ok := <-m.watcher.Events:
			if !ok {
				continue
			}
			if event.Has(fsnotify.Write) {
				m.Log.Info("lease file changed, updating cache", "file", m.LeaseFile)
				if err := m.LoadLeases(); err != nil {
					m.Log.Error(err, "failed to reload lease file", "file", m.LeaseFile)
				}
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				continue
			}
			m.Log.Error(err, "error watching lease file", "file", m.LeaseFile)
		}
	}
}

// Close closes the file watcher and cleans up resources.
func (m *LeaseManager) Close() error {
	if m.watcher != nil {
		return m.watcher.Close()
	}
	return nil
}
