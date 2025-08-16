# DNSMasq Backend Automatic Lease Assignment

This document describes the automatic lease assignment feature in the DNSMasq backend.

## Overview

The DNSMasq backend now supports automatic IP address assignment for unknown MAC addresses. When enabled, the backend will automatically assign IP addresses from a configured pool to devices that are not already in the lease database.

## Configuration

Add the following settings to your `config.yaml` under the `dnsmasq` section:

```yaml
dnsmasq:
  enabled: true
  root_directory: "/shared/dnsmasq"
  tftp_server: "192.168.1.1"
  http_server: "192.168.1.1"
  
  # Automatic lease assignment settings
  auto_assign_enabled: true              # Enable automatic assignment
  ip_pool_start: "192.168.1.100"        # Start of IP pool for auto-assignment
  ip_pool_end: "192.168.1.200"          # End of IP pool for auto-assignment
  default_lease_time: 604800             # Default lease time in seconds (1 week)
  default_gateway: "192.168.1.1"        # Default gateway for auto-assigned leases
  default_subnet: "255.255.255.0"       # Default subnet mask
  default_dns:                           # Default DNS servers
    - "8.8.8.8"
    - "8.8.4.4"
  default_domain: "local"                # Default domain name
```

## How It Works

### Assignment Algorithm

1. **MAC-based Deterministic Assignment**: The system uses MD5 hash of the MAC address to calculate a deterministic offset within the IP pool. This ensures that the same MAC address always gets the same IP address.

2. **Collision Handling**: If the calculated IP is already assigned to a different MAC address, the system searches sequentially through the pool until it finds an available IP.

3. **Pool Exhaustion**: If no IPs are available in the configured range, the assignment fails with an appropriate error.

### Lease Lifecycle

1. **First Request**: When an unknown MAC address requests an IP, the backend automatically assigns one from the pool and creates a lease entry.

2. **Subsequent Requests**: The same MAC address will always receive the same IP address, ensuring consistency.

3. **Persistence**: Auto-assigned leases are saved to the DNSMasq lease file and persist across backend restarts.

4. **File Watching**: Changes to the lease file are automatically monitored and the in-memory cache is updated in real-time.

## Features

### Deterministic Assignment
- **Consistent IPs**: Same MAC address always gets the same IP
- **Hash-based**: Uses MD5 hash of MAC address for deterministic distribution
- **Collision-free**: Automatically handles IP conflicts

### Integration
- **DNSMasq Compatible**: Lease file format is fully compatible with DNSMasq
- **File Watching**: Real-time monitoring of lease file changes
- **Thread-safe**: Concurrent access is properly synchronized

### Configuration Flexibility
- **Optional**: Auto-assignment can be disabled (default)
- **Configurable Pool**: Define custom IP ranges for auto-assignment
- **Custom Defaults**: Set default network parameters for auto-assigned leases

## Usage Examples

### Basic Auto-Assignment

```go
import "github.com/metal3-community/metal-boot/internal/backend/dnsmasq"

config := dnsmasq.Config{
    RootDir:           "/var/lib/dnsmasq",
    TFTPServer:        "192.168.1.1",
    HTTPServer:        "192.168.1.1",
    AutoAssignEnabled: true,
    IPPoolStart:       "192.168.1.100",
    IPPoolEnd:         "192.168.1.200",
    DefaultLeaseTime:  3600, // 1 hour
}

backend, err := dnsmasq.NewBackend(logger, config)
if err != nil {
    log.Fatal(err)
}

// Unknown MAC will be automatically assigned an IP
dhcp, netboot, err := backend.GetByMac(ctx, unknownMAC)
```

### Integration with Main Config

```go
// Convert main config to backend config
backendConfig := dnsmasq.Config{
    RootDir:           config.Dnsmasq.RootDirectory,
    TFTPServer:        config.Dnsmasq.TFTPServer,
    HTTPServer:        config.Dnsmasq.HTTPServer,
    AutoAssignEnabled: config.Dnsmasq.AutoAssignEnabled,
    IPPoolStart:       config.Dnsmasq.IPPoolStart,
    IPPoolEnd:         config.Dnsmasq.IPPoolEnd,
    DefaultLeaseTime:  config.Dnsmasq.DefaultLeaseTime,
    // ... other fields
}
```

## Security Considerations

- **Network Isolation**: Only enable auto-assignment in trusted networks
- **IP Pool Separation**: Use separate IP ranges for auto-assigned devices
- **Monitoring**: Monitor logs for unexpected auto-assignments
- **Access Control**: Consider MAC address filtering for production environments

## Logging

The backend provides structured logging for auto-assignment events:

```json
{"level":"INFO","msg":"MAC address not found, auto-assigning lease","mac":"aa:bb:cc:dd:ee:ff"}
{"level":"ERROR","msg":"failed to save auto-assigned lease","mac":"aa:bb:cc:dd:ee:ff","ip":"192.168.1.100","error":"..."}
```

## Troubleshooting

### Common Issues

1. **Pool Exhaustion**: Increase IP pool size or clean up expired leases
2. **Invalid Range**: Ensure `ip_pool_start` is less than `ip_pool_end`
3. **Permission Errors**: Verify write access to the lease file directory
4. **IP Conflicts**: Check for overlapping static assignments

### Debug Commands

```bash
# Check current leases
cat /var/lib/dnsmasq/dnsmasq.leases

# Monitor auto-assignment logs
journalctl -u pibmc -f | grep "auto-assigning"

# Test IP pool configuration
curl -X GET http://localhost:8080/api/backend/keys
```

## Integration with DHCP Fallback

The auto-assignment feature works alongside the existing DHCP fallback functionality:

- **Backend Level**: Auto-assignment handles MAC addresses at the backend storage level
- **DHCP Level**: Fallback handles unknown devices at the DHCP protocol level
- **Complementary**: Both can be enabled simultaneously for comprehensive coverage

For complete DHCP coverage, consider enabling both features with overlapping or separate IP ranges.
