package arp

import (
	"net"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

func TestConflictDetector_IsIPInUse(t *testing.T) {
	tests := []struct {
		name          string
		interfaceName string
		ip            net.IP
		want          bool
		wantErr       bool
	}{
		{
			name:          "disabled when no interface",
			interfaceName: "",
			ip:            net.ParseIP("192.168.1.1"),
			want:          false,
			wantErr:       false,
		},
		{
			name:          "invalid interface returns false",
			interfaceName: "nonexistent-interface",
			ip:            net.ParseIP("192.168.1.1"),
			want:          false,
			wantErr:       false,
		},
		{
			name:          "localhost should not be in use",
			interfaceName: "lo",                         // loopback interface should exist on most systems
			ip:            net.ParseIP("192.168.99.99"), // likely unused IP
			want:          false,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd := &ConflictDetector{
				InterfaceName: tt.interfaceName,
				Log:           logr.Discard(),
				ProbeCount:    1, // Reduce probe count for faster tests
				ProbeInterval: 10 * time.Millisecond,
			}

			got := cd.IsIPInUse(tt.ip)
			if got != tt.want {
				t.Errorf("ConflictDetector.IsIPInUse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConflictDetector_NewConflictDetector(t *testing.T) {
	cd := NewConflictDetector("eth0", logr.Discard())

	if cd.InterfaceName != "eth0" {
		t.Errorf("Expected interface name 'eth0', got %s", cd.InterfaceName)
	}

	if cd.ProbeCount != 3 {
		t.Errorf("Expected default probe count 3, got %d", cd.ProbeCount)
	}

	if cd.ProbeInterval != 100*time.Millisecond {
		t.Errorf("Expected default probe interval 100ms, got %v", cd.ProbeInterval)
	}
}

func TestBroadcastAndZeroMAC(t *testing.T) {
	broadcastMAC := net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	zeroMAC := net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	normalMAC := net.HardwareAddr{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc}

	if !isBroadcastMAC(broadcastMAC) {
		t.Error("Expected broadcast MAC to be detected as broadcast")
	}

	if isBroadcastMAC(normalMAC) {
		t.Error("Expected normal MAC not to be detected as broadcast")
	}

	if !isZeroMAC(zeroMAC) {
		t.Error("Expected zero MAC to be detected as zero")
	}

	if isZeroMAC(normalMAC) {
		t.Error("Expected normal MAC not to be detected as zero")
	}
}
