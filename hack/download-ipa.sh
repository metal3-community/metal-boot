#!/usr/bin/env bash

mkdir -p internal/firmware/edk2/images
curl -sfLo internal/firmware/edk2/images/ironic-python-agent.initramfs https://tarballs.opendev.org/openstack/ironic-python-agent-builder/dib/files/ipa-debian-arm64-master.initramfs
curl -sfLo internal/firmware/edk2/images/ironic-python-agent.kernel https://tarballs.opendev.org/openstack/ironic-python-agent-builder/dib/files/ipa-debian-arm64-master.kernel
