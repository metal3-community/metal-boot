# DNSMasq Backend

This backend provides DNSMasq-compatible DHCP lease and configuration management for Metal Boot, with support for Ironic-style configuration files.

## Overview

The DNSMasq backend manages DHCP leases and netboot configurations using a file-based approach compatible with OpenStack Ironic's DNSMasq integration. It reads configuration from a directory structure containing host definitions and DHCP options.

## Directory Structure

The backend expects the following directory structure:

```
${config.dnsmasq.root_dir}/
├── dhcp.leases              # DHCP lease file (DNSMasq format)
├── hosts/                   # Host configuration directory
│   ├── ironic-${mac}.conf   # Per-MAC host configuration
│   └── ...
└── opts/                    # DHCP options directory
    ├── ironic-${node_id}.conf # Per-node DHCP options
    └── ...
```

## Host Configuration Files

Host configuration files are located in `${root_dir}/hosts/` and follow the naming pattern `ironic-${mac}.conf` where `${mac}` is the MAC address with colons.

### File Formats

**Enabled for netboot:**
```
${mac},set:${node_id},set:ironic
```
Example: `9c:6b:00:70:59:8a,set:92ef984e-995f-4aea-8088-9cde6a970a88,set:ironic`

**Disabled for netboot:**
```
${mac},ignore
```
Example: `d8:3a:dd:61:4d:15,ignore`

## DHCP Options Files

For hosts that are enabled for netboot (have `set:ironic`), the backend will read DHCP options from `${root_dir}/opts/ironic-${node_id}.conf`.

### Option File Format

DHCP options use the standard DNSMasq format:
```
tag:${tag},tag:${conditional},${option_code},${value}
```

Example:
```
# Boot filename for non-iPXE clients
tag:9c:6b:00:70:59:8a,tag:!ipxe,67,ipxe.efi

# Boot script URL for iPXE clients  
tag:9c:6b:00:70:59:8a,tag:ipxe,67,http://192.168.1.1/boot.ipxe

# TFTP server
tag:9c:6b:00:70:59:8a,66,192.168.1.1

# TFTP server (Cisco extension)
tag:9c:6b:00:70:59:8a,150,192.168.1.1
```

## Configuration

Add the following to your Metal Boot configuration:

```yaml
dnsmasq:
  enabled: true
  root_dir: "/var/lib/dnsmasq"
  tftp_server: "192.168.1.1"
  http_server: "192.168.1.1"
```

## Usage

The backend implements the standard Metal Boot Backend interfaces:

- `BackendReader`: Read DHCP and netboot configuration by MAC or IP
- `BackendWriter`: Write DHCP leases and netboot configuration
- `BackendPower`: Power management (not supported, returns success)
- `BackendSyncer`: Reload configuration from files

### Example Usage

```go
config := dnsmasq.Config{
    RootDir:    "/var/lib/dnsmasq",
    TFTPServer: "192.168.1.1", 
    HTTPServer: "192.168.1.1",
}

backend, err := dnsmasq.NewBackend(logger, config)
if err != nil {
    log.Fatal(err)
}

// Get configuration for a MAC address
mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
dhcp, netboot, power, err := backend.GetByMac(ctx, mac)

// Enable netboot for a device
netbootData := &data.Netboot{AllowNetboot: true}
err = backend.Put(ctx, mac, nil, netbootData, nil)
```

## Features

- **Ironic Compatibility**: Compatible with OpenStack Ironic DNSMasq configuration
- **Atomic Updates**: Configuration changes are written atomically
- **DHCP Lease Management**: Standard DNSMasq lease file format
- **Conditional DHCP Options**: Support for iPXE conditional options
- **Auto Node ID Generation**: Automatic node ID generation for new devices

## File Management

The backend automatically:

1. Creates the directory structure if it doesn't exist
2. Generates node IDs for new devices (MAC without colons)
3. Manages host and option files atomically
4. Cleans up expired leases
5. Handles missing files gracefully

## Thread Safety

The backend is thread-safe and uses read-write mutexes to protect concurrent access to configuration data.
