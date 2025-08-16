// Package unifi provides SSH-based power management for Unifi switches.
package unifi

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/metal3-community/metal-boot/internal/dhcp/data"
	"golang.org/x/crypto/ssh"
)

// Config holds the configuration for connecting to a Unifi switch via SSH.
type Config struct {
	// Host is the IP address or hostname of the Unifi switch
	Host string
	// Port is the SSH port (default: 22)
	Port int
	// Username for SSH authentication
	Username string
	// PrivateKey is the SSH private key for authentication
	PrivateKey []byte
	// HostKeyCallback for host key verification (use ssh.InsecureIgnoreHostKey() for testing)
	HostKeyCallback ssh.HostKeyCallback
	// Timeout for SSH operations
	Timeout time.Duration
}

// Client provides SSH-based power management for Unifi switches.
type Client struct {
	config    *Config
	sshConfig *ssh.ClientConfig
}

// PortMapping maps MAC addresses to switch ports.
type PortMapping map[string]int // MAC address (string) -> Port ID (int)

// NewClient creates a new Unifi SSH client.
func NewClient(config *Config) (*Client, error) {
	if config.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if config.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if len(config.PrivateKey) == 0 {
		return nil, fmt.Errorf("private key is required")
	}

	// Parse the private key
	signer, err := ssh.ParsePrivateKey(config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Set default values
	if config.Port == 0 {
		config.Port = 22
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.HostKeyCallback == nil {
		config.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: config.HostKeyCallback,
		Timeout:         config.Timeout,
	}

	return &Client{
		config:    config,
		sshConfig: sshConfig,
	}, nil
}

// executeCommand executes a command on the Unifi switch via SSH.
func (c *Client) executeCommand(ctx context.Context, command string) (string, error) {
	// Create SSH connection
	conn, err := ssh.Dial(
		"tcp",
		net.JoinHostPort(c.config.Host, strconv.Itoa(c.config.Port)),
		c.sshConfig,
	)
	if err != nil {
		return "", fmt.Errorf("failed to connect to SSH server: %w", err)
	}
	defer conn.Close()

	// Create a session
	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Execute the command with context timeout
	done := make(chan error, 1)
	var output []byte

	go func() {
		output, err = session.CombinedOutput(command)
		done <- err
	}()

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("command execution cancelled: %w", ctx.Err())
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("command execution failed: %w", err)
		}
	}

	return string(output), nil
}

// GetPortPowerState gets the current power state of a specific port.
func (c *Client) GetPortPowerState(ctx context.Context, portID int) (data.PowerState, error) {
	command := fmt.Sprintf("swctrl poe show id %d", portID)
	output, err := c.executeCommand(ctx, command)
	if err != nil {
		return data.PowerOff, fmt.Errorf("failed to get port power state: %w", err)
	}

	// Parse the output to determine power state
	// Expected output format varies, but typically contains POE mode information
	output = strings.ToLower(strings.TrimSpace(output))

	if strings.Contains(output, "auto") ||
		strings.Contains(output, "poe") ||
		strings.Contains(output, "plus") {
		// Check if actually providing power vs just enabled
		if strings.Contains(output, "providing") || strings.Contains(output, "enabled") {
			return data.PowerOn, nil
		}
		return data.PoweringOn, nil // POE enabled but not necessarily providing power
	} else if strings.Contains(output, "off") {
		return data.PowerOff, nil
	}

	// Default to off if we can't determine the state
	return data.PowerOff, nil
}

// SetPortPower sets the power state of a specific port.
func (c *Client) SetPortPower(ctx context.Context, portID int, state data.PowerState) error {
	var command string

	switch state {
	case data.PowerOn, data.PoweringOn:
		command = fmt.Sprintf("swctrl poe set auto id %d", portID)
	case data.PowerOff, data.PoweringOff:
		command = fmt.Sprintf("swctrl poe set off id %d", portID)
	default:
		return fmt.Errorf("unsupported power state: %s", state.String())
	}

	_, err := c.executeCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("failed to set port power state: %w", err)
	}

	return nil
}

// RestartPortPower restarts (power cycles) a specific port.
func (c *Client) RestartPortPower(ctx context.Context, portID int) error {
	command := fmt.Sprintf("swctrl poe restart id %d", portID)
	_, err := c.executeCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("failed to restart port power: %w", err)
	}

	return nil
}

// PowerManager implements the backend.BackendPower interface for Unifi switches.
type PowerManager struct {
	client      *Client
	portMapping PortMapping
}

// NewPowerManager creates a new PowerManager instance.
func NewPowerManager(client *Client, portMapping PortMapping) *PowerManager {
	return &PowerManager{
		client:      client,
		portMapping: portMapping,
	}
}

// getPortIDFromMAC retrieves the port ID for a given MAC address.
func (pm *PowerManager) getPortIDFromMAC(mac net.HardwareAddr) (int, error) {
	macStr := strings.ToLower(mac.String())

	// Try exact match first
	if portID, exists := pm.portMapping[macStr]; exists {
		return portID, nil
	}

	// Try without colons
	macStrNoColons := strings.ReplaceAll(macStr, ":", "")
	if portID, exists := pm.portMapping[macStrNoColons]; exists {
		return portID, nil
	}

	// Try with dashes
	macStrDashes := strings.ReplaceAll(macStr, ":", "-")
	if portID, exists := pm.portMapping[macStrDashes]; exists {
		return portID, nil
	}

	return 0, fmt.Errorf("no port mapping found for MAC address %s", macStr)
}

// GetPower gets the current power state for a device by MAC address.
func (pm *PowerManager) GetPower(
	ctx context.Context,
	mac net.HardwareAddr,
) (*data.PowerState, error) {
	portID, err := pm.getPortIDFromMAC(mac)
	if err != nil {
		return nil, err
	}

	state, err := pm.client.GetPortPowerState(ctx, portID)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

// SetPower sets the power state for a device by MAC address.
func (pm *PowerManager) SetPower(
	ctx context.Context,
	mac net.HardwareAddr,
	state data.PowerState,
) error {
	portID, err := pm.getPortIDFromMAC(mac)
	if err != nil {
		return err
	}

	return pm.client.SetPortPower(ctx, portID, state)
}

// PowerCycle power cycles a device by MAC address.
func (pm *PowerManager) PowerCycle(ctx context.Context, mac net.HardwareAddr) error {
	portID, err := pm.getPortIDFromMAC(mac)
	if err != nil {
		return err
	}

	return pm.client.RestartPortPower(ctx, portID)
}
