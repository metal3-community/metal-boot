#!/usr/bin/env bash

curl -sfLo internal/firmware/edk2/overlays/rpi-poe-plus.dtbo https://github.com/raspberrypi/firmware/raw/refs/heads/master/boot/overlays/rpi-poe-plus.dtbo
curl -sfLo internal/firmware/edk2/overlays/upstream-pi4.dtbo https://github.com/raspberrypi/firmware/raw/refs/heads/master/boot/overlays/upstream-pi4.dtbo
