# Use minimal Alpine Linux base image
FROM alpine:3.19

# Install required packages for iPXE build
RUN apk add --no-cache \
    build-base \
    git \
    musl-dev \
    perl \
    mtools \
    xorriso \
    gcc-aarch64-none-elf \
    binutils-aarch64-none-elf \
    tar \
    curl \
    zlib-dev \
    openssl-dev \
    xz-dev \
    syslinux \
    && rm -rf /var/cache/apk/*

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
RUN make bin/undionly.kpxe -j4 EMBED=embed.ipxe
RUN make bin/ipxe.lkrn -j4 EMBED=embed.ipxe

# Copy iPXE customizations for EFI
COPY internal/ipxe/binary/script/ipxe-customizations/general.efi.h config/local/general.h


# Copy iPXE customizations for ISA
COPY internal/ipxe/binary/script/ipxe-customizations/isa.h config/local/

RUN make EMBED=embed.ipxe -j4 bin-x86_64-efi/ipxe.efi
RUN make EMBED=embed.ipxe -j4 bin-x86_64-efi/ipxe.iso
RUN rm config/local/isa.h

RUN sed -i.bak '/^WORKAROUND_CFLAGS/ s|^|#|' "arch/arm64/Makefile"
# Copy iPXE customizations for ARM64 EFI
COPY internal/ipxe/binary/script/ipxe-customizations/nap.h config/local/
# Build iPXE with ARM64 EFI using proper cross-compiler
RUN make bin-arm64-efi/snp.efi -j2 EMBED=embed.ipxe CROSS=aarch64-none-elf-

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

ENTRYPOINT ["/bin/sh"]
