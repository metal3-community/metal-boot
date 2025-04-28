# Firmware Package

This package provides firmware management functionality for the BMC Pi project.

## Structure

- `firmware.go`: Main entry point for the package
- `edk2/`: EDK2 firmware specific code and embedded files
- `efi/`: EFI variable and device path handling
- `manager/`: Firmware manager interface and implementations
- `types/`: Common firmware-related types and structures
- `update/`: Firmware update handling
- `util/`: Utility functions for firmware operations
- `varstore/`: Variable store interface and implementations

## Usage

```go
import (
    "github.com/bmcpi/pibmc/internal/firmware"
    "github.com/go-logr/logr"
)

func main() {
    logger := // initialize your logger
    
    // Create a firmware manager
    manager, err := firmware.CreateManager("/path/to/firmware.bin", logger)
    if err != nil {
        // handle error
    }
    
    // Use the manager to manipulate firmware
    bootEntries, err := manager.GetBootEntries()
    // ...
    
    // Save changes
    if err := manager.SaveChanges(); err != nil {
        // handle error
    }
}
```

## Manager Interface

The `FirmwareManager` interface provides methods for:

- Boot order management
- Network configuration
- UEFI variable access
- Firmware updates
- System information

See the interface definition in `manager/manager.go` for details.
