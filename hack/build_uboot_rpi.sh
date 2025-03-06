#!/bin/bash

set -e

docker build -t u-root -f hack/Dockerfile .

docker run --rm -v $(pwd):/dst u-root cp /build/u-boot/u-boot.bin /build/u-boot/arch/arm/dts/bcm2711-rpi-4-b.dtb /build/u-boot/dts/dt.dtb /dst/internal/firmware/uboot/
