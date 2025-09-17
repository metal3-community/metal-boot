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
