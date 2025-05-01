package main

import (
	"fmt"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
)

func main() {
	// Example device paths
	samples := []string{
		"ACPI(hid=0xa0841d0,uid=0x0)/PCI(dev=00:0)/PCI(dev=00:0)/USB(port=2)",
		"PCI(dev=00:0)/USB(port=2)/ISCSI(iqn.target.name)",
		"PCI(dev=03:2)/Sata(5)/Partition(nr=1)",
		"PCI(dev=00:0)/MAC(aa:bb:cc:dd:ee:ff)/IPv4(192.168.1.1)",
		"MAC()/IPv4()",
		"URI(https://example.com/boot.efi)", // Test URI by itself
		"FvFileName(9a15aa37-d555-4a4e-b541-86391ff68164)",
		"FvName(9a15aa37-d555-4a4e-b541-86391ff68164)/FvFileName(7c04a583-9e3e-4f1c-ad65-e05268d0b4d1)",
		"VendorHW(100c2cfa-b586-4198-9b4c-1683d195b1da)",
		"ACPI(hid=0xa0841d0,uid=0x0)/PCI(dev=00:0)/PCI(dev=00:0)/USB(port=2)",
		"MAC()/IPv4()",
		"MAC()/IPv6()",
		"MAC()/IPv4()/URI()",
		"MAC()/IPv6()/URI()",
		"FvName(9a15aa37-d555-4a4e-b541-86391ff68164)/FvFileName(7c04a583-9e3e-4f1c-ad65-e05268d0b4d1)",
	}

	for _, sample := range samples {
		dp, err := efi.ParseDevicePathFromString(sample)
		if err != nil {
			fmt.Printf("Error parsing %s: %v\n", sample, err)
			continue
		}

		fmt.Printf("Original: %s\n", sample)
		fmt.Printf("Parsed  : %s\n", dp.String())
		fmt.Println("----")
	}
}
