# DHCP Fallback Configuration

This document describes how to configure DHCP fallback responses for unknown devices that are not found in the backend configuration.

## Overview

By default, the DHCP server only responds to devices that have entries in the backend configuration file. With fallback enabled, the server can provide default DHCP configuration to unknown devices.

## Configuration

Add the following settings to your `config.yaml` under the `dhcp` section:

```yaml
dhcp:
  enabled: true
  
  # Fallback configuration for unknown devices
  fallback_enabled: true                    # Enable/disable fallback responses
  fallback_ip_start: "192.168.1.100"      # Start of IP pool for unknown devices  
  fallback_ip_end: "192.168.1.200"        # End of IP pool for unknown devices
  fallback_gateway: "192.168.1.1"         # Default gateway
  fallback_subnet: "255.255.255.0"        # Subnet mask
  fallback_dns:                            # DNS servers
    - "8.8.8.8"
    - "8.8.4.4"
  fallback_domain: "local"                 # Domain name
  fallback_netboot: true                   # Allow netboot for unknown devices
```

## How It Works

1. **Known Devices**: Devices with MAC addresses in the backend configuration receive their configured IP and options as usual.

2. **Unknown Devices**: Devices not in the backend receive:
   - A dynamically assigned IP from the configured pool (based on MAC hash for consistency)
   - Default network configuration (gateway, DNS, domain)
   - Netboot permission based on `fallback_netboot` setting

3. **IP Assignment**: Unknown devices get consistent IP addresses based on their MAC address hash, ensuring the same device always gets the same IP.

## Benefits

- **Zero-touch deployment**: New devices can immediately get network access
- **Raspberry Pi support**: Unknown RPIs can network boot if `fallback_netboot: true`
- **Consistent assignment**: Same MAC always gets same IP
- **Secure by default**: Fallback is disabled by default

## Security Considerations

- Only enable fallback in trusted networks
- Use a separate IP pool range for unknown devices
- Consider setting `fallback_netboot: false` if you don't want unknown devices to network boot
- Monitor logs for unknown device activity

## Logging

When fallback is used, you'll see log messages like:
```
"no reservation found, using fallback configuration"
```

This helps distinguish between configured devices and fallback responses.
