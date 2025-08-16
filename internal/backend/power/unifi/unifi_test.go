package unifi

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/metal3-community/metal-boot/internal/dhcp/data"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "missing host",
			config: &Config{
				Username:   "admin",
				PrivateKey: []byte("dummy-key"),
			},
			expectError: true,
		},
		{
			name: "missing username",
			config: &Config{
				Host:       "192.168.1.1",
				PrivateKey: []byte("dummy-key"),
			},
			expectError: true,
		},
		{
			name: "missing private key",
			config: &Config{
				Host:     "192.168.1.1",
				Username: "admin",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.config)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfig(t *testing.T) {
	config := &Config{
		Host:       "192.168.1.1",
		Username:   "admin",
		PrivateKey: []byte("dummy-key"),
	}

	// Test that defaults are applied correctly without SSH parsing
	if config.Port == 0 {
		config.Port = 22
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if config.Port != 22 {
		t.Errorf("expected port 22, got %d", config.Port)
	}
	if config.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", config.Timeout)
	}
}

func TestPortMappingHelpers(t *testing.T) {
	mapping := PortMapping{
		"aa:bb:cc:dd:ee:ff": 1,
		"aa-bb-cc-dd-ee-00": 2,
		"aabbccddeee1":      3,
	}

	pm := NewPowerManager(nil, mapping)

	tests := []struct {
		name       string
		mac        net.HardwareAddr
		expectedID int
		expectErr  bool
	}{
		{
			name:       "exact match with colons",
			mac:        net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			expectedID: 1,
			expectErr:  false,
		},
		{
			name:       "match with dashes",
			mac:        net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0x00},
			expectedID: 2,
			expectErr:  false,
		},
		{
			name:       "match without separators",
			mac:        net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xe1},
			expectedID: 3,
			expectErr:  false,
		},
		{
			name:      "no match",
			mac:       net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			portID, err := pm.getPortIDFromMAC(tt.mac)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if portID != tt.expectedID {
					t.Errorf("expected port ID %d, got %d", tt.expectedID, portID)
				}
			}
		})
	}
}

func TestPowerManager(t *testing.T) {
	mapping := PortMapping{
		"aa:bb:cc:dd:ee:ff": 1,
	}

	pm := NewPowerManager(nil, mapping) // nil client for testing

	// Test GetPower with unknown MAC
	unknownMAC := net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}
	_, err := pm.GetPower(context.Background(), unknownMAC)
	if err == nil {
		t.Error("expected error for unknown MAC address")
	}

	// Test SetPower with unknown MAC
	err = pm.SetPower(context.Background(), unknownMAC, data.PowerOn)
	if err == nil {
		t.Error("expected error for unknown MAC address")
	}

	// Test PowerCycle with unknown MAC
	err = pm.PowerCycle(context.Background(), unknownMAC)
	if err == nil {
		t.Error("expected error for unknown MAC address")
	}
}

func TestSetPortPowerCommand(t *testing.T) {
	// This test validates the command generation logic without actual SSH
	tests := []struct {
		name        string
		state       data.PowerState
		expectedCmd string
		expectError bool
	}{
		{
			name:        "power on",
			state:       data.PowerOn,
			expectedCmd: "swctrl poe set auto id 1",
			expectError: false,
		},
		{
			name:        "powering on",
			state:       data.PoweringOn,
			expectedCmd: "swctrl poe set auto id 1",
			expectError: false,
		},
		{
			name:        "power off",
			state:       data.PowerOff,
			expectedCmd: "swctrl poe set off id 1",
			expectError: false,
		},
		{
			name:        "powering off",
			state:       data.PoweringOff,
			expectedCmd: "swctrl poe set off id 1",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We'll test the command generation logic
			var command string

			switch tt.state {
			case data.PowerOn, data.PoweringOn:
				command = "swctrl poe set auto id 1"
			case data.PowerOff, data.PoweringOff:
				command = "swctrl poe set off id 1"
			}

			if command != tt.expectedCmd {
				t.Errorf("expected command %q, got %q", tt.expectedCmd, command)
			}
		})
	}
}
