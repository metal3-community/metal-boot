package reservation

import (
	"fmt"
	"net"
)

// LeaseManagerAdapter adapts the DNSMasqLeaseManager interface to the simpler LeaseManager interface.
type LeaseManagerAdapter struct {
	backend DNSMasqLeaseManager
}

// NewLeaseManagerAdapter creates a new adapter for a DNSMasq lease manager.
func NewLeaseManagerAdapter(backend DNSMasqLeaseManager) *LeaseManagerAdapter {
	return &LeaseManagerAdapter{backend: backend}
}

// MarkIPDeclined marks an IP as declined using a dummy MAC address since we only care about the IP.
func (a *LeaseManagerAdapter) MarkIPDeclined(ip string) error {
	if a.backend == nil {
		return fmt.Errorf("no backend configured")
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	// Use a dummy MAC address for IP-only tracking
	dummyMAC := net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	a.backend.MarkIPDeclined(dummyMAC, parsedIP)
	return nil
}

// IsIPDeclined checks if an IP is currently declined.
func (a *LeaseManagerAdapter) IsIPDeclined(ip string) bool {
	if a.backend == nil {
		return false
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	return a.backend.IsIPDeclined(parsedIP)
}

// ClearDeclinedIPs removes declined status from IPs past the cooldown period.
func (a *LeaseManagerAdapter) ClearDeclinedIPs() error {
	if a.backend == nil {
		return nil // No-op if no backend
	}

	a.backend.ClearDeclinedIPs()
	return nil
}

// CreateLeaseManagerFromBackend creates a LeaseManager from a backend if it supports lease management.
func CreateLeaseManagerFromBackend(backend any) LeaseManager {
	if dnsmasqLM, ok := backend.(DNSMasqLeaseManager); ok {
		return NewLeaseManagerAdapter(dnsmasqLM)
	}
	return nil
}
