// Package lease provides DNSMasq-compatible DHCP lease file management.
package lease

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
}

// Manager handles DHCP lease file operations in DNSMasq format.
type Manager struct {
	// LeaseFile is the path to the lease file
	LeaseFile string
	// leases maps MAC addresses to lease entries
	leases map[string]*Lease
}

// NewManager creates a new lease manager with the specified lease file path.
func NewManager(leaseFile string) *Manager {
	return &Manager{
		LeaseFile: leaseFile,
		leases:    make(map[string]*Lease),
	}
}

// LoadLeases reads and parses the DNSMasq lease file format.
// DNSMasq lease file format:
// <expiry-time> <mac-address> <ip-address> <hostname> <client-id>
// Example: 1692123456 aa:bb:cc:dd:ee:ff 192.168.1.100 hostname 01:aa:bb:cc:dd:ee:ff.
func (m *Manager) LoadLeases() error {
	file, err := os.Open(m.LeaseFile)
	if err != nil {
		// If file doesn't exist, that's OK - we'll create it when we write
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open lease file %s: %w", m.LeaseFile, err)
	}
	defer file.Close()

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
			fmt.Fprintf(os.Stderr, "Warning: failed to parse lease line %d: %v\n", lineNum, err)
			continue
		}

		m.leases[lease.MAC.String()] = lease
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading lease file: %w", err)
	}

	return nil
}

// parseLeaseLine parses a single lease line from the DNSMasq format.
func (m *Manager) parseLeaseLine(line string) (*Lease, error) {
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
func (m *Manager) AddLease(mac net.HardwareAddr, ip net.IP, hostname string, leaseTime uint32) {
	expiry := time.Now().Add(time.Duration(leaseTime) * time.Second).Unix()

	lease := &Lease{
		Expiry:   expiry,
		MAC:      mac,
		IP:       ip,
		Hostname: hostname,
	}

	m.leases[mac.String()] = lease
}

// GetLease retrieves a lease by MAC address.
func (m *Manager) GetLease(mac net.HardwareAddr) (*Lease, bool) {
	lease, exists := m.leases[mac.String()]
	return lease, exists
}

// RemoveLease removes a lease by MAC address.
func (m *Manager) RemoveLease(mac net.HardwareAddr) {
	delete(m.leases, mac.String())
}

// SaveLeases writes all leases to the lease file in DNSMasq format.
func (m *Manager) SaveLeases() error {
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

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close temporary lease file: %w", err)
	}

	// Atomically replace the original file
	if err := os.Rename(tmpFile, m.LeaseFile); err != nil {
		return fmt.Errorf("failed to replace lease file: %w", err)
	}

	return nil
}

// CleanExpiredLeases removes expired leases from memory.
func (m *Manager) CleanExpiredLeases() {
	now := time.Now().Unix()
	for mac, lease := range m.leases {
		if lease.Expiry < now {
			delete(m.leases, mac)
		}
	}
}

// GetActiveLeases returns all non-expired leases.
func (m *Manager) GetActiveLeases() map[string]*Lease {
	active := make(map[string]*Lease)
	now := time.Now().Unix()

	for mac, lease := range m.leases {
		if lease.Expiry >= now {
			active[mac] = lease
		}
	}

	return active
}
