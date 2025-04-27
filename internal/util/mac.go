package util

import (
	"bytes"
	"net"
)

// IsRaspberryPI checks if the mac address is from a Raspberry PI by matching prefixes against OUI registrations of the Raspberry Pi Trading Ltd.
// https://www.netify.ai/resources/macs/brands/raspberry-pi
// https://udger.com/resources/mac-address-vendor-detail?name=raspberry_pi_foundation
// https://macaddress.io/statistics/company/27594
func IsRaspberryPI(mac net.HardwareAddr) bool {
	prefixes := [][]byte{
		{0xb8, 0x27, 0xeb}, // B8:27:EB
		{0xdc, 0xa6, 0x32}, // DC:A6:32
		{0xe4, 0x5f, 0x01}, // E4:5F:01
		{0x28, 0xcd, 0xc1}, // 28:CD:C1
		{0xd8, 0x3a, 0xdd}, // D8:3A:DD
	}
	for _, prefix := range prefixes {
		if bytes.HasPrefix(mac, prefix) {
			return true
		}
	}

	return false
}
