package lease

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLeaseManager(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dhcp-lease-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	leaseFile := filepath.Join(tmpDir, "dhcp.leases")
	manager := NewManager(leaseFile)

	// Test adding a lease
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	ip := net.ParseIP("192.168.1.100")
	hostname := "test-host"
	leaseTime := uint32(3600) // 1 hour

	manager.AddLease(mac, ip, hostname, leaseTime)

	// Test getting the lease
	lease, exists := manager.GetLease(mac)
	if !exists {
		t.Fatal("Lease should exist")
	}

	if lease.MAC.String() != mac.String() {
		t.Errorf("Expected MAC %s, got %s", mac, lease.MAC)
	}

	if !lease.IP.Equal(ip) {
		t.Errorf("Expected IP %s, got %s", ip, lease.IP)
	}

	if lease.Hostname != hostname {
		t.Errorf("Expected hostname %s, got %s", hostname, lease.Hostname)
	}

	// Test saving leases
	if err := manager.SaveLeases(); err != nil {
		t.Fatalf("Failed to save leases: %v", err)
	}

	// Test loading leases
	newManager := NewManager(leaseFile)
	if err := newManager.LoadLeases(); err != nil {
		t.Fatalf("Failed to load leases: %v", err)
	}

	loadedLease, exists := newManager.GetLease(mac)
	if !exists {
		t.Fatal("Loaded lease should exist")
	}

	if loadedLease.MAC.String() != mac.String() {
		t.Errorf("Expected loaded MAC %s, got %s", mac, loadedLease.MAC)
	}

	if !loadedLease.IP.Equal(ip) {
		t.Errorf("Expected loaded IP %s, got %s", ip, loadedLease.IP)
	}

	if loadedLease.Hostname != hostname {
		t.Errorf("Expected loaded hostname %s, got %s", hostname, loadedLease.Hostname)
	}
}

func TestExpiredLeaseCleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dhcp-lease-cleanup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	leaseFile := filepath.Join(tmpDir, "dhcp.leases")
	manager := NewManager(leaseFile)

	// Add an expired lease (1 second ago)
	mac1, _ := net.ParseMAC("aa:bb:cc:dd:ee:f1")
	ip1 := net.ParseIP("192.168.1.101")
	expiredLease := &Lease{
		Expiry:   time.Now().Add(-time.Second).Unix(),
		MAC:      mac1,
		IP:       ip1,
		Hostname: "expired-host",
	}
	manager.leases[mac1.String()] = expiredLease

	// Add a valid lease
	mac2, _ := net.ParseMAC("aa:bb:cc:dd:ee:f2")
	manager.AddLease(mac2, net.ParseIP("192.168.1.102"), "valid-host", 3600)

	// Clean expired leases
	manager.CleanExpiredLeases()

	// Check that expired lease is removed
	if _, exists := manager.GetLease(mac1); exists {
		t.Error("Expired lease should be removed")
	}

	// Check that valid lease still exists
	if _, exists := manager.GetLease(mac2); !exists {
		t.Error("Valid lease should still exist")
	}

	// Test that expired leases are not saved
	if err := manager.SaveLeases(); err != nil {
		t.Fatalf("Failed to save leases: %v", err)
	}

	// Load and verify only valid lease exists
	newManager := NewManager(leaseFile)
	if err := newManager.LoadLeases(); err != nil {
		t.Fatalf("Failed to load leases: %v", err)
	}

	if _, exists := newManager.GetLease(mac1); exists {
		t.Error("Expired lease should not be in saved file")
	}

	if _, exists := newManager.GetLease(mac2); !exists {
		t.Error("Valid lease should be in saved file")
	}
}

func TestConfigManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dhcp-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "dhcp.conf")
	manager := NewConfigManager(configFile)

	// Test adding options
	tag := "92ef984e-995f-4aea-8088-9cde6a970a88"
	manager.AddOption(tag, "!ipxe", 67, "ipxe.efi")
	manager.AddOption(tag, "ipxe", 67, "http://ironic.appkins.io/boot.ipxe")
	manager.AddOption(tag, "", 66, "10.1.1.1")

	// Test getting options
	options := manager.GetOptions(tag)
	if len(options) != 3 {
		t.Errorf("Expected 3 options, got %d", len(options))
	}

	// Test saving config
	if err := manager.SaveConfig(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Test loading config
	newManager := NewConfigManager(configFile)
	if err := newManager.LoadConfig(); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	loadedOptions := newManager.GetOptions(tag)
	if len(loadedOptions) != 3 {
		t.Errorf("Expected 3 loaded options, got %d", len(loadedOptions))
	}

	// Verify option details
	found := false
	for _, opt := range loadedOptions {
		if opt.ConditionalTag == "!ipxe" && opt.OptionCode == 67 && opt.Value == "ipxe.efi" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected option not found in loaded config")
	}
}

func TestNetbootOptions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dhcp-netboot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "dhcp.conf")
	manager := NewConfigManager(configFile)

	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	tftpServer := "10.1.1.1"
	httpServer := "ironic.appkins.io"

	// Add netboot options
	manager.AddNetbootOptions(mac, tftpServer, httpServer)

	// Get options for this MAC
	options := manager.GetOptions(mac.String())

	// We should have several options
	if len(options) < 4 {
		t.Errorf("Expected at least 4 netboot options, got %d", len(options))
	}

	// Check for specific options
	foundIPXEEfi := false
	foundIPXEScript := false
	foundTFTPServer := false

	for _, opt := range options {
		switch {
		case opt.OptionCode == 67 && opt.ConditionalTag == "!ipxe" && opt.Value == "ipxe.efi":
			foundIPXEEfi = true
		case opt.OptionCode == 67 && opt.ConditionalTag == "ipxe" && opt.Value == "http://ironic.appkins.io/boot.ipxe":
			foundIPXEScript = true
		case opt.OptionCode == 66 && opt.Value == "10.1.1.1":
			foundTFTPServer = true
		}
	}

	if !foundIPXEEfi {
		t.Error("iPXE EFI option not found")
	}
	if !foundIPXEScript {
		t.Error("iPXE script option not found")
	}
	if !foundTFTPServer {
		t.Error("TFTP server option not found")
	}
}
