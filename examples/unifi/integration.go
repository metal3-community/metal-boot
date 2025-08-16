// Package integration shows an example of how to integrate the Unifi SSH power management
// with the PiBMC system's backend architecture.
package integration

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/metal3-community/metal-boot/internal/backend"
	"github.com/metal3-community/metal-boot/internal/backend/power/unifi"
	"github.com/metal3-community/metal-boot/internal/dhcp/data"
	"golang.org/x/crypto/ssh"
)

// UnifiBackendConfig holds configuration for integrating Unifi power management
// into the PiBMC backend system.
type UnifiBackendConfig struct {
	// SSH connection settings
	SwitchHost     string
	SwitchPort     int
	Username       string
	PrivateKeyPath string

	// Port mapping file or configuration
	PortMappingFile string

	// Static port mapping (alternative to file-based)
	StaticPortMapping map[string]int
}

// UnifiPowerBackend implements backend.BackendPower using Unifi SSH.
type UnifiPowerBackend struct {
	powerManager *unifi.PowerManager
	config       *UnifiBackendConfig
}

// NewUnifiPowerBackend creates a new backend that implements BackendPower
// for Unifi switches via SSH.
func NewUnifiPowerBackend(config *UnifiBackendConfig) (*UnifiPowerBackend, error) {
	// Read the SSH private key
	privateKey, err := ioutil.ReadFile(config.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH private key: %w", err)
	}

	// Create the SSH client configuration
	sshConfig := &unifi.Config{
		Host:       config.SwitchHost,
		Port:       config.SwitchPort,
		Username:   config.Username,
		PrivateKey: privateKey,
		// In production, use proper host key verification
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Create the SSH client
	client, err := unifi.NewClient(sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH client: %w", err)
	}

	// Create port mapping
	var portMapping unifi.PortMapping
	if config.StaticPortMapping != nil {
		portMapping = make(unifi.PortMapping)
		for mac, port := range config.StaticPortMapping {
			portMapping[mac] = port
		}
	} else {
		// In a real implementation, you would load this from config.PortMappingFile
		// For this example, we'll use a static mapping
		portMapping = unifi.PortMapping{
			"b8:27:eb:12:34:56": 1, // Raspberry Pi 1
			"b8:27:eb:78:9a:bc": 2, // Raspberry Pi 2
			"b8:27:eb:de:f0:12": 3, // Raspberry Pi 3
			"b8:27:eb:34:56:78": 4, // Raspberry Pi 4
		}
	}

	// Create the power manager
	powerManager := unifi.NewPowerManager(client, portMapping)

	return &UnifiPowerBackend{
		powerManager: powerManager,
		config:       config,
	}, nil
}

// GetPower implements backend.BackendPower.
func (u *UnifiPowerBackend) GetPower(
	ctx context.Context,
	mac net.HardwareAddr,
) (*data.PowerState, error) {
	return u.powerManager.GetPower(ctx, mac)
}

// SetPower implements backend.BackendPower.
func (u *UnifiPowerBackend) SetPower(
	ctx context.Context,
	mac net.HardwareAddr,
	state data.PowerState,
) error {
	return u.powerManager.SetPower(ctx, mac, state)
}

// PowerCycle implements backend.BackendPower.
func (u *UnifiPowerBackend) PowerCycle(ctx context.Context, mac net.HardwareAddr) error {
	return u.powerManager.PowerCycle(ctx, mac)
}

// Verify that UnifiPowerBackend implements the BackendPower interface.
var _ backend.BackendPower = (*UnifiPowerBackend)(nil)

// ExampleUsage demonstrates how to use the UnifiPowerBackend in a PiBMC setup.
func ExampleUsage() error {
	// Configuration for the Unifi backend
	config := &UnifiBackendConfig{
		SwitchHost:     "192.168.1.10",
		SwitchPort:     22,
		Username:       "pibmc-service",
		PrivateKeyPath: "/etc/pibmc/ssh/id_rsa",
		StaticPortMapping: map[string]int{
			"b8:27:eb:12:34:56": 1, // Pi Node 1
			"b8:27:eb:78:9a:bc": 2, // Pi Node 2
			"b8:27:eb:de:f0:12": 3, // Pi Node 3
			"b8:27:eb:34:56:78": 4, // Pi Node 4
		},
	}

	// Create the backend
	powerBackend, err := NewUnifiPowerBackend(config)
	if err != nil {
		return fmt.Errorf("failed to create power backend: %w", err)
	}

	// Example usage - this would typically be called by DHCP handlers or Redfish API
	ctx := context.Background()
	mac := net.HardwareAddr{0xb8, 0x27, 0xeb, 0x12, 0x34, 0x56}

	// Get current power state
	state, err := powerBackend.GetPower(ctx, mac)
	if err != nil {
		return fmt.Errorf("failed to get power state: %w", err)
	}
	fmt.Printf("Current power state: %s\n", state.String())

	// Power on the device
	err = powerBackend.SetPower(ctx, mac, data.PowerOn)
	if err != nil {
		return fmt.Errorf("failed to set power state: %w", err)
	}
	fmt.Println("Device powered on")

	// Power cycle the device
	err = powerBackend.PowerCycle(ctx, mac)
	if err != nil {
		return fmt.Errorf("failed to power cycle: %w", err)
	}
	fmt.Println("Device power cycled")

	return nil
}

// IntegrateWithDHCPHandler shows how the power backend could be integrated
// with a DHCP handler in the PiBMC system.
func IntegrateWithDHCPHandler(powerBackend backend.BackendPower) {
	// This is a conceptual example showing how the power backend
	// would be used in a DHCP handler context

	// Example: When a DHCP DECLINE is received, power cycle the device
	handleDHCPDecline := func(ctx context.Context, mac net.HardwareAddr) error {
		fmt.Printf("DHCP DECLINE received from %s, power cycling...\n", mac.String())

		err := powerBackend.PowerCycle(ctx, mac)
		if err != nil {
			return fmt.Errorf("failed to power cycle device %s: %w", mac.String(), err)
		}

		fmt.Printf("Successfully power cycled device %s\n", mac.String())
		return nil
	}

	// Example: Check power state before assigning IP
	checkPowerBeforeAssign := func(ctx context.Context, mac net.HardwareAddr) (*data.PowerState, error) {
		state, err := powerBackend.GetPower(ctx, mac)
		if err != nil {
			return nil, fmt.Errorf("failed to get power state for %s: %w", mac.String(), err)
		}

		// If device is off, power it on
		if *state == data.PowerOff {
			fmt.Printf("Device %s is off, powering on...\n", mac.String())
			err = powerBackend.SetPower(ctx, mac, data.PowerOn)
			if err != nil {
				return nil, fmt.Errorf("failed to power on device %s: %w", mac.String(), err)
			}
		}

		return state, nil
	}

	// These functions would be called by the actual DHCP handlers
	_ = handleDHCPDecline
	_ = checkPowerBeforeAssign
}
