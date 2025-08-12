# iPXE Build Environment for ARM64 EFI

This directory contains a Docker-based build environment for compiling iPXE for ARM64 EFI platforms, specifically for Raspberry Pi 4.

## Files

- `../../hack/Dockerfile.ipxe` - Dockerfile for building iPXE for ARM64 EFI
- `main.go` - Go program to build and extract iPXE EFI binary

## Quick Start

### Build iPXE using go generate
```bash
go generate ./...
```

### Build iPXE manually
```bash
go run ./cmd/ipxe
```

## Configuration

The iPXE build includes the following custom configurations:

### Power Management (nap.h)
- Disables EFI x86 and EFI ARM power management
- Uses NULL power management driver for better compatibility

### General Features (general.h)
- Name resolution command (nslookup)
- Ping command
- NTP commands 
- VLAN commands
- EFI image support
- HTTPS, FTP, NFS, and local file system support

## Embedded Script

The build includes an embedded iPXE script that:

1. Attempts DHCP configuration with retry on failure
2. Displays the acquired IP address and netmask
3. Attempts autoboot with retry on failure

The script ensures robust network booting by retrying failed operations.

## Output

The build process generates:
- `internal/firmware/edk2/snp-uboot.efi` - iPXE EFI binary for ARM64

## Integration with pibmc

The iPXE EFI binary is automatically built during `go generate` and placed in the firmware directory where it can be used by the pibmc system for:

- Network booting via EFI
- DHCP-based boot configurations
- Automated network discovery and boot processes

## Build Dependencies

The build environment uses Alpine Linux with the following packages:
- `build-base` - Essential build tools
- `gcc-aarch64-none-elf` - ARM64 cross-compiler
- `binutils-aarch64-none-elf` - ARM64 binary utilities
- `perl` - Required for iPXE build system
- `mtools`, `xorriso` - Additional build utilities

## Customization

To modify the iPXE configuration:

1. Edit the configuration in `../../hack/Dockerfile.ipxe`
2. Modify the embedded script as needed
3. Rebuild using `go run .` (from this directory) or `go run ./cmd/ipxe` (from project root)

The build system supports embedding custom iPXE scripts and configuration changes through the Dockerfile.
