# Redfish Host Interface Library Configuration

## Overview

The `RedfishPlatformHostInterfaceLib` is a critical component that determines how the UEFI Redfish client discovers and connects to Redfish services. The choice of library implementation depends on your deployment architecture.

## Library Options

### 1. PlatformHostInterfaceLibNull (Recommended for RPi4)

**Use Case**: Remote Redfish service over standard network interface  
**File**: `RedfishPkg/Library/PlatformHostInterfaceLibNull/PlatformHostInterfaceLibNull.inf`

**When to Use**:

- ✅ Connecting to external/remote Redfish services
- ✅ Simulator testing and development
- ✅ Lab environments with dedicated Redfish services
- ✅ Cloud-based or networked Redfish management

**How It Works**:

- Uses standard network interfaces (Ethernet, WiFi)
- Service discovery via EFI variables or static configuration
- No special hardware requirements
- Configuration via `RedfishPlatformConfig.efi` or manual variables

**RPi4 Configuration**:

```c
RedfishPlatformHostInterfaceLib|RedfishPkg/Library/PlatformHostInterfaceLibNull/PlatformHostInterfaceLibNull.inf
```

### 2. PlatformHostInterfaceBmcUsbNicLib (Specialized Use)

**Use Case**: BMC-exposed USB Network Interface Card  
**File**: `RedfishPkg/Library/PlatformHostInterfaceBmcUsbNicLib/PlatformHostInterfaceBmcUsbNicLib.inf`

**When to Use**:

- ❌ **NOT typically needed for RPi4**
- ✅ Server platforms with dedicated BMC
- ✅ Enterprise hardware with IPMI-capable BMC
- ✅ BMC exposes USB NIC for in-band management

**How It Works**:

- Discovers BMC-exposed USB NIC via IPMI commands
- Automatic service discovery through SMBIOS Type 42 records
- Requires IPMI infrastructure and BMC support
- Uses Redfish Credential Bootstrapping

**Requirements**:

- BMC with USB NIC exposure capability
- IPMI channel configuration (802.3 LAN/IPMB 1.0)
- MAC address matching (Host MAC = BMC MAC - 1)
- Network configuration via IPMI transport commands

## Current RPi4 Configuration

### Why PlatformHostInterfaceLibNull?

The Raspberry Pi 4 typically does **not** have:

- ❌ Dedicated Baseboard Management Controller (BMC)
- ❌ BMC-exposed USB NIC
- ❌ IPMI infrastructure
- ❌ SMBIOS Type 42 records from BMC

The Raspberry Pi 4 **does** have:

- ✅ Standard Ethernet interface
- ✅ WiFi capability
- ✅ Standard UEFI network stack
- ✅ Ability to connect to external services

### Configuration Examples

#### Current Setup (Remote Service)

```bash
# Configure for remote Redfish service
./testing/configure-redfish-client.sh --type simulator --server 127.0.0.1 --port 5001
```

Creates configuration:

```c
RedfishPlatformHostInterfaceLib|RedfishPkg/Library/PlatformHostInterfaceLibNull/PlatformHostInterfaceLibNull.inf
```

#### Hypothetical BMC Setup (Not Typical for RPi4)

```bash
# This would be for enterprise hardware with BMC
./testing/configure-redfish-client.sh --type bmcd --server 192.168.1.100
```

Would create:

```c
RedfishPlatformHostInterfaceLib|RedfishPkg/Library/PlatformHostInterfaceBmcUsbNicLib/PlatformHostInterfaceBmcUsbNicLib.inf
```

## Implementation Details

### PlatformHostInterfaceLibNull Implementation

**Service Discovery**:

1. Reads EFI variables set by `RedfishPlatformConfig.efi`
2. Creates SMBIOS Type 42 records from variable data
3. Provides network configuration to Redfish discovery driver

**Variables Used**:

- `HostIpAssignmentType`: Static (1) or Auto (3)
- `HostIpAddress`: Local IP address
- `RedfishServiceIpAddress`: Remote service IP
- `RedfishServiceIpPort`: Service port

### PlatformHostInterfaceBmcUsbNicLib Implementation

**Service Discovery**:

1. Scans for BMC-exposed USB NIC via IPMI
2. Issues NetFn Transport commands for network config
3. Automatically creates SMBIOS Type 42 records
4. Bootstraps credentials via IPMI Group Extension commands

**IPMI Commands Used**:

- App NetFn (0x06) Command 0x42: Channel medium/protocol check
- Transport NetFn (0x0C) Command 0x02: MAC, IP, subnet, gateway
- Group Ext NetFn (0x2C) Command 0x52/0x02: Credential bootstrapping

## Makefile Configuration

The Makefile has been updated to use the correct library for RPi4:

```makefile
# Remove BMC USB NIC library (not needed for remote Redfish service)
@sed -i '/RedfishPlatformHostInterfaceLib.*PlatformHostInterfaceBmcUsbNicLib/d' platforms/Platform/RaspberryPi/RPi4/RPi4.dsc || true
```

This ensures we only use `PlatformHostInterfaceLibNull` which is appropriate for:

- Remote Redfish service connections
- Standard network interface usage
- EFI variable-based configuration

## Summary

For the RPi4 Redfish client implementation:

✅ **Use**: `PlatformHostInterfaceLibNull`  
✅ **Reason**: Connecting to remote Redfish services over standard network  
✅ **Configuration**: Via EFI variables or static configuration  
✅ **Benefits**: Simple, flexible, works with any network-accessible Redfish service

❌ **Don't Use**: `PlatformHostInterfaceBmcUsbNicLib`  
❌ **Reason**: RPi4 doesn't have BMC or USB NIC exposure  
❌ **Requirements**: Would need IPMI infrastructure not present on RPi4

This configuration allows the RPi4 to act as a Redfish client that can connect to any standard Redfish service over the network, which is the most common and useful deployment scenario.
