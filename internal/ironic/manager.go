package ironic

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	healthCheckInterval = 30 * time.Second
	shutdownTimeout     = 30 * time.Second
)

// ProcessManager handles supervision of Ironic processes.
type ProcessManager struct {
	mu        sync.RWMutex
	processes map[string]*Process
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	logger    *slog.Logger
	config    *Config
}

// Process represents a supervised process.
type Process struct {
	name    string
	cmd     *exec.Cmd
	healthy bool
	restart chan struct{}
	mu      sync.RWMutex
}

// NewProcessManager creates a new process manager.
func NewProcessManager(ctx context.Context, logger *slog.Logger, config *Config) *ProcessManager {
	ctx, cancel := context.WithCancel(ctx)

	// Set default paths if not configured
	if config != nil {
		if config.SocketPath == "" {
			config.SocketPath = "/tmp/ironic.sock"
		}
		if config.ConfigPath == "" {
			config.ConfigPath = "/etc/ironic/ironic.conf"
		}
		if config.SharedRoot == "" {
			config.SharedRoot = "/shared"
		}

		// Apply all default configuration values
		config.SetDefaults()

		// Set runtime-specific paths
		config.SetRuntimePaths(config.SocketPath, config.SharedRoot)
	}

	return &ProcessManager{
		processes: make(map[string]*Process),
		ctx:       ctx,
		cancel:    cancel,
		logger:    logger,
		config:    config,
	}
}

// sharedPath constructs a path relative to the configured shared root.
func (pm *ProcessManager) sharedPath(path string) string {
	if pm.config == nil || pm.config.SharedRoot == "" {
		return "/shared" + path
	}
	return pm.config.SharedRoot + path
}

// isMariaDB returns true if the database connection is not SQLite.
func (pm *ProcessManager) isMariaDB() bool {
	if pm.config == nil || pm.config.Database.Connection == "" {
		return false
	}
	return !strings.HasPrefix(pm.config.Database.Connection, "sqlite://")
}

// runIronicDbsync runs database synchronization before starting Ironic.
func (pm *ProcessManager) runIronicDbsync() error {
	pm.logger.Info("Running Ironic database synchronization")

	if pm.isMariaDB() {
		// MariaDB: retry upgrade until success
		pm.logger.Debug("Using MariaDB, running upgrade with retry logic")
		for {
			cmd := exec.CommandContext(
				pm.ctx,
				"ironic-dbsync",
				"--config-file",
				pm.config.ConfigPath,
				"upgrade",
			)
			if err := cmd.Run(); err != nil {
				pm.logger.Warn("ironic-dbsync failed, retrying", "error", err)
				select {
				case <-pm.ctx.Done():
					return fmt.Errorf("database sync cancelled: %w", pm.ctx.Err())
				case <-time.After(1 * time.Second):
					continue
				}
			}
			pm.logger.Info("Database upgrade completed successfully")
			break
		}
	} else {
		// SQLite: copy template and create schema if needed
		pm.logger.Debug("Using SQLite, checking schema version")

		// Create database directory if needed
		dbDir := pm.sharedPath("/var/lib/ironic")
		if err := os.MkdirAll(dbDir, 0o755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}

		// Copy template database if it doesn't exist
		dbPath := filepath.Join(dbDir, "ironic.sqlite")
		templatePath := "/var/lib/ironic/ironic.sqlite"

		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			if _, err := os.Stat(templatePath); err == nil {
				if err := pm.copyFile(templatePath, dbPath); err != nil {
					pm.logger.Warn("Failed to copy template database", "error", err)
				} else {
					pm.logger.Debug("Copied template database", "from", templatePath, "to", dbPath)
				}
			}
		}

		// Check database version
		versionCmd := exec.CommandContext(pm.ctx, "ironic-dbsync", "--config-file", pm.config.ConfigPath, "version")
		output, err := versionCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to check database version: %w", err)
		}

		version := strings.TrimSpace(string(output))
		pm.logger.Debug("Database version", "version", version)

		if version == "None" {
			pm.logger.Info("Creating database schema")
			createCmd := exec.CommandContext(pm.ctx, "ironic-dbsync", "--config-file", pm.config.ConfigPath, "create_schema")
			if err := createCmd.Run(); err != nil {
				return fmt.Errorf("failed to create database schema: %w", err)
			}
			pm.logger.Info("Database schema created successfully")
		} else {
			pm.logger.Info("Database schema already exists", "version", version)
		}
	}

	return nil
}

// copyFile copies a file from src to dst.
func (pm *ProcessManager) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// Start begins supervising all processes.
func (pm *ProcessManager) Start() error {
	pm.logger.Debug("Starting Ironic process supervision")

	// Ensure socket directory exists and is clean
	if err := pm.prepareSocketDir(); err != nil {
		return fmt.Errorf("failed to prepare socket directory: %w", err)
	}

	// Generate Ironic configuration
	if err := pm.generateIronicConfig(); err != nil {
		return fmt.Errorf("failed to generate Ironic config: %w", err)
	}

	// Create default policy file
	if err := pm.createDefaultPolicy(); err != nil {
		return fmt.Errorf("failed to create default policy: %w", err)
	}

	// Run database synchronization (unless skipped)
	if !pm.config.SkipDBSync {
		if err := pm.runIronicDbsync(); err != nil {
			return fmt.Errorf("failed to synchronize database: %w", err)
		}
	} else {
		pm.logger.Info("Skipping database synchronization (SkipDBSync=true)")
	}

	// Start all-in-one Ironic process
	if err := pm.startProcess("ironic", []string{
		"/usr/bin/ironic",
		"--config-file", pm.config.ConfigPath,
	}); err != nil {
		return fmt.Errorf("failed to start ironic: %w", err)
	}

	// Start health check routine
	pm.wg.Add(1)
	go pm.healthCheckLoop()

	return nil
}

// prepareSocketDir ensures the socket directory is ready.
func (pm *ProcessManager) prepareSocketDir() error {
	socketDir := filepath.Dir(pm.config.SocketPath)
	if err := os.MkdirAll(socketDir, 0o755); err != nil {
		return err
	}

	// Remove existing socket if it exists
	if err := os.RemoveAll(pm.config.SocketPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// generateIronicConfig creates the Ironic configuration file.
func (pm *ProcessManager) generateIronicConfig() error {
	configDir := filepath.Dir(pm.config.ConfigPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	cfg := pm.config
	if cfg == nil {
		cfg = &Config{}
	}

	config, err := cfg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal Ironic config: %w", err)
	}

	return os.WriteFile(pm.config.ConfigPath, config, 0o644)
}

// startProcess starts and supervises a single process.
func (pm *ProcessManager) startProcess(name string, args []string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	process := &Process{
		name:    name,
		restart: make(chan struct{}, 1),
	}

	pm.processes[name] = process
	pm.wg.Add(1)
	go pm.superviseProcess(process, args)

	return nil
}

// superviseProcess handles the lifecycle of a single process.
func (pm *ProcessManager) superviseProcess(proc *Process, args []string) {
	defer pm.wg.Done()

	for {
		select {
		case <-pm.ctx.Done():
			pm.logger.Info("Shutting down process", "process name", proc.name)
			pm.stopProcess(proc)
			return
		case <-proc.restart:
			pm.logger.Info("Restarting process", "process name", proc.name)
			pm.stopProcess(proc)
			time.Sleep(2 * time.Second) // Brief delay before restart
		default:
		}

		pm.logger.Debug("Starting process", "process name", proc.name)
		proc.mu.Lock()
		proc.cmd = exec.CommandContext(pm.ctx, args[0], args[1:]...)

		// Set up process group for proper signal handling
		proc.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		// Capture stdout and stderr
		proc.cmd.Stdout = &logWriter{logger: pm.logger, prefix: fmt.Sprintf("[%s] ", proc.name)}
		proc.cmd.Stderr = &logWriter{
			logger: pm.logger,
			prefix: fmt.Sprintf("[%s:ERROR] ", proc.name),
		}

		if err := proc.cmd.Start(); err != nil {
			pm.logger.Info("Failed to start process", "process name", proc.name, "error", err)
			proc.mu.Unlock()
			time.Sleep(5 * time.Second)
			continue
		}

		proc.healthy = true
		proc.mu.Unlock()

		// Wait for process to exit
		err := proc.cmd.Wait()

		proc.mu.Lock()
		proc.healthy = false
		proc.mu.Unlock()

		if err != nil && pm.ctx.Err() == nil {
			pm.logger.Info("Process exited with error", "process name", proc.name, "error", err)
			time.Sleep(5 * time.Second) // Backoff before restart
		} else if pm.ctx.Err() == nil {
			pm.logger.Info("Process exited normally", "process name", proc.name)
		}
	}
}

// stopProcess gracefully stops a process.
func (pm *ProcessManager) stopProcess(proc *Process) {
	proc.mu.Lock()
	defer proc.mu.Unlock()

	if proc.cmd == nil || proc.cmd.Process == nil {
		return
	}

	pm.logger.Debug("Stopping process", "process name", proc.name, "PID", proc.cmd.Process.Pid)

	// Send SIGTERM to process group
	if err := syscall.Kill(-proc.cmd.Process.Pid, syscall.SIGTERM); err != nil {
		pm.logger.Info("Failed to send SIGTERM", "process name", proc.name, "error", err)
	}

	// Wait for graceful shutdown with timeout
	done := make(chan error, 1)
	go func() {
		done <- proc.cmd.Wait()
	}()

	select {
	case <-done:
		pm.logger.Info("Process stopped gracefully", "process name", proc.name)
	case <-time.After(10 * time.Second):
		pm.logger.Info("Process didn't stop gracefully, sending SIGKILL", "process name", proc.name)
		if err := syscall.Kill(-proc.cmd.Process.Pid, syscall.SIGKILL); err != nil {
			pm.logger.Info("Failed to send SIGKILL", "process name", proc.name, "error", err)
		}
	}
}

// healthCheckLoop performs periodic health checks.
func (pm *ProcessManager) healthCheckLoop() {
	defer pm.wg.Done()

	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.performHealthChecks()
		}
	}
}

// performHealthChecks checks the health of all processes.
func (pm *ProcessManager) performHealthChecks() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for name, proc := range pm.processes {
		proc.mu.RLock()
		if proc.cmd == nil || proc.cmd.Process == nil {
			proc.mu.RUnlock()
			continue
		}

		// Check if process is still running
		if err := proc.cmd.Process.Signal(syscall.Signal(0)); err != nil {
			pm.logger.Info("Health check failed", "process name", name, "error", err)
			proc.mu.RUnlock()
			select {
			case proc.restart <- struct{}{}:
			default:
			}
			continue
		}
		proc.mu.RUnlock()

		// For ironic (all-in-one), also check socket connectivity
		if name == "ironic" {
			if err := pm.checkSocketHealth(); err != nil {
				pm.logger.Info("Socket health check failed", "process name", name, "error", err)
			}
		}
	}
}

// checkSocketHealth verifies the Unix socket is responsive.
func (pm *ProcessManager) checkSocketHealth() error {
	conn, err := net.DialTimeout("unix", pm.config.SocketPath, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Send a simple HTTP request to check responsiveness
	req := "GET /v1/ HTTP/1.1\r\nHost: localhost\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		return err
	}

	// Read response (we don't need to parse it fully)
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Read(buffer)
	return err
}

// Shutdown gracefully shuts down all processes.
func (pm *ProcessManager) Shutdown() {
	pm.logger.Info("Initiating graceful shutdown")
	pm.cancel()

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		pm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		pm.logger.Info("All processes stopped gracefully")
	case <-time.After(shutdownTimeout):
		pm.logger.Info(
			"Shutdown timeout reached, some processes may not have stopped gracefully",
		)
	}
}

// createDefaultPolicy creates a default policy.yaml file for Ironic with the specified permissions.
func (pm *ProcessManager) createDefaultPolicy() error {
	// Default policy configuration as a map
	policyMap := map[string]string{
		"show_password":                           "",
		"show_instance_secrets":                   "",
		"baremetal:node:get:last_error":           "",
		"baremetal:node:get:reservation":          "",
		"baremetal:node:get:driver_internal_info": "",
		"baremetal:node:get:driver_info":          "",
		"baremetal:node:update:driver_info":       "",
		"baremetal:allocation:get":                "",
		"baremetal:allocation:list_all":           "",
		"baremetal:node:update:properties":        "",
		"baremetal:node:update:chassis_uuid":      "",
		"baremetal:node:update:instance_uuid":     "",
		"baremetal:node:update:lessee":            "",
		"baremetal:node:update:owner":             "",
		"baremetal:node:update:driver_interfaces": "",
		"baremetal:node:update:network_data":      "",
		"baremetal:node:update:conductor_group":   "",
		"baremetal:node:update:name":              "",
		"baremetal:node:update:retired":           "",
	}

	// Marshal the map to YAML
	yamlData, err := yaml.Marshal(policyMap)
	if err != nil {
		return fmt.Errorf("failed to marshal policy to YAML: %w", err)
	}

	// Determine the policy file path (in the same directory as the Ironic config)
	configDir := filepath.Dir(pm.config.ConfigPath)
	policyPath := filepath.Join(configDir, "policy.yaml")

	// Ensure the directory exists
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write the policy file
	if err := os.WriteFile(policyPath, yamlData, 0o644); err != nil {
		return fmt.Errorf("failed to write policy file: %w", err)
	}

	pm.logger.Debug("Created default policy file", "path", policyPath)
	return nil
}
