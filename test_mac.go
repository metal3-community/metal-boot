package main

import (
	"fmt"
	"net"

	"github.com/metal3-community/metal-boot/internal/util"
)

func main() {
	mac, _ := net.ParseMAC("d8:3a:dd:61:4d:15")
	fmt.Printf("MAC: %s\n", mac.String())
	fmt.Printf("IsRaspberryPI: %v\n", util.IsRaspberryPI(mac))

	// Test a few other MACs
	macX86, _ := net.ParseMAC("00:11:22:33:44:55")
	fmt.Printf("MAC: %s\n", macX86.String())
	fmt.Printf("IsRaspberryPI: %v\n", util.IsRaspberryPI(macX86))
}
