# go-redfish-uefi

## TFTP Auto Serve UEFI Rpi based on mac prefix

## DHCP Proxy - Serve http ipxe config files from remote sidero metal

## IPMI/Redfish API for changing RPI.fd firmware vars in TFTP dir and power state

Serve read/write TFTP server with rpi4 uefi firmware for each device. Add Redfish api endpoints for fw var modification.

Use go-uefi to patch firmware: https://github.com/Foxboron/go-uefi

Optional: Provide power on/off via unifi poe switch.

# DESIGN

TFTP should exist here

Power state should be web hook

# Requirements

Should contain method for power on and off + state
