// Package arp provides ARP-based IP conflict detection for DHCP servers.
package arp

import (
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/go-logr/logr"
	"github.com/mdlayher/arp"
)

// ConflictDetector provides ARP-based IP address conflict detection.
type ConflictDetector struct {
	// InterfaceName is the network interface to use for ARP probes
	InterfaceName string
	// Log is the logger for ARP operations
	Log logr.Logger
	// ProbeCount is the number of ARP probes to send (default: 3)
	ProbeCount int
	// ProbeInterval is the time between ARP probes (default: 100ms)
	ProbeInterval time.Duration
}

// NewConflictDetector creates a new ARP conflict detector.
func NewConflictDetector(interfaceName string, log logr.Logger) *ConflictDetector {
	return &ConflictDetector{
		InterfaceName: interfaceName,
		Log:           log,
		ProbeCount:    3,
		ProbeInterval: 100 * time.Millisecond,
	}
}

// IsIPInUse checks if an IP address is already in use on the network.
// It sends multiple ARP requests to increase reliability.
func (cd *ConflictDetector) IsIPInUse(ip net.IP) bool {
	if cd.InterfaceName == "" {
		cd.Log.Info("ARP conflict detection disabled: no interface specified")
		return false
	}

	// Convert net.IP to netip.Addr
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		cd.Log.Error(fmt.Errorf("invalid IP address"), "ARP: invalid IP", "ip", ip.String())
		return false
	}

	iface, err := net.InterfaceByName(cd.InterfaceName)
	if err != nil {
		cd.Log.Error(err, "ARP: failed to get interface", "interface", cd.InterfaceName)
		return false
	}

	client, err := arp.Dial(iface)
	if err != nil {
		cd.Log.Error(err, "ARP: failed to create client", "interface", cd.InterfaceName)
		return false
	}
	defer client.Close()

	// Send multiple probes for reliability
	for i := 0; i < cd.ProbeCount; i++ {
		if i > 0 {
			time.Sleep(cd.ProbeInterval)
		}

		// Set a reasonable timeout for each probe
		client.SetReadDeadline(time.Now().Add(time.Second))

		hwAddr, err := client.Resolve(addr)
		if err == nil && !isZeroMAC(hwAddr) && !isBroadcastMAC(hwAddr) {
			cd.Log.Info("ARP: IP address is in use",
				"ip", ip.String(),
				"mac", hwAddr.String(),
				"probe", i+1)
			return true
		}

		// Log detailed error for debugging
		if err != nil {
			cd.Log.V(1).Info("ARP: probe failed",
				"ip", ip.String(),
				"probe", i+1,
				"error", err.Error())
		}
	}

	cd.Log.V(1).Info("ARP: IP address appears to be free",
		"ip", ip.String(),
		"probes", cd.ProbeCount)
	return false
}

// IsIPInUseWithTimeout checks if an IP is in use with a specific timeout per probe.
func (cd *ConflictDetector) IsIPInUseWithTimeout(ip net.IP, timeout time.Duration) bool {
	if cd.InterfaceName == "" {
		cd.Log.Info("ARP conflict detection disabled: no interface specified")
		return false
	}

	// Convert net.IP to netip.Addr
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		cd.Log.Error(fmt.Errorf("invalid IP address"), "ARP: invalid IP", "ip", ip.String())
		return false
	}

	iface, err := net.InterfaceByName(cd.InterfaceName)
	if err != nil {
		cd.Log.Error(err, "ARP: failed to get interface", "interface", cd.InterfaceName)
		return false
	}

	client, err := arp.Dial(iface)
	if err != nil {
		cd.Log.Error(err, "ARP: failed to create client", "interface", cd.InterfaceName)
		return false
	}
	defer client.Close()

	// Send multiple probes for reliability
	for i := 0; i < cd.ProbeCount; i++ {
		if i > 0 {
			time.Sleep(cd.ProbeInterval)
		}

		client.SetReadDeadline(time.Now().Add(timeout))

		hwAddr, err := client.Resolve(addr)
		if err == nil && !isZeroMAC(hwAddr) && !isBroadcastMAC(hwAddr) {
			cd.Log.Info("ARP: IP address is in use",
				"ip", ip.String(),
				"mac", hwAddr.String(),
				"probe", i+1)
			return true
		}
	}

	cd.Log.V(1).Info("ARP: IP address appears to be free",
		"ip", ip.String(),
		"probes", cd.ProbeCount)
	return false
}

// ProbeIP sends a single ARP probe and returns the responding MAC address if any.
func (cd *ConflictDetector) ProbeIP(ip net.IP) (net.HardwareAddr, error) {
	if cd.InterfaceName == "" {
		return nil, fmt.Errorf("no interface specified for ARP probe")
	}

	// Convert net.IP to netip.Addr
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return nil, fmt.Errorf("invalid IP address: %s", ip.String())
	}

	iface, err := net.InterfaceByName(cd.InterfaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface %s: %w", cd.InterfaceName, err)
	}

	client, err := arp.Dial(iface)
	if err != nil {
		return nil, fmt.Errorf("failed to create ARP client: %w", err)
	}
	defer client.Close()

	client.SetReadDeadline(time.Now().Add(time.Second))

	hwAddr, err := client.Resolve(addr)
	if err != nil {
		return nil, err
	}

	if isBroadcastMAC(hwAddr) || isZeroMAC(hwAddr) {
		return nil, fmt.Errorf("no valid response")
	}

	return hwAddr, nil
}

// isZeroMAC checks if a MAC address is all zeros.
func isZeroMAC(mac net.HardwareAddr) bool {
	for _, b := range mac {
		if b != 0 {
			return false
		}
	}
	return true
}

// isBroadcastMAC checks if a MAC address is the broadcast address.
func isBroadcastMAC(mac net.HardwareAddr) bool {
	for _, b := range mac {
		if b != 0xff {
			return false
		}
	}
	return len(mac) == 6 // Ethernet broadcast is 6 bytes of 0xff
}
