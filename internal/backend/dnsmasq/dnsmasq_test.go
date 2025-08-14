package dnsmasq

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bmcpi/pibmc/internal/dhcp/data"
	"github.com/go-logr/logr"
)

func TestConfigManagerHostFiles(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dnsmasq-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create hosts and opts directories
	hostsDir := filepath.Join(tmpDir, "hosts")
	optsDir := filepath.Join(tmpDir, "opts")

	if err := os.MkdirAll(hostsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(optsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create test host file with enabled entry
	hostFile1 := filepath.Join(hostsDir, "ironic-9c:6b:00:70:59:8a.conf")
	content1 := "9c:6b:00:70:59:8a,set:92ef984e-995f-4aea-8088-9cde6a970a88,set:ironic\n"
	if err := os.WriteFile(hostFile1, []byte(content1), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create test host file with ignore entry
	hostFile2 := filepath.Join(hostsDir, "ironic-d8:3a:dd:61:4d:15.conf")
	content2 := "d8:3a:dd:61:4d:15,ignore\n"
	if err := os.WriteFile(hostFile2, []byte(content2), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create test options file
	optsFile := filepath.Join(optsDir, "ironic-92ef984e-995f-4aea-8088-9cde6a970a88.conf")
	optsContent := `# Test options
tag:9c:6b:00:70:59:8a,tag:!ipxe,67,ipxe.efi
tag:9c:6b:00:70:59:8a,tag:ipxe,67,http://192.168.1.1/boot.ipxe
tag:9c:6b:00:70:59:8a,66,192.168.1.1
`
	if err := os.WriteFile(optsFile, []byte(optsContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create config manager and load
	cm := NewConfigManager(tmpDir)
	if err := cm.LoadConfig(); err != nil {
		t.Fatal(err)
	}

	// Test that the enabled MAC was loaded correctly
	mac1, _ := net.ParseMAC("9c:6b:00:70:59:8a")
	if !cm.IsNetbootEnabled(mac1) {
		t.Error("Expected MAC 9c:6b:00:70:59:8a to be enabled for netboot")
	}

	// Test that the ignored MAC was loaded correctly
	mac2, _ := net.ParseMAC("d8:3a:dd:61:4d:15")
	if cm.IsNetbootEnabled(mac2) {
		t.Error("Expected MAC d8:3a:dd:61:4d:15 to be disabled for netboot")
	}

	// Test that options were loaded for the enabled MAC
	options := cm.GetOptions(mac1.String())
	if len(options) == 0 {
		t.Error("Expected options to be loaded for enabled MAC")
	}

	// Find option 67 with ipxe conditional
	found := false
	for _, opt := range options {
		if opt.OptionCode == 67 && opt.ConditionalTag == "ipxe" {
			if opt.Value != "http://192.168.1.1/boot.ipxe" {
				t.Errorf("Expected option value http://192.168.1.1/boot.ipxe, got %s", opt.Value)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find option 67 with ipxe conditional")
	}
}

func TestBackendIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dnsmasq-backend-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backend
	config := Config{
		RootDir:    tmpDir,
		TFTPServer: "192.168.1.1",
		HTTPServer: "192.168.1.1",
	}

	logger := logr.Discard()
	backend, err := NewBackend(logger, config)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close() // Clean up resources

	ctx := context.Background()
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")

	// Test adding a lease and netboot config
	ipAddr, _ := netip.ParseAddr("192.168.1.100")
	dhcpData := &data.DHCP{
		MACAddress: mac,
		IPAddress:  ipAddr,
		Hostname:   "test-host",
		LeaseTime:  3600,
	}

	netbootData := &data.Netboot{
		AllowNetboot: true,
	}

	if err := backend.Put(ctx, mac, dhcpData, netbootData, nil); err != nil {
		t.Fatal(err)
	}

	// Test retrieval
	retrievedDHCP, retrievedNetboot, _, err := backend.GetByMac(ctx, mac)
	if err != nil {
		t.Fatal(err)
	}

	if retrievedDHCP.Hostname != "test-host" {
		t.Errorf("Expected hostname test-host, got %s", retrievedDHCP.Hostname)
	}

	if !retrievedNetboot.AllowNetboot {
		t.Error("Expected netboot to be allowed")
	}

	// Test disabling netboot
	netbootData.AllowNetboot = false
	if err := backend.Put(ctx, mac, nil, netbootData, nil); err != nil {
		t.Fatal(err)
	}

	// Test retrieval after disabling
	_, retrievedNetboot, _, err = backend.GetByMac(ctx, mac)
	if err != nil {
		t.Fatal(err)
	}

	if retrievedNetboot.AllowNetboot {
		t.Error("Expected netboot to be disabled")
	}
}

func TestLeaseManagerFileWatching(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dnsmasq-watcher-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	leaseFile := filepath.Join(tmpDir, "dhcp.leases")
	logger := logr.Discard()

	// Create lease manager
	manager, err := NewLeaseManager(logger, leaseFile)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	// Initially should have no leases
	if len(manager.GetActiveLeases()) != 0 {
		t.Error("Expected no leases initially")
	}

	// Write a lease file manually with future timestamps
	futureTime := time.Now().Add(time.Hour).Unix()
	leaseContent := fmt.Sprintf(`%d aa:bb:cc:dd:ee:ff 192.168.1.100 test-host
%d aa:bb:cc:dd:ee:11 192.168.1.101 test-host2
`, futureTime, futureTime+1)
	if err := os.WriteFile(leaseFile, []byte(leaseContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Give some time for the file watcher to pick up the change
	// In a real scenario, the Start() method would be running in a goroutine
	if err := manager.LoadLeases(); err != nil {
		t.Fatal(err)
	}

	// Should now have leases
	leases := manager.GetActiveLeases()
	if len(leases) != 2 {
		t.Errorf("Expected 2 leases, got %d", len(leases))
	}

	// Check specific lease
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	lease, exists := manager.GetLease(mac)
	if !exists {
		t.Error("Expected to find lease for aa:bb:cc:dd:ee:ff")
	}
	if lease.Hostname != "test-host" {
		t.Errorf("Expected hostname test-host, got %s", lease.Hostname)
	}
}
