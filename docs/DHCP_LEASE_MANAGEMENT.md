# DHCP Lease Management

Metal Boot now supports DNSMasq-compatible DHCP lease and configuration management, providing seamless integration with existing DNSMasq-based infrastructure.

## Features

### Lease File Management
- **DNSMasq-compatible lease format**: Stores DHCP leases in the standard DNSMasq format
- **Automatic lease tracking**: Records all DHCP ACK responses 
- **Lease expiration**: Automatically cleans up expired leases
- **Persistent storage**: Leases survive service restarts

### Configuration File Management  
- **DHCP option configuration**: Supports DNSMasq-style DHCP option configuration
- **Tag-based options**: Uses MAC address-based tags for client-specific options
- **Netboot support**: Automatically configures iPXE/PXE boot options
- **Dynamic updates**: Updates configuration when netboot clients are detected

## Configuration

Add the following to your Metal Boot configuration file:

```yaml
dhcp:
  enabled: true
  proxy_enabled: false  # Must be false to use reservation handler
  
  # Lease management files
  lease_file: "/var/lib/dhcp/dhcp.leases"     # DNSMasq lease file
  config_file: "/etc/dhcp/dhcp.conf"          # DHCP options configuration
  
  # ... other DHCP settings
```

## File Formats

### Lease File Format

The lease file follows the standard DNSMasq format:

```
# DHCP leases file - DNSMasq compatible format
# <expiry-time> <mac-address> <ip-address> <hostname> <client-id>
1692123456 aa:bb:cc:dd:ee:ff 192.168.1.100 hostname01 *
1692127056 bb:cc:dd:ee:ff:aa 192.168.1.101 hostname02 *
```

Fields:
- **expiry-time**: Unix timestamp when lease expires
- **mac-address**: Client MAC address in colon-separated format
- **ip-address**: Assigned IP address
- **hostname**: Client hostname or "*" if not provided
- **client-id**: DHCP client identifier or "*" if not provided

### Configuration File Format

The configuration file supports DNSMasq-style DHCP option configuration:

```
# DHCP options configuration - DNSMasq compatible format
# Format: tag:<tag>,tag:<conditional>,<option-code>,<value>

# Example for MAC aa:bb:cc:dd:ee:ff
tag:aa:bb:cc:dd:ee:ff,tag:!ipxe,67,ipxe.efi
tag:aa:bb:cc:dd:ee:ff,tag:ipxe,67,http://ironic.appkins.io/boot.ipxe
tag:aa:bb:cc:dd:ee:ff,66,10.1.1.1
tag:aa:bb:cc:dd:ee:ff,150,10.1.1.1
tag:aa:bb:cc:dd:ee:ff,255,10.1.1.1
tag:aa:bb:cc:dd:ee:ff,tag:!ipxe6,59,tftp://10.1.1.1/ipxe.efi
tag:aa:bb:cc:dd:ee:ff,tag:ipxe6,59,http://ironic.appkins.io/boot.ipxe
```

Format explanation:
- **tag:<tag>**: Primary tag (typically MAC address)
- **tag:<conditional>**: Optional conditional tag (e.g., `!ipxe`, `ipxe6`)
- **option-code**: DHCP option number
- **value**: Option value

## Common DHCP Options

| Option | Description | Example Value |
|--------|-------------|---------------|
| 66 | TFTP Server Name | `10.1.1.1` |
| 67 | Boot File Name | `ipxe.efi` or `http://server/boot.ipxe` |
| 150 | TFTP Server Address (Cisco) | `10.1.1.1` |
| 59 | IPv6 Boot File URL | `tftp://10.1.1.1/ipxe.efi` |

## Operation

### Lease Lifecycle

1. **DHCP DISCOVER**: Client broadcasts discovery request
2. **DHCP OFFER**: Metal Boot responds with IP offer (no lease recorded)
3. **DHCP REQUEST**: Client requests specific IP
4. **DHCP ACK**: Metal Boot confirms assignment and **records lease**
5. **Lease Expiry**: Automatic cleanup of expired leases every 5 minutes

### Configuration Updates

When a netboot-capable client receives a DHCP response:

1. Metal Boot detects the client supports PXE/iPXE
2. Automatically generates DHCP option configuration
3. Updates the configuration file with appropriate boot options
4. Configuration persists across service restarts

### File Management

- **Atomic updates**: Uses temporary files and atomic renames
- **Directory creation**: Automatically creates parent directories
- **Error handling**: Logs errors but continues service operation
- **Async writes**: Lease/config saves don't block DHCP responses

## Integration with DNSMasq

The lease and configuration files are fully compatible with DNSMasq:

1. **Shared lease file**: DNSMasq can read Metal Boot's lease file
2. **Configuration import**: DNSMasq can include Metal Boot's config file
3. **Seamless migration**: Easy transition between Metal Boot and DNSMasq

Example DNSMasq integration:

```bash
# Include Metal Boot DHCP options
conf-file=/etc/dhcp/dhcp.conf

# Share lease file  
dhcp-leasefile=/var/lib/dhcp/dhcp.leases
```

## Monitoring

Metal Boot provides structured logging for lease management:

```json
{"level":"INFO","msg":"DHCP lease management enabled","lease_file":"/var/lib/dhcp/dhcp.leases","config_file":"/etc/dhcp/dhcp.conf"}
{"level":"INFO","msg":"starting DHCP lease cleanup routine"}
{"level":"DEBUG","msg":"cleaned up expired DHCP leases"}
```

## Security Considerations

- **File permissions**: Ensure appropriate permissions on lease/config directories
- **Disk space**: Monitor disk usage for lease file growth
- **Log rotation**: Configure log rotation for audit trails

## Troubleshooting

### Common Issues

1. **Permission denied**: Ensure Metal Boot has write access to lease/config directories
2. **File conflicts**: Avoid concurrent DNSMasq and Metal Boot writing to same files
3. **Disk full**: Monitor available disk space

### Debug Commands

```bash
# Check lease file contents
cat /var/lib/dhcp/dhcp.leases

# Check configuration file
cat /etc/dhcp/dhcp.conf

# Monitor Metal Boot logs
journalctl -u pibmc -f
```
