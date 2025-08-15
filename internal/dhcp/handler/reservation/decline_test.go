package reservation

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

// mockLeaseManager implements the LeaseManager interface for testing.
type mockLeaseManager struct {
	declinedIPs map[string]int64
}

func (m *mockLeaseManager) MarkIPDeclined(ip string) error {
	if m.declinedIPs == nil {
		m.declinedIPs = make(map[string]int64)
	}
	m.declinedIPs[ip] = time.Now().Unix()
	return nil
}

func (m *mockLeaseManager) IsIPDeclined(ip string) bool {
	if m.declinedIPs == nil {
		return false
	}
	declineTime, exists := m.declinedIPs[ip]
	if !exists {
		return false
	}
	// 5 minute cooldown period
	return time.Now().Unix()-declineTime < 300
}

func (m *mockLeaseManager) ClearDeclinedIPs() error {
	// Clear old declined IPs (for testing, just clear all)
	m.declinedIPs = make(map[string]int64)
	return nil
}

func TestHandler_handleDecline(t *testing.T) {
	// Create a DHCP DECLINE packet
	decline, err := dhcpv4.NewDiscovery(net.HardwareAddr{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc})
	if err != nil {
		t.Fatalf("Failed to create test packet: %v", err)
	}

	// Convert to DECLINE and add requested IP
	decline.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeDecline))
	decline.UpdateOption(dhcpv4.OptRequestedIPAddress(net.ParseIP("192.168.1.100")))

	mockLM := &mockLeaseManager{}

	handler := &Handler{
		LeaseBackend: mockLM,
		Log:          logr.Discard(),
		IPAddr:       netip.MustParseAddr("192.168.1.1"),
	}

	// Test handleDecline
	handler.handleDecline(context.Background(), decline, logr.Discard())

	// Verify IP was marked as declined
	if !mockLM.IsIPDeclined("192.168.1.100") {
		t.Error("Expected IP to be marked as declined")
	}
}

func TestHandler_hasIPConflict(t *testing.T) {
	mockLM := &mockLeaseManager{}

	handler := &Handler{
		LeaseBackend: mockLM,
		Log:          logr.Discard(),
		IPAddr:       netip.MustParseAddr("192.168.1.1"),
	}

	testIP := netip.MustParseAddr("192.168.1.100")

	// Initially no conflict
	if handler.hasIPConflict(context.Background(), testIP) {
		t.Error("Expected no conflict for fresh IP")
	}

	// Mark IP as declined
	mockLM.MarkIPDeclined(testIP.String())

	// Now should have conflict
	if !handler.hasIPConflict(context.Background(), testIP) {
		t.Error("Expected conflict for declined IP")
	}
}

func TestHandler_createNAK(t *testing.T) {
	request, err := dhcpv4.NewRequestFromOffer(&dhcpv4.DHCPv4{})
	if err != nil {
		t.Fatalf("Failed to create test request: %v", err)
	}

	handler := &Handler{
		Log:    logr.Discard(),
		IPAddr: netip.MustParseAddr("192.168.1.1"),
	}

	nak := handler.createNAK(request, "Test NAK message")

	if nak.MessageType() != dhcpv4.MessageTypeNak {
		t.Errorf("Expected NAK message type, got %v", nak.MessageType())
	}

	if !nak.ServerIPAddr.Equal(net.ParseIP("192.168.1.1")) {
		t.Errorf("Expected server IP 192.168.1.1, got %v", nak.ServerIPAddr)
	}
}

func TestLeaseManagerAdapter(t *testing.T) {
	// Test with nil backend
	adapter := &LeaseManagerAdapter{backend: nil}

	// Test invalid IP handling
	err := adapter.MarkIPDeclined("invalid-ip")
	if err == nil {
		t.Error("Expected error for invalid IP address")
	}

	// Test valid IP with nil backend (should return error for MarkIPDeclined)
	err = adapter.MarkIPDeclined("192.168.1.1")
	if err == nil {
		t.Error("Expected error for nil backend on MarkIPDeclined")
	}

	// IsIPDeclined should return false for nil backend
	valid := adapter.IsIPDeclined("192.168.1.1")
	if valid {
		t.Error("Expected false for nil backend")
	}

	// ClearDeclinedIPs should not error even with nil backend
	err = adapter.ClearDeclinedIPs()
	if err != nil {
		t.Errorf("Unexpected error clearing declined IPs: %v", err)
	}
}
