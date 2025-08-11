# Use Ubuntu base image
FROM ubuntu:22.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install required packages for iPXE build and cross-compilation
RUN apt-get update && apt-get install -y \
    build-essential \
    git \
    perl \
    mtools \
    xorriso \
    gcc-aarch64-linux-gnu \
    binutils-aarch64-linux-gnu \
    gcc-arm-linux-gnueabihf \
    binutils-arm-linux-gnueabihf \
    gcc-x86-64-linux-gnu \
    binutils-x86-64-linux-gnu \
    tar \
    curl \
    wget \
    zlib1g-dev \
    libssl-dev \
    liblzma-dev \
    syslinux-common \
    syslinux-efi \
    isolinux \
    make \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /build

# Clone iPXE source code
RUN git clone https://github.com/ipxe/ipxe.git

# Set working directory to ipxe/src
WORKDIR /build/ipxe/src

# Create local configuration directory
RUN mkdir -p config/local

# Copy embedded iPXE script
COPY internal/ipxe/binary/script/embed.ipxe embed.ipxe

# Copy common iPXE customizations
COPY internal/ipxe/binary/script/ipxe-customizations/colour.h config/local/
COPY internal/ipxe/binary/script/ipxe-customizations/common.h config/local/
COPY internal/ipxe/binary/script/ipxe-customizations/console.h config/local/
COPY internal/ipxe/binary/script/ipxe-customizations/crypto.h config/local/

# Copy iPXE customizations for undionly
COPY internal/ipxe/binary/script/ipxe-customizations/general.undionly.h config/local/general.h
RUN make bin/undionly.kpxe -j4 EMBED=embed.ipxe CROSS=x86_64-linux-gnu-
RUN make bin/ipxe.lkrn -j4 EMBED=embed.ipxe CROSS=x86_64-linux-gnu-

# Copy iPXE customizations for EFI
COPY internal/ipxe/binary/script/ipxe-customizations/general.efi.h config/local/general.h


# Copy iPXE customizations for ISA
COPY internal/ipxe/binary/script/ipxe-customizations/isa.h config/local/

RUN make EMBED=embed.ipxe -j4 bin-x86_64-efi/ipxe.efi CROSS=x86_64-linux-gnu-
RUN make EMBED=embed.ipxe -j4 bin-x86_64-efi/ipxe.iso CROSS=x86_64-linux-gnu-
RUN rm config/local/isa.h

RUN sed -i.bak '/^WORKAROUND_CFLAGS/ s|^|#|' "arch/arm64/Makefile"
# Copy iPXE customizations for ARM64 EFI
COPY internal/ipxe/binary/script/ipxe-customizations/nap.h config/local/
# Build iPXE with ARM64 EFI using proper cross-compiler
RUN make bin-arm64-efi/snp.efi -j2 EMBED=embed.ipxe CROSS=aarch64-linux-gnu-

# Create output directory and copy the built EFI file
RUN mkdir -p /output/ \
    && cp bin/undionly.kpxe /output/ \
    && cp bin/ipxe.lkrn /output/ \
    && cp bin-arm64-efi/snp.efi /output/ \
    && cp bin-x86_64-efi/ipxe.efi /output/ \
    && cp bin-x86_64-efi/ipxe.iso /output/ \
    && ls -la /output/

# Set the output directory as volume
VOLUME ["/output"]

ENTRYPOINT ["/bin/bash"]
