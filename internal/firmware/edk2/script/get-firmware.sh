#!/usr/bin/env bash

uefi_firmware="https://github.com/pftf/RPi4/releases/download/v1.38/RPi4_UEFI_Firmware_v1.38.zip" # "https://github.com/appkins/rpi4-ipxe/releases/download/v0.0.1/RPi4_UEFI_Firmware_v0.0.2.zip" # https://github.com/pftf/RPi4/releases/download/v1.38/RPi4_UEFI_Firmware_v1.38.zip

mkdir -p internal/rpi4
wget -qO- "${uefi_firmware}" | bsdtar -xvf- -C internal/rpi4

virt-fw-vars --inplace internal/rpi4/RPI_EFI.fd --set-json internal/rpi4/script/fw-vars.json
