package ironic

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const (
	ironicSocketPath    = "/tmp/ironic-api.sock"
	ironicConfigPath    = "/etc/ironic/ironic.conf"
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
	return &ProcessManager{
		processes: make(map[string]*Process),
		ctx:       ctx,
		cancel:    cancel,
		logger:    logger,
		config:    config,
	}
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

	// Start all-in-one Ironic process
	if err := pm.startProcess("ironic", []string{
		"/usr/bin/ironic",
		"--config-file", ironicConfigPath,
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
	socketDir := filepath.Dir(ironicSocketPath)
	if err := os.MkdirAll(socketDir, 0o755); err != nil {
		return err
	}

	// Remove existing socket if it exists
	if err := os.RemoveAll(ironicSocketPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// generateIronicConfig creates the Ironic configuration file.
func (pm *ProcessManager) generateIronicConfig() error {
	configDir := filepath.Dir(ironicConfigPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	cfg := pm.config
	if cfg == nil {
		cfg = &Config{}
	}

	// Set default values for all-in-one operation with Unix sockets
	if cfg.Default.LogFile == "" {
		cfg.Default.LogFile = "/var/log/ironic/ironic.log"
	}

	if cfg.Default.RPCTransport == "" {
		cfg.Default.RPCTransport = "json-rpc"
	}

	if cfg.API.UnixSocket == "" {
		cfg.API.UnixSocket = ironicSocketPath
	}

	if cfg.API.UnixSocketMode == "" {
		cfg.API.UnixSocketMode = "0666"
	}

	if cfg.JSONRPC.UnixSocket == "" {
		cfg.JSONRPC.UnixSocket = "/tmp/ironic-rpc.sock"
	}

	if cfg.JSONRPC.UnixSocketMode == "" {
		cfg.JSONRPC.UnixSocketMode = "0666"
	}

	if cfg.JSONRPC.AuthStrategy == "" {
		cfg.JSONRPC.AuthStrategy = "noauth"
	}

	if cfg.OsloMessagingNotifications.Driver == "" {
		cfg.OsloMessagingNotifications.Driver = "noop"
	}

	if cfg.Conductor.APIURL == "" {
		cfg.Conductor.APIURL = fmt.Sprintf("http+unix://%s", ironicSocketPath)
	}

	// Set up database for all-in-one operation
	if cfg.Database.Connection == "" {
		cfg.Database.Connection = "sqlite:////var/lib/ironic/ironic.db"
	}

	// Configure for standalone operation
	if cfg.Default.AuthStrategy == "" {
		cfg.Default.AuthStrategy = "noauth"
	}

	config, err := cfg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal Ironic config: %w", err)
	}

	return os.WriteFile(ironicConfigPath, config, 0o644)
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
	conn, err := net.DialTimeout("unix", ironicSocketPath, 5*time.Second)
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
