# Metal Boot Copilot Instructions

## Project Overview

Metal Boot is a comprehensive BMC (Baseboard Management Controller) solution for Raspberry Pi 4 devices. It provides enterprise-grade server management capabilities including DHCP proxy, TFTP, HTTP services, UEFI firmware management, Redfish API, and power control via PoE switches.

## Core Architecture

### Service Orchestration Pattern
The main application (`cmd/metal-boot/main.go`) uses `errgroup.WithContext()` for concurrent service management:
- HTTP API server with multiple handlers
- DHCP server (proxy or reservation modes)
- TFTP server for firmware delivery
- Ironic supervisor (optional)

All services share graceful shutdown via context cancellation and 30-second timeouts.

### Configuration System
- **Viper-based**: Auto-binding environment variables (dot notation → SNAKE_CASE: `dhcp.enabled` → `DHCP_ENABLED`)
- **File watching**: Hot-reload configuration changes via `fsnotify`
- **Defaults cascade**: Code defaults → config file → environment variables
- **Structured types**: Each service has dedicated config structs (e.g., `DhcpConfig`, `TftpConfig`)

Example: Always use `cfg.Dhcp.Enabled` pattern, never hardcoded values.

### Backend Abstraction
Three backend interfaces define data operations:
- `BackendReader`: Device discovery (`GetByMac`, `GetByIP`, `GetKeys`)
- `BackendPower`: Power control (`SetPower`, `PowerCycle`) 
- `BackendSyncer`: Data synchronization (`Sync`)

Implementations:
- **dnsmasq**: File-based backend with lease management
- **unifi**: PoE switch integration for power control

## Key Patterns

### API Handler Registration
```go
// Pattern: All API handlers follow this registration pattern
apiServer.AddHandler("/path/", handler.New(logger, config, backends...))
```

Handlers:
- `/healthcheck` - System health
- `/metrics` - Prometheus metrics  
- `/redfish/v1/` - BMC management API
- `/v1/` - Ironic API proxy (when supervisor enabled)
- `/` - iPXE script serving
- `/iso/` - ISO image serving

### Structured Logging
Use `slog` with consistent fields:
```go
logger.Info("Starting service", "bind_addr", addr, "enabled", config.Enabled)
logger.Error("Operation failed", "error", err, "device", mac.String())
```

For Ironic log parsing, see `internal/ironic/log.go` - regex pattern extracts timestamp, level, module, and request ID.

### Error Handling
- Services return errors via errgroup for centralized handling
- Configuration errors are fatal (os.Exit(1))
- Runtime errors log and attempt graceful degradation
- Network timeouts use context.WithTimeout

## Development Workflows

### Building and Testing
```bash
# Standard build (no Makefile - use go directly)
go build ./cmd/metal-boot/

# Run all tests with verbose output
go test ./... -v

# Test specific package with coverage
go test ./internal/ironic/ -v -cover

# Generate code (iPXE binaries)
//go:generate go run ../../internal/ipxe/generate
```

### Configuration Development
1. Start with `configs/config.example.yaml`
2. Test with environment overrides: `DHCP_ENABLED=true ./metal-boot`
3. Watch logs for viper configuration changes (hot-reload)

### Adding New Backends
1. Implement `backend.BackendReader` interface
2. Add to backend factory in main.go
3. Add configuration struct to `internal/config/config.go`
4. Follow dnsmasq backend pattern for file operations

## Critical Integration Points

### DHCP Flow
- **Proxy mode**: Forward PXE requests, don't manage leases
- **Reservation mode**: Full DHCP server with DNSMasq-compatible lease files
- **Handler selection**: Based on `cfg.Dhcp.ProxyEnabled` in main.go

### UEFI Firmware Pipeline
1. Firmware stored in `data/` directory (per-device)
2. TFTP serves `RPI_EFI.fd` files with dynamic UEFI variables
3. Uses `metal3-community/uefi-firmware-manager` for EFI manipulation
4. Variables stored as JSON alongside firmware files

### Ironic Integration
- **Supervisor mode**: Runs Ironic as child process with structured log parsing
- **Proxy mode**: HTTP proxy to external Ironic API
- **Configuration**: Uses `data/defaults.conf` for comprehensive Ironic defaults
- **Unix sockets**: API and JSON-RPC communication via domain sockets

### Network Boot Chain
```
Pi DHCP → TFTP (firmware) → UEFI → iPXE → HTTP (scripts) → Boot
```

## Testing Strategy

### Unit Tests
- Mock backends using testify interfaces
- Configuration tests use TOML/YAML fixtures
- Log parsing tests use real Ironic output examples

### Integration Tests
- HTTP handlers test full request/response cycles
- DHCP tests use actual packet structures
- Backend tests verify file format compatibility

### Test Data Location
- `internal/*/testdata/` - Package-specific test fixtures
- `data/` - Runtime configuration and firmware files
- Use `//go:embed` for embedding test data when needed

## Project-Specific Conventions

### File Organization
- `api/` - HTTP handlers (one directory per endpoint)
- `internal/backend/` - Data storage backends
- `internal/dhcp/` - DHCP server and handlers  
- `internal/config/` - Configuration management
- `internal/ironic/` - Ironic integration and supervision
- `cmd/metal-boot/` - Main application entry point

### Naming Patterns
- Config structs: `ServiceConfig` (e.g., `DhcpConfig`, `TftpConfig`)
- Handlers: `handler.New()` constructor pattern
- Backends: Interface + concrete implementation per provider
- Services: Start with capital letter, follow Go conventions

### Error Patterns
- Wrap errors with context: `fmt.Errorf("failed to start DHCP server: %w", err)`
- Use structured logging for error details
- Fatal configuration errors, recoverable runtime errors

This codebase prioritizes production reliability with comprehensive configuration management, graceful error handling, and enterprise integration patterns.