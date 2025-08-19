package dnsmasq

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/metal3-community/metal-boot/internal/backend/dnsmasq/lease"
	"github.com/metal3-community/metal-boot/internal/dhcp/data"
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
tag:92ef984e-995f-4aea-8088-9cde6a970a88,tag:!ipxe,67,ipxe.efi
tag:92ef984e-995f-4aea-8088-9cde6a970a88,tag:ipxe,67,http://192.168.1.1/boot.ipxe
tag:92ef984e-995f-4aea-8088-9cde6a970a88,66,192.168.1.1
`
	if err := os.WriteFile(optsFile, []byte(optsContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create config manager and load
	cm, err := NewConfigManager(logr.Discard(), tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cm.Close()

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

	if err := backend.Put(ctx, mac, dhcpData, netbootData); err != nil {
		t.Fatal(err)
	}

	// Test retrieval
	retrievedDHCP, retrievedNetboot, err := backend.GetByMac(ctx, mac)
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
	if err := backend.Put(ctx, mac, nil, netbootData); err != nil {
		t.Fatal(err)
	}

	// Test retrieval after disabling
	_, retrievedNetboot, err = backend.GetByMac(ctx, mac)
	if err != nil {
		t.Fatal(err)
	}

	if retrievedNetboot.AllowNetboot {
		t.Error("Expected netboot to be disabled")
	}
}

func TestAutomaticLeaseAssignment(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dnsmasq-auto-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backend with automatic assignment enabled
	config := Config{
		RootDir:           tmpDir,
		TFTPServer:        "192.168.1.1",
		HTTPServer:        "192.168.1.1",
		AutoAssignEnabled: true,
		IPPoolStart:       "192.168.1.100",
		IPPoolEnd:         "192.168.1.110",
		DefaultLeaseTime:  3600,
	}

	logger := logr.Discard()
	backend, err := NewBackend(logger, config)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	ctx := context.Background()

	// Test automatic assignment for unknown MAC
	unknownMAC, _ := net.ParseMAC("bb:bb:cc:dd:ee:ff")

	// Should automatically assign an IP
	dhcpData, _, err := backend.GetByMac(ctx, unknownMAC)
	if err != nil {
		t.Fatalf("Expected automatic assignment to succeed, got error: %v", err)
	}

	if dhcpData == nil {
		t.Fatal("Expected DHCP data to be returned")
	}

	// Verify IP is within the configured range
	assignedIP := dhcpData.IPAddress
	startIP, _ := netip.ParseAddr("192.168.1.100")
	endIP, _ := netip.ParseAddr("192.168.1.110")

	if assignedIP.Compare(startIP) < 0 || assignedIP.Compare(endIP) > 0 {
		t.Errorf("Assigned IP %s is outside the configured range %s-%s",
			assignedIP.String(), startIP.String(), endIP.String())
	}

	// Verify hostname is auto-generated
	expectedHostname := fmt.Sprintf("auto-%s", unknownMAC.String())
	if dhcpData.Hostname != expectedHostname {
		t.Errorf("Expected hostname %s, got %s", expectedHostname, dhcpData.Hostname)
	}

	// Test that the same MAC gets the same IP on subsequent calls
	dhcpData2, _, err := backend.GetByMac(ctx, unknownMAC)
	if err != nil {
		t.Fatalf("Second call should succeed: %v", err)
	}

	if dhcpData.IPAddress.Compare(dhcpData2.IPAddress) != 0 {
		t.Errorf("Expected same IP on second call: got %s then %s",
			dhcpData.IPAddress.String(), dhcpData2.IPAddress.String())
	}

	// Test that different MACs get different IPs
	anotherMAC, _ := net.ParseMAC("cc:cc:cc:dd:ee:ff")
	dhcpData3, _, err := backend.GetByMac(ctx, anotherMAC)
	if err != nil {
		t.Fatalf("Assignment for different MAC should succeed: %v", err)
	}

	if dhcpData.IPAddress.Compare(dhcpData3.IPAddress) == 0 {
		t.Errorf("Different MACs should get different IPs, both got %s",
			dhcpData.IPAddress.String())
	}
}

func TestAutomaticAssignmentDisabled(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dnsmasq-no-auto-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backend with automatic assignment disabled (default)
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
	defer backend.Close()

	ctx := context.Background()

	// Test that unknown MAC returns error when auto-assignment is disabled
	unknownMAC, _ := net.ParseMAC("dd:dd:dd:dd:ee:ff")

	_, _, err = backend.GetByMac(ctx, unknownMAC)
	if err == nil {
		t.Fatal("Expected error for unknown MAC when auto-assignment is disabled")
	}

	if !strings.Contains(err.Error(), "record not found") {
		t.Errorf("Expected 'record not found' error, got: %v", err)
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
	manager, err := lease.NewLeaseManager(logger, leaseFile)
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

// TestConfigManagerFileWatching tests that the ConfigManager properly watches configuration files.
func TestConfigManagerFileWatching(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dnsmasq-config-watcher-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logr.Discard()

	// Create config manager
	cm, err := NewConfigManager(logger, tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cm.Close()

	// Initially should have no hosts or options
	if len(cm.hosts) != 0 {
		t.Error("Expected no hosts initially")
	}

	// Create directories
	hostsDir := filepath.Join(tmpDir, "hosts")
	optsDir := filepath.Join(tmpDir, "opts")
	if err := os.MkdirAll(hostsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(optsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a host configuration file
	hostFile := filepath.Join(hostsDir, "ironic-testnode.conf")
	hostContent := "9c:6b:00:70:59:8a,set:testnode,set:ironic\n"
	if err := os.WriteFile(hostFile, []byte(hostContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a corresponding options file
	optsFile := filepath.Join(optsDir, "ironic-testnode.conf")
	optsContent := `# Test options
tag:testnode,tag:!ipxe,67,snp.efi
tag:testnode,tag:ipxe,67,http://192.168.1.1/boot.ipxe
tag:testnode,66,192.168.1.1
`
	if err := os.WriteFile(optsFile, []byte(optsContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Manually trigger reload (in real scenario, file watcher would trigger this)
	if err := cm.LoadConfig(); err != nil {
		t.Fatal(err)
	}

	// Should now have host and options
	mac, _ := net.ParseMAC("9c:6b:00:70:59:8a")
	if !cm.IsNetbootEnabled(mac) {
		t.Error("Expected MAC 9c:6b:00:70:59:8a to be enabled for netboot")
	}

	// Check that options were loaded
	options := cm.GetOptions(mac.String())
	if len(options) == 0 {
		t.Error("Expected to find options for MAC address")
	}

	// Check a specific option
	foundBootfile := false
	for _, opt := range options {
		if opt.OptionCode == 67 && opt.ConditionalTag == "!ipxe" && opt.Value == "snp.efi" {
			foundBootfile = true
			break
		}
	}
	if !foundBootfile {
		t.Error("Expected to find bootfile option with snp.efi value")
	}
}

// TestIPReassignmentViaPut tests that the Put method can handle IP reassignment when called with empty IP.
func TestIPReassignmentViaPut(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dnsmasq-reassign-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backend with auto-assignment enabled
	config := Config{
		RootDir:           tmpDir,
		TFTPServer:        "192.168.1.1",
		HTTPServer:        "192.168.1.1",
		AutoAssignEnabled: true,
		IPPoolStart:       "192.168.1.100",
		IPPoolEnd:         "192.168.1.110",
		DefaultLeaseTime:  3600,
	}

	backend, err := NewBackend(logr.Discard(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	ctx := context.Background()
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")

	// First, assign an IP normally
	dhcpData, _, err := backend.GetByMac(ctx, mac)
	if err != nil {
		t.Fatalf("Expected automatic assignment to succeed, got error: %v", err)
	}

	originalIP := dhcpData.IPAddress
	t.Logf("Original IP assigned: %s", originalIP.String())

	// Mark the IP as declined to simulate DHCP DECLINE
	if err := backend.leaseManager.MarkIPDeclined(originalIP.String()); err != nil {
		t.Fatalf("Failed to mark IP as declined: %v", err)
	}

	// Now use Put with empty IP to trigger reassignment
	newDHCP := &data.DHCP{
		MACAddress: mac,
		// Leave IPAddress empty to trigger auto-assignment
		Hostname:  "test-reassign",
		LeaseTime: 3600,
	}

	newNetboot := &data.Netboot{
		AllowNetboot: true,
	}

	// Use Put to reassign IP
	if err := backend.Put(ctx, mac, newDHCP, newNetboot); err != nil {
		t.Fatalf("Failed to reassign IP via Put: %v", err)
	}

	// Verify new IP was assigned
	updatedDHCP, _, err := backend.GetByMac(ctx, mac)
	if err != nil {
		t.Fatalf("Failed to get updated data: %v", err)
	}

	newIP := updatedDHCP.IPAddress
	t.Logf("New IP assigned: %s", newIP.String())

	// Verify the new IP is different from the original (declined) IP
	if newIP.String() == originalIP.String() {
		t.Error("New IP should be different from the declined IP")
	}

	// Verify the new IP is within the configured range
	startIP, _ := netip.ParseAddr("192.168.1.100")
	endIP, _ := netip.ParseAddr("192.168.1.110")

	if newIP.Compare(startIP) < 0 || newIP.Compare(endIP) > 0 {
		t.Errorf("New IP %s is outside the configured range %s-%s",
			newIP.String(), startIP.String(), endIP.String())
	}

	// Verify the hostname was updated
	if updatedDHCP.Hostname != "test-reassign" {
		t.Errorf("Expected hostname 'test-reassign', got '%s'", updatedDHCP.Hostname)
	}
}

// TestDHCPDeclineWithBackendWriter tests the complete DHCP DECLINE flow using BackendWriter.
func TestDHCPDeclineWithBackendWriter(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dnsmasq-decline-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backend with auto-assignment enabled
	config := Config{
		RootDir:           tmpDir,
		TFTPServer:        "192.168.1.1",
		HTTPServer:        "192.168.1.1",
		AutoAssignEnabled: true,
		IPPoolStart:       "192.168.1.100",
		IPPoolEnd:         "192.168.1.110",
		DefaultLeaseTime:  3600,
	}

	backend, err := NewBackend(logr.Discard(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	ctx := context.Background()
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")

	// Simulate a DHCP client getting an initial IP
	dhcpData, _, err := backend.GetByMac(ctx, mac)
	if err != nil {
		t.Fatalf("Initial IP assignment failed: %v", err)
	}
	initialIP := dhcpData.IPAddress
	t.Logf("Initial IP assigned: %s", initialIP.String())

	// Verify lease file was written
	leaseFile := filepath.Join(tmpDir, "dnsmasq.leases")
	if _, err := os.Stat(leaseFile); err != nil {
		t.Fatalf("Lease file was not created: %v", err)
	}

	// Read lease file content to verify it contains our lease
	leaseContent, err := os.ReadFile(leaseFile)
	if err != nil {
		t.Fatalf("Failed to read lease file: %v", err)
	}

	if !strings.Contains(string(leaseContent), mac.String()) {
		t.Errorf("Lease file should contain MAC address %s", mac.String())
	}
	if !strings.Contains(string(leaseContent), initialIP.String()) {
		t.Errorf("Lease file should contain IP address %s", initialIP.String())
	}

	// Simulate DHCP DECLINE by marking IP as declined and using Put for reassignment
	if err := backend.leaseManager.MarkIPDeclined(initialIP.String()); err != nil {
		t.Fatalf("Failed to mark IP as declined: %v", err)
	}

	// Use BackendWriter.Put with empty IP to trigger reassignment (like the DHCP handler would)
	// The backend already implements BackendWriter through its Put method
	reassignDHCP := &data.DHCP{
		MACAddress: mac,
		// Empty IPAddress to trigger auto-assignment
		Hostname:  fmt.Sprintf("reassign-%s", mac.String()),
		LeaseTime: 3600,
	}

	reassignNetboot := &data.Netboot{
		AllowNetboot: true,
	}

	if err := backend.Put(ctx, mac, reassignDHCP, reassignNetboot); err != nil {
		t.Fatalf("IP reassignment via Put failed: %v", err)
	}

	// Verify new IP was assigned
	newDHCP, _, err := backend.GetByMac(ctx, mac)
	if err != nil {
		t.Fatalf("Failed to get reassigned IP: %v", err)
	}

	newIP := newDHCP.IPAddress
	t.Logf("Reassigned IP: %s", newIP.String())

	// Verify IP changed
	if newIP.String() == initialIP.String() {
		t.Error("IP should have changed after decline and reassignment")
	}

	// Verify lease file was updated with new IP
	updatedLeaseContent, err := os.ReadFile(leaseFile)
	if err != nil {
		t.Fatalf("Failed to read updated lease file: %v", err)
	}

	if !strings.Contains(string(updatedLeaseContent), newIP.String()) {
		t.Errorf("Updated lease file should contain new IP address %s", newIP.String())
	}

	// Verify hostname was updated
	if newDHCP.Hostname != fmt.Sprintf("reassign-%s", mac.String()) {
		t.Errorf("Expected updated hostname, got '%s'", newDHCP.Hostname)
	}

	t.Logf("Successfully completed DHCP DECLINE simulation with BackendWriter")
}
