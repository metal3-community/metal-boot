# Unifi SSH Power Management Package

This package provides SSH-based power management for Unifi switches, implementing the `backend.BackendPower` interface for the Metal Boot system.

## Features

- SSH-based communication with Unifi switches
- Support for SSH key authentication following Go's standard library patterns
- Implementation of the `backend.BackendPower` interface
- MAC address to port mapping
- Support for multiple MAC address formats (colons, dashes, no separators)
- Direct port control and MAC-based device control
- Context-aware operations with timeout support

## Usage

### Basic Client Setup

```go
package main

import (
    "context"
    "io/ioutil"
    "log"
    "time"

    "golang.org/x/crypto/ssh"
    "github.com/metal3-community/metal-boot/internal/backend/power/unifi"
    "github.com/metal3-community/metal-boot/internal/dhcp/data"
)

func main() {
    // Read SSH private key from file
    privateKey, err := ioutil.ReadFile("/path/to/your/ssh/private/key")
    if err != nil {
        log.Fatal(err)
    }

    // Create client configuration
    config := &unifi.Config{
        Host:       "192.168.1.10", // IP of your Unifi switch
        Port:       22,             // SSH port (default: 22)
        Username:   "admin",        // SSH username
        PrivateKey: privateKey,     // SSH private key
        Timeout:    30 * time.Second,
        // For production, implement proper host key verification
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }

    // Create the client
    client, err := unifi.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Get power state of port 2
    state, err := client.GetPortPowerState(ctx, 2)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Port 2 power state: %s", state.String())

    // Set power state to on
    err = client.SetPortPower(ctx, 2, data.PowerOn)
    if err != nil {
        log.Fatal(err)
    }

    // Power cycle the port
    err = client.RestartPortPower(ctx, 2)
    if err != nil {
        log.Fatal(err)
    }
}
```

### PowerManager for Backend Integration

```go
package main

import (
    "context"
    "io/ioutil"
    "net"

    "golang.org/x/crypto/ssh"
    "github.com/metal3-community/metal-boot/internal/backend/power/unifi"
    "github.com/metal3-community/metal-boot/internal/dhcp/data"
)

func main() {
    // Setup client as above...
    privateKey, _ := ioutil.ReadFile("/path/to/ssh/key")
    config := &unifi.Config{
        Host:            "192.168.1.10",
        Username:        "admin",
        PrivateKey:      privateKey,
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }
    client, _ := unifi.NewClient(config)

    // Create port mapping (MAC address -> Port ID)
    portMapping := unifi.PortMapping{
        "aa:bb:cc:dd:ee:ff": 1,  // Device 1 on port 1
        "11:22:33:44:55:66": 2,  // Device 2 on port 2
        "aa-bb-cc-dd-ee-00": 3,  // Supports dash format
        "aabbccddeeff":       4,  // Supports no separator format
    }

    // Create PowerManager that implements backend.BackendPower interface
    powerManager := unifi.NewPowerManager(client, portMapping)

    // Use with MAC addresses
    mac := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
    
    // Get power state by MAC address
    state, err := powerManager.GetPower(context.Background(), mac)
    if err != nil {
        log.Fatal(err)
    }

    // Set power state by MAC address
    err = powerManager.SetPower(context.Background(), mac, data.PowerOn)
    if err != nil {
        log.Fatal(err)
    }

    // Power cycle by MAC address
    err = powerManager.PowerCycle(context.Background(), mac)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Supported Commands

The package uses the following Unifi switch commands via SSH:

- `swctrl poe show id PORT_ID` - Get current POE status
- `swctrl poe set auto id PORT_ID` - Enable POE (power on)
- `swctrl poe set off id PORT_ID` - Disable POE (power off)
- `swctrl poe restart id PORT_ID` - Restart POE (power cycle)

## Power State Mapping

| Data.PowerState | POE Mode | Description |
|----------------|----------|-------------|
| `PowerOn`      | `auto`   | POE enabled and providing power |
| `PowerOff`     | `off`    | POE disabled |
| `PoweringOn`   | `auto`   | POE enabled but not yet providing power |
| `PoweringOff`  | `off`    | POE being disabled |

## Configuration

### SSH Key Authentication

The package requires SSH key authentication. Generate an SSH key pair and add the public key to the Unifi switch's authorized keys:

```bash
# Generate SSH key pair
ssh-keygen -t rsa -b 4096 -f ~/.ssh/unifi_key

# Copy public key to switch (method varies by switch model)
ssh-copy-id -i ~/.ssh/unifi_key.pub admin@192.168.1.10
```

### MAC Address Formats

The port mapping supports multiple MAC address formats:

- Colon-separated: `aa:bb:cc:dd:ee:ff`
- Dash-separated: `aa-bb-cc-dd-ee-ff`
- No separators: `aabbccddeeff`

All formats are case-insensitive.

### Host Key Verification

For production use, implement proper host key verification:

```go
import "golang.org/x/crypto/ssh/knownhosts"

hostKeyCallback, err := knownhosts.New("/path/to/known_hosts")
if err != nil {
    log.Fatal(err)
}

config := &unifi.Config{
    // ... other settings
    HostKeyCallback: hostKeyCallback,
}
```

## Error Handling

The package returns descriptive errors for common scenarios:

- SSH connection failures
- Authentication failures
- Command execution timeouts
- Unknown MAC addresses (port mapping not found)
- Invalid power states

## Integration with Metal Boot

The `PowerManager` type implements the `backend.BackendPower` interface, making it compatible with the Metal Boot system's backend architecture:

```go
type BackendPower interface {
    GetPower(context.Context, net.HardwareAddr) (*data.PowerState, error)
    SetPower(ctx context.Context, mac net.HardwareAddr, state data.PowerState) error
    PowerCycle(ctx context.Context, mac net.HardwareAddr) error
}
```

### Example Integration

```go
package main

import (
    "context"
    "io/ioutil"
    "net"

    "golang.org/x/crypto/ssh"
    "github.com/metal3-community/metal-boot/internal/backend"
    "github.com/metal3-community/metal-boot/internal/backend/power/unifi"
    "github.com/metal3-community/metal-boot/internal/dhcp/data"
)

// Create a backend that implements BackendPower
func createUnifiBackend() (backend.BackendPower, error) {
    privateKey, err := ioutil.ReadFile("/etc/pibmc/ssh/id_rsa")
    if err != nil {
        return nil, err
    }

    config := &unifi.Config{
        Host:            "192.168.1.10",
        Username:        "pibmc-service",
        PrivateKey:      privateKey,
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }

    client, err := unifi.NewClient(config)
    if err != nil {
        return nil, err
    }

    portMapping := unifi.PortMapping{
        "b8:27:eb:12:34:56": 1, // Pi Node 1
        "b8:27:eb:78:9a:bc": 2, // Pi Node 2
        "b8:27:eb:de:f0:12": 3, // Pi Node 3
        "b8:27:eb:34:56:78": 4, // Pi Node 4
    }

    return unifi.NewPowerManager(client, portMapping), nil
}

// Use in DHCP handlers or Redfish API
func handleDHCPDecline(powerBackend backend.BackendPower, mac net.HardwareAddr) error {
    ctx := context.Background()
    return powerBackend.PowerCycle(ctx, mac)
}
```

This allows the Unifi power manager to be used in DHCP handlers, Redfish API endpoints, and other Metal Boot components that require power management capabilities.

## Security Considerations

1. **SSH Key Management**: Store SSH private keys securely with appropriate file permissions (600)
2. **Host Key Verification**: Always verify host keys in production environments
3. **Network Security**: Ensure SSH traffic is protected (use VPN or secure networks)
4. **Access Control**: Use dedicated service accounts with minimal privileges
5. **Logging**: Monitor SSH access and power management operations

## Testing

Run the unit tests:

```bash
go test ./internal/backend/power/unifi
```

The tests cover:
- Client configuration validation
- Port mapping resolution
- MAC address format handling
- Error scenarios

## Limitations

- Requires SSH access to the Unifi switch
- Switch must support the `swctrl` command interface
- Port mapping must be manually configured
- No automatic discovery of device-to-port relationships
