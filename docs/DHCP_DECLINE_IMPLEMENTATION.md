# DHCP DECLINE Handling Implementation

## Overview

This implementation provides comprehensive DHCP DECLINE handling for the PIBMC DHCP server, including ARP-based IP conflict detection and proper lease management with cooldown periods.

## Features

1. **ARP Conflict Detection**: Probes IP addresses using ARP to detect conflicts before assignment
2. **DHCP DECLINE Processing**: Properly handles DHCP DECLINE messages from clients
3. **Lease Tracking**: Tracks declined IPs with configurable cooldown periods (default: 5 minutes)
4. **DNSMasq Compatibility**: Integrates with existing DNSMasq lease file format
5. **DHCP NAK Responses**: Sends NAK responses when IP conflicts are detected

## Architecture

### Components

1. **ARP Conflict Detector** (`internal/dhcp/arp/arp.go`)
   - Performs ARP probes to detect IP address conflicts
   - Configurable probe count and intervals
   - Non-blocking operation with timeouts

2. **Enhanced DHCP Handler** (`internal/dhcp/handler/reservation/handler.go`)
   - Integrates ARP detection into DISCOVER/REQUEST processing
   - Processes DHCP DECLINE messages
   - Sends appropriate NAK responses for conflicts

3. **Lease Manager Integration** (`internal/dhcp/handler/reservation/adapter.go`)
   - Bridges string-based interface with DNSMasq lease manager
   - Provides decline tracking and cleanup

4. **Enhanced Lease Storage** (`internal/backend/dnsmasq/lease.go`)
   - Extended Lease struct with decline tracking fields
   - IP cooldown management
   - Automatic cleanup of expired declined IPs

## Usage Example

```go
package main

import (
    "context"
    "net/netip"
    
    "github.com/metal3-community/metal-boot/internal/dhcp/handler/reservation"
    "github.com/metal3-community/metal-boot/internal/dhcp/arp"
    "github.com/metal3-community/metal-boot/internal/backend/dnsmasq"
    "github.com/go-logr/logr"
)

func main() {
    // Create DNSMasq lease manager
    leaseManager := dnsmasq.NewLeaseManager("/path/to/dnsmasq.leases", logr.Discard())
    
    // Create ARP conflict detector
    arpDetector := arp.NewConflictDetector("eth0", logr.Discard())
    
    // Create DHCP handler with DECLINE support
    handler := &reservation.Handler{
        Backend:       myBackend,           // Your existing backend
        LeaseBackend:  reservation.NewLeaseManagerAdapter(leaseManager),
        ARPDetector:   arpDetector,
        IPAddr:        netip.MustParseAddr("192.168.1.1"),
        InterfaceName: "eth0",              // Enable ARP detection
        Log:           logr.Discard(),
    }
    
    // The handler will now:
    // 1. Check for ARP conflicts before offering IPs
    // 2. Process DHCP DECLINE messages
    // 3. Track declined IPs with cooldown periods
    // 4. Send NAK responses for conflicted IPs
}
```

## Configuration

### ARP Detection Configuration

```go
arpDetector := arp.NewConflictDetector("eth0", logger)
arpDetector.ProbeCount = 3                        // Number of ARP probes (default: 3)
arpDetector.ProbeInterval = 100 * time.Millisecond // Interval between probes (default: 100ms)
```

### Lease Management Configuration

The lease manager uses a 5-minute cooldown period by default. Declined IPs are automatically cleaned up after this period.

## Message Flow

### DHCP DISCOVER with ARP Check
1. Client sends DHCP DISCOVER
2. Server checks if assigned IP is declined (from lease manager)
3. Server performs ARP probe to detect conflicts
4. If conflict detected: IP is marked as declined, no OFFER sent
5. If no conflict: Normal DHCP OFFER sent

### DHCP REQUEST with Conflict Detection
1. Client sends DHCP REQUEST
2. Server checks for IP conflicts (declined status + ARP)
3. If conflict detected: DHCP NAK sent with error message
4. If no conflict: Normal DHCP ACK sent

### DHCP DECLINE Processing
1. Client sends DHCP DECLINE with requested IP
2. Server extracts declined IP from option 50
3. Server marks IP as declined in lease manager
4. Server optionally verifies conflict with ARP probe
5. No response sent (per RFC 2131)

## Testing

Run the DHCP DECLINE handling tests:

```bash
cd /path/to/pibmc
go test ./internal/dhcp/arp          # ARP conflict detection tests
go test -run TestHandler_handleDecline ./internal/dhcp/handler/reservation  # DECLINE handling tests
go test -run TestHandler_hasIPConflict ./internal/dhcp/handler/reservation  # Conflict detection tests
```

## Monitoring and Logging

The implementation provides detailed logging for:
- ARP probe results
- DHCP DECLINE processing
- IP conflict detection
- Lease decline tracking

Example log entries:
```
INFO ARP: IP address is in use ip=192.168.1.100 mac=aa:bb:cc:dd:ee:ff probe=1
INFO processing DHCP DECLINE declined_ip=192.168.1.100 client_mac=aa:bb:cc:dd:ee:ff
INFO marked IP as declined ip=192.168.1.100
INFO ARP conflict detected for IP ip=192.168.1.100
```

## Integration with Existing Systems

This implementation is designed to integrate seamlessly with existing PIBMC deployments:

1. **Backward Compatibility**: Existing configurations continue to work
2. **Optional Features**: ARP detection can be disabled by not setting `InterfaceName`
3. **DNSMasq Compatibility**: Lease file format remains compatible
4. **Graceful Degradation**: System works even if ARP detection fails

## Performance Considerations

- ARP probes add ~300ms delay to DHCP processing (3 probes × 100ms interval)
- Lease file operations are cached and only written when necessary
- Failed ARP operations don't block DHCP responses
- Declined IP cleanup happens in background

## Troubleshooting

### Common Issues

1. **ARP Detection Not Working**
   - Verify interface name is correct
   - Check interface is up and has valid IP
   - Ensure sufficient permissions for raw socket operations

2. **High DHCP DECLINE Rates**
   - Check for actual IP conflicts on network
   - Verify DHCP pool doesn't overlap with static assignments
   - Monitor ARP conflict detection logs

3. **Performance Issues**
   - Reduce ARP probe count for faster responses
   - Increase probe interval if network is congested
   - Consider disabling ARP detection on stable networks

This implementation successfully addresses the original issues with Raspberry Pi DHCP DECLINE messages while providing a robust foundation for IP conflict management.
