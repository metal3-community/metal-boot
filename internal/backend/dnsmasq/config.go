// Package dnsmasq provides DNSMasq-compatible configuration management.
package dnsmasq

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
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

// HostEntry represents a host configuration entry from ironic files.
type HostEntry struct {
	MAC        net.HardwareAddr
	NodeID     string
	TagID      string
	ShouldBoot bool
}

// ConfigManager handles DNSMasq DHCP option configuration files with file watching.
type ConfigManager struct {
	// RootDir is the dnsmasq configuration root directory
	RootDir string
	// HostsDir is the hosts directory under RootDir
	HostsDir string
	// OptsDir is the opts directory under RootDir
	OptsDir string

	// Logger for the ConfigManager
	Log logr.Logger

	// Data protection
	dataMu sync.RWMutex // protects options and hosts

	// options stores the parsed DHCP options
	options []*DHCPOption
	// hosts stores the parsed host entries
	hosts map[string]*HostEntry

	// File watching
	watcher   *fsnotify.Watcher
	watcherMu sync.RWMutex // protects watcher operations
}

// NewConfigManager creates a new configuration manager with file watching capabilities.
func NewConfigManager(log logr.Logger, rootDir string) (*ConfigManager, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	cm := &ConfigManager{
		RootDir:  rootDir,
		HostsDir: filepath.Join(rootDir, "hosts"),
		OptsDir:  filepath.Join(rootDir, "opts"),
		Log:      log,
		options:  make([]*DHCPOption, 0),
		hosts:    make(map[string]*HostEntry),
		watcher:  watcher,
	}

	// Load initial data
	if err := cm.LoadConfig(); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to load initial config data: %w", err)
	}

	// Add directories to watcher if they exist
	if err := cm.setupWatchers(); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to setup file watchers: %w", err)
	}

	return cm, nil
}

// setupWatchers adds directories to the file watcher.
func (c *ConfigManager) setupWatchers() error {
	c.watcherMu.Lock()
	defer c.watcherMu.Unlock()

	// Watch hosts directory
	if _, err := os.Stat(c.HostsDir); err == nil {
		if err := c.watcher.Add(c.HostsDir); err != nil {
			return fmt.Errorf("failed to add hosts directory to watcher: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check hosts directory: %w", err)
	}

	// Watch opts directory
	if _, err := os.Stat(c.OptsDir); err == nil {
		if err := c.watcher.Add(c.OptsDir); err != nil {
			return fmt.Errorf("failed to add opts directory to watcher: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check opts directory: %w", err)
	}

	return nil
}

func (c *ConfigManager) GetHost(mac net.HardwareAddr) (*HostEntry, bool) {
	c.dataMu.RLock()
	defer c.dataMu.RUnlock()

	host, ok := c.hosts[mac.String()]
	return host, ok
}

// LoadConfig reads host configurations and DHCP options from the directory structure.
// It reads ironic-*.conf files from the hosts directory and corresponding option files
// from the opts directory.
func (c *ConfigManager) LoadConfig() error {
	c.dataMu.Lock()
	defer c.dataMu.Unlock()

	// Clear existing data
	c.options = make([]*DHCPOption, 0)
	c.hosts = make(map[string]*HostEntry)

	// Read host files from hosts directory
	if err := c.loadHostFiles(); err != nil {
		return fmt.Errorf("failed to load host files: %w", err)
	}

	// Read option files for enabled hosts
	if err := c.loadOptionFiles(); err != nil {
		return fmt.Errorf("failed to load option files: %w", err)
	}

	return nil
}

// loadHostFiles reads all ironic-*.conf files from the hosts directory.
func (c *ConfigManager) loadHostFiles() error {
	pattern := filepath.Join(c.HostsDir, "ironic-*.conf")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob host files: %w", err)
	}

	for _, hostFile := range matches {
		if err := c.loadHostFile(hostFile); err != nil {
			// Log the error but continue with other files
			fmt.Fprintf(os.Stderr, "Warning: failed to load host file %s: %v\n", hostFile, err)
		}
	}

	return nil
}

// loadHostFile reads a single host configuration file.
func (c *ConfigManager) loadHostFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open host file %s: %w", filename, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		entry, err := c.parseHostLine(line)
		if err != nil {
			return fmt.Errorf("failed to parse host line in %s: %w", filename, err)
		}

		c.hosts[entry.MAC.String()] = entry
	}

	return scanner.Err()
}

// parseHostLine parses a host configuration line.
// Format 1: ${mac},set:${node_id},set:ironic
// Format 2: ${mac},ignore
// Do not boot when ignore.
func (c *ConfigManager) parseHostLine(line string) (*HostEntry, error) {
	parts := strings.Split(line, ",")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid host line format: %s", line)
	}

	// Parse MAC address
	mac, err := net.ParseMAC(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid MAC address: %s", parts[0])
	}

	entry := &HostEntry{
		MAC: mac,
	}

	// Check if this is an ignore entry
	if len(parts) == 2 && parts[1] == "ignore" {
		entry.ShouldBoot = false
		return entry, nil
	}

	// Parse set:node_id,set:ironic format
	if len(parts) >= 3 {
		for _, part := range parts[1:] {
			if after, ok := strings.CutPrefix(part, "set:"); ok {
				setValue := after
				if setValue != "ironic" {
					// This should be the tag ID
					entry.TagID = setValue

					// Find the matching options file and extract nodeID
					if nodeID := c.findNodeIDFromOptionsFile(setValue); nodeID != "" {
						entry.NodeID = nodeID
					}
				}
			}
		}

		// Check if this entry has both node ID and ironic tag
		if slices.Contains(parts, "set:ironic") {
			entry.ShouldBoot = true
		}
	}

	return entry, nil
}

// findNodeIDFromOptionsFile scans the options directory for a file containing the given tag
// and returns the nodeID extracted from the filename (ironic-${nodeID}.conf format).
func (c *ConfigManager) findNodeIDFromOptionsFile(tag string) string {
	// Get all option files in the opts directory
	pattern := filepath.Join(c.OptsDir, "ironic-*.conf")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return ""
	}

	// Check each file for the matching tag
	for _, optionFile := range matches {
		if c.hasTag(optionFile, tag) {
			// Extract nodeID from filename: ironic-${nodeID}.conf
			basename := filepath.Base(optionFile)
			nodeID := strings.TrimPrefix(basename, "ironic-")
			nodeID = strings.TrimSuffix(nodeID, ".conf")
			return nodeID
		}
	}

	return ""
}

// loadOptionFiles reads DHCP option files for hosts that should boot.
func (c *ConfigManager) loadOptionFiles() error {
	for _, host := range c.hosts {
		if !host.ShouldBoot || host.TagID == "" {
			continue
		}

		matches, err := filepath.Glob(filepath.Join(c.OptsDir, "ironic-*.conf"))
		if err != nil {
			return fmt.Errorf("failed to glob option files: %w", err)
		}
		for _, optionFile := range matches {
			if c.hasTag(optionFile, host.TagID) {
				if err := c.loadOptionFile(optionFile, host.MAC.String()); err != nil {
					// Log the error but continue with other files
					fmt.Fprintf(
						os.Stderr,
						"Warning: failed to load option file %s: %v\n",
						optionFile,
						err,
					)
				}
			}
		}
	}

	return nil
}

func (c *ConfigManager) hasTag(fileName string, tag string) bool {
	f, err := os.Open(fileName)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fs := strings.Split(line, ",")
		if len(fs) > 0 {
			f := fs[0]
			if after, ok := strings.CutPrefix(f, "tag:"); ok {
				t := after
				return t == tag
			}
		}
	}

	return false
}

// loadOptionFile reads DHCP options for a specific node/MAC.
func (c *ConfigManager) loadOptionFile(filename, macTag string) error {
	file, err := os.Open(filename)
	if err != nil {
		// Option file might not exist, which is okay
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open option file %s: %w", filename, err)
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
			fmt.Fprintf(
				os.Stderr,
				"Warning: failed to parse option line %d in %s: %v\n",
				lineNum,
				filename,
				err,
			)
			continue
		}

		// Override the tag with the MAC address
		option.Tag = macTag
		c.options = append(c.options, option)
	}

	return scanner.Err()
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
	c.dataMu.Lock()
	defer c.dataMu.Unlock()

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
	c.dataMu.RLock()
	defer c.dataMu.RUnlock()

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
	c.dataMu.Lock()
	defer c.dataMu.Unlock()

	var filtered []*DHCPOption
	for _, option := range c.options {
		if option.Tag != tag {
			filtered = append(filtered, option)
		}
	}
	c.options = filtered
}

// SaveConfig writes host configurations and DHCP options to the directory structure.
func (c *ConfigManager) SaveConfig() error {
	c.dataMu.RLock()
	defer c.dataMu.RUnlock()

	// Create directories if they don't exist
	if err := os.MkdirAll(c.HostsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create hosts directory: %w", err)
	}
	if err := os.MkdirAll(c.OptsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create opts directory: %w", err)
	}

	// Add directories to watcher if they were just created
	c.setupWatchersIfNeeded()

	// Save host files
	if err := c.saveHostFiles(); err != nil {
		return fmt.Errorf("failed to save host files: %w", err)
	}

	// Save option files
	if err := c.saveOptionFiles(); err != nil {
		return fmt.Errorf("failed to save option files: %w", err)
	}

	return nil
}

// setupWatchersIfNeeded adds directories to watcher if they weren't being watched.
func (c *ConfigManager) setupWatchersIfNeeded() {
	c.watcherMu.Lock()
	defer c.watcherMu.Unlock()

	// Try to add hosts directory if not already watching
	if _, err := os.Stat(c.HostsDir); err == nil {
		c.watcher.Add(c.HostsDir) // Ignore errors as directory might already be watched
	}

	// Try to add opts directory if not already watching
	if _, err := os.Stat(c.OptsDir); err == nil {
		c.watcher.Add(c.OptsDir) // Ignore errors as directory might already be watched
	}
}

// saveHostFiles writes host configuration files.
func (c *ConfigManager) saveHostFiles() error {
	for _, host := range c.hosts {
		filename := filepath.Join(c.HostsDir, fmt.Sprintf("ironic-%s.conf", host.MAC.String()))
		if err := c.saveHostFile(filename, host); err != nil {
			return fmt.Errorf("failed to save host file %s: %w", filename, err)
		}
	}
	return nil
}

// saveHostFile writes a single host configuration file.
func (c *ConfigManager) saveHostFile(filename string, host *HostEntry) error {
	tmpFile := filename + ".tmp"
	file, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary host file: %w", err)
	}
	defer file.Close()

	// Write host entry
	if host.ShouldBoot && host.NodeID != "" {
		fmt.Fprintf(file, "%s,set:%s,set:ironic\n", host.MAC.String(), host.NodeID)
	} else {
		fmt.Fprintf(file, "%s,ignore\n", host.MAC.String())
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close temporary host file: %w", err)
	}

	// Atomically replace the original file
	if err := os.Rename(tmpFile, filename); err != nil {
		return fmt.Errorf("failed to replace host file: %w", err)
	}

	return nil
}

// saveOptionFiles writes DHCP option files for enabled hosts.
func (c *ConfigManager) saveOptionFiles() error {
	// Group options by MAC address
	optionsByMAC := make(map[string][]*DHCPOption)
	for _, option := range c.options {
		optionsByMAC[option.Tag] = append(optionsByMAC[option.Tag], option)
	}

	// Write option files for each MAC that has boot enabled
	for macStr, options := range optionsByMAC {
		if host, exists := c.hosts[macStr]; exists && host.ShouldBoot && host.NodeID != "" {
			filename := filepath.Join(c.OptsDir, fmt.Sprintf("ironic-%s.conf", host.NodeID))
			if err := c.saveOptionFile(filename, options); err != nil {
				return fmt.Errorf("failed to save option file %s: %w", filename, err)
			}
		}
	}

	return nil
}

// saveOptionFile writes DHCP options for a specific node.
func (c *ConfigManager) saveOptionFile(filename string, options []*DHCPOption) error {
	tmpFile := filename + ".tmp"
	file, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary option file: %w", err)
	}
	defer file.Close()

	// Write header comment
	fmt.Fprintf(file, "# DHCP options configuration - DNSMasq compatible format\n")
	fmt.Fprintf(file, "# Format: tag:<tag>,tag:<conditional>,<option-code>,<value>\n")
	fmt.Fprintf(file, "#\n")

	// Write all options
	for _, option := range options {
		line := fmt.Sprintf("tag:%s", option.Tag)

		if option.ConditionalTag != "" {
			line += fmt.Sprintf(",tag:%s", option.ConditionalTag)
		}

		line += fmt.Sprintf(",%d,%s", option.OptionCode, option.Value)

		fmt.Fprintln(file, line)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close temporary option file: %w", err)
	}

	// Atomically replace the original file
	if err := os.Rename(tmpFile, filename); err != nil {
		return fmt.Errorf("failed to replace option file: %w", err)
	}

	return nil
}

// AddNetbootOptions adds standard netboot DHCP options for a MAC address.
func (c *ConfigManager) AddNetbootOptions(mac net.HardwareAddr, tftpServer, httpServer string) {
	c.AddNetbootOptionsWithBootFile(mac, tftpServer, httpServer, "ipxe.efi")
}

// AddNetbootOptionsWithBootFile adds standard netboot DHCP options for a MAC address with a specific boot file.
func (c *ConfigManager) AddNetbootOptionsWithBootFile(
	mac net.HardwareAddr,
	tftpServer, httpServer, bootFile string,
) {
	c.dataMu.Lock()
	defer c.dataMu.Unlock()

	macStr := mac.String()

	// Remove existing options for this MAC
	c.removeOptionsForTagUnsafe(macStr)

	// Create or update host entry
	if host, exists := c.hosts[macStr]; exists {
		host.ShouldBoot = true
		if host.NodeID == "" {
			// Generate a simple node ID if none exists
			host.NodeID = strings.ReplaceAll(macStr, ":", "")
		}
	} else {
		c.hosts[macStr] = &HostEntry{
			MAC:        mac,
			NodeID:     strings.ReplaceAll(macStr, ":", ""),
			ShouldBoot: true,
		}
	}

	// Add standard netboot options with the specified boot file
	c.addOptionUnsafe(macStr, "!ipxe", 67, bootFile)
	c.addOptionUnsafe(macStr, "ipxe", 67, fmt.Sprintf("http://%s/boot.ipxe", httpServer))
	c.addOptionUnsafe(macStr, "", 66, tftpServer)
	c.addOptionUnsafe(macStr, "", 150, tftpServer)
	c.addOptionUnsafe(macStr, "", 255, tftpServer)

	// For ARM64 devices, use snp.efi for IPv6 as well
	if bootFile == "snp.efi" {
		c.addOptionUnsafe(macStr, "!ipxe6", 59, fmt.Sprintf("tftp://%s/snp.efi", tftpServer))
	} else {
		c.addOptionUnsafe(macStr, "!ipxe6", 59, fmt.Sprintf("tftp://%s/ipxe.efi", tftpServer))
	}
	c.addOptionUnsafe(macStr, "ipxe6", 59, fmt.Sprintf("http://%s/boot.ipxe", httpServer))
}

// addOptionUnsafe adds an option without acquiring mutex (for internal use when mutex already held).
func (c *ConfigManager) addOptionUnsafe(tag, conditionalTag string, optionCode int, value string) {
	option := &DHCPOption{
		Tag:            tag,
		ConditionalTag: conditionalTag,
		OptionCode:     optionCode,
		Value:          value,
	}
	c.options = append(c.options, option)
}

// removeOptionsForTagUnsafe removes options without acquiring mutex (for internal use when mutex already held).
func (c *ConfigManager) removeOptionsForTagUnsafe(tag string) {
	var filtered []*DHCPOption
	for _, option := range c.options {
		if option.Tag != tag {
			filtered = append(filtered, option)
		}
	}
	c.options = filtered
}

// DisableNetboot disables netbooting for a MAC address.
func (c *ConfigManager) DisableNetboot(mac net.HardwareAddr) {
	c.dataMu.Lock()
	defer c.dataMu.Unlock()

	macStr := mac.String()

	// Remove options for this MAC
	c.removeOptionsForTagUnsafe(macStr)

	// Update or create host entry to disable boot
	if host, exists := c.hosts[macStr]; exists {
		host.ShouldBoot = false
	} else {
		c.hosts[macStr] = &HostEntry{
			MAC:        mac,
			ShouldBoot: false,
		}
	}
}

// IsNetbootEnabled checks if netboot is enabled for a MAC address.
func (c *ConfigManager) IsNetbootEnabled(mac net.HardwareAddr) bool {
	c.dataMu.RLock()
	defer c.dataMu.RUnlock()

	if host, exists := c.hosts[mac.String()]; exists {
		return host.ShouldBoot
	}

	// Default to false if host entry doesn't exist
	return false
}

// GetAllOptions returns all configured DHCP options.
func (c *ConfigManager) GetAllOptions() []*DHCPOption {
	c.dataMu.RLock()
	defer c.dataMu.RUnlock()

	return c.options
}

// Start starts watching the configuration directories for changes and updates the in-memory data on changes.
// Start is a blocking method. Use a context cancellation to exit.
func (c *ConfigManager) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.Log.Info("stopping config file watcher")
			return
		case event, ok := <-c.watcher.Events:
			if !ok {
				continue
			}
			if event.Has(fsnotify.Write) ||
				event.Has(fsnotify.Create) ||
				event.Has(fsnotify.Remove) {
				c.Log.Info("config file changed, updating cache", "file", event.Name)
				if err := c.LoadConfig(); err != nil {
					c.Log.Error(err, "failed to reload config files", "file", event.Name)
				}
			}
		case err, ok := <-c.watcher.Errors:
			if !ok {
				continue
			}
			c.Log.Error(err, "error watching config files")
		}
	}
}

// Close closes the file watcher and cleans up resources.
func (c *ConfigManager) Close() error {
	c.watcherMu.Lock()
	defer c.watcherMu.Unlock()

	if c.watcher != nil {
		return c.watcher.Close()
	}
	return nil
}
