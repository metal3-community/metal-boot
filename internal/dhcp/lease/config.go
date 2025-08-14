// Package lease provides DNSMasq-compatible configuration management.
package lease

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// DHCPOption represents a single DHCP option configuration entry.
type DHCPOption struct {
	// Tag identifies the option set (typically a UUID or MAC-based identifier)
	Tag string
	// ConditionalTag is an optional condition (e.g., "!ipxe", "ipxe6")
	ConditionalTag string
	// OptionCode is the DHCP option number (e.g., 67, 66, 150)
	OptionCode int
	// Value is the option value
	Value string
}

// ConfigManager handles DNSMasq DHCP option configuration files.
type ConfigManager struct {
	// ConfigFile is the path to the configuration file
	ConfigFile string
	// options stores the parsed DHCP options
	options []*DHCPOption
}

// NewConfigManager creates a new configuration manager.
func NewConfigManager(configFile string) *ConfigManager {
	return &ConfigManager{
		ConfigFile: configFile,
		options:    make([]*DHCPOption, 0),
	}
}

// LoadConfig reads and parses the DNSMasq DHCP options configuration file.
// Format: tag:<tag>,tag:<conditional>,<option-code>,<value>
// Example: tag:92ef984e-995f-4aea-8088-9cde6a970a88,tag:!ipxe,67,ipxe.efi.
func (c *ConfigManager) LoadConfig() error {
	file, err := os.Open(c.ConfigFile)
	if err != nil {
		// If file doesn't exist, that's OK - we'll create it when we write
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open config file %s: %w", c.ConfigFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		option, err := c.parseOptionLine(line)
		if err != nil {
			// Log the error but continue parsing other options
			fmt.Fprintf(os.Stderr, "Warning: failed to parse option line %d: %v\n", lineNum, err)
			continue
		}

		c.options = append(c.options, option)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	return nil
}

// parseOptionLine parses a single DHCP option line.
// Format: tag:<tag>,tag:<conditional>,<option-code>,<value>.
func (c *ConfigManager) parseOptionLine(line string) (*DHCPOption, error) {
	// Split by comma
	parts := strings.Split(line, ",")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid option line format: %s", line)
	}

	option := &DHCPOption{}

	// Parse tags and option code
	var optionCodeIndex int
	for i, part := range parts {
		if after, ok := strings.CutPrefix(part, "tag:"); ok {
			tag := after
			if option.Tag == "" {
				option.Tag = tag
			} else {
				// This is a conditional tag
				option.ConditionalTag = tag
			}
		} else {
			// This should be the option code
			if optionCode, err := parseInt(part); err == nil {
				option.OptionCode = optionCode
				optionCodeIndex = i
				break
			} else {
				return nil, fmt.Errorf("invalid option code: %s", part)
			}
		}
	}

	// Everything after the option code is the value
	if optionCodeIndex+1 < len(parts) {
		option.Value = strings.Join(parts[optionCodeIndex+1:], ",")
	}

	return option, nil
}

// parseInt safely parses an integer from a string.
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// AddOption adds a new DHCP option.
func (c *ConfigManager) AddOption(tag, conditionalTag string, optionCode int, value string) {
	option := &DHCPOption{
		Tag:            tag,
		ConditionalTag: conditionalTag,
		OptionCode:     optionCode,
		Value:          value,
	}
	c.options = append(c.options, option)
}

// GetOptions returns all configured DHCP options for a specific tag.
func (c *ConfigManager) GetOptions(tag string) []*DHCPOption {
	var result []*DHCPOption
	for _, option := range c.options {
		if option.Tag == tag {
			result = append(result, option)
		}
	}
	return result
}

// RemoveOptionsForTag removes all DHCP options for a specific tag.
func (c *ConfigManager) RemoveOptionsForTag(tag string) {
	var filtered []*DHCPOption
	for _, option := range c.options {
		if option.Tag != tag {
			filtered = append(filtered, option)
		}
	}
	c.options = filtered
}

// SaveConfig writes all DHCP options to the configuration file.
func (c *ConfigManager) SaveConfig() error {
	// Create directory if it doesn't exist
	if dir := filepath.Dir(c.ConfigFile); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// Write to temporary file first
	tmpFile := c.ConfigFile + ".tmp"
	file, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary config file: %w", err)
	}
	defer file.Close()

	// Write header comment
	fmt.Fprintf(file, "# DHCP options configuration - DNSMasq compatible format\n")
	fmt.Fprintf(file, "# Format: tag:<tag>,tag:<conditional>,<option-code>,<value>\n")
	fmt.Fprintf(file, "#\n")

	// Write all options
	for _, option := range c.options {
		line := fmt.Sprintf("tag:%s", option.Tag)

		if option.ConditionalTag != "" {
			line += fmt.Sprintf(",tag:%s", option.ConditionalTag)
		}

		line += fmt.Sprintf(",%d,%s", option.OptionCode, option.Value)

		fmt.Fprintln(file, line)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close temporary config file: %w", err)
	}

	// Atomically replace the original file
	if err := os.Rename(tmpFile, c.ConfigFile); err != nil {
		return fmt.Errorf("failed to replace config file: %w", err)
	}

	return nil
}

// AddNetbootOptions adds standard netboot DHCP options for a MAC address.
func (c *ConfigManager) AddNetbootOptions(mac net.HardwareAddr, tftpServer, httpServer string) {
	tag := mac.String()

	// Remove existing options for this MAC
	c.RemoveOptionsForTag(tag)

	// Option 67: Boot filename
	c.AddOption(tag, "!ipxe", 67, "ipxe.efi")
	c.AddOption(tag, "ipxe", 67, fmt.Sprintf("http://%s/boot.ipxe", httpServer))

	// Option 66: TFTP server name
	c.AddOption(tag, "", 66, tftpServer)

	// Option 150: TFTP server address (Cisco extension)
	c.AddOption(tag, "", 150, tftpServer)

	// Option 255: End option (sometimes needed)
	c.AddOption(tag, "", 255, tftpServer)

	// IPv6 variants
	c.AddOption(tag, "!ipxe6", 59, fmt.Sprintf("tftp://%s/ipxe.efi", tftpServer))
	c.AddOption(tag, "ipxe6", 59, fmt.Sprintf("http://%s/boot.ipxe", httpServer))
}

// GetAllOptions returns all configured DHCP options.
func (c *ConfigManager) GetAllOptions() []*DHCPOption {
	return c.options
}
