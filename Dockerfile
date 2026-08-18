# syntax=docker/dockerfile:1
#
# Multi-stage build replacing ko:
#   1. apko renders apko.yaml into a minimal Wolfi/Ironic root filesystem tar
#      (cross-built with --arch, so no QEMU is needed for multi-arch builds).
#   2. The metal-boot binary is cross-compiled and UPX-compressed.
#   3. Both are layered onto scratch.
#
#   docker buildx build --platform linux/amd64,linux/arm64 -t <ref> .

FROM cgr.dev/chainguard/apko:latest AS apko-bin

# Build the base rootfs on the native build platform; apko cross-builds the
# target architecture. wolfi-base provides the shell and tar needed to unpack
# the OCI tarball apko emits.
FROM --platform=$BUILDPLATFORM cgr.dev/chainguard/wolfi-base:latest AS rootfs
COPY --from=apko-bin /usr/bin/apko /usr/bin/apko
WORKDIR /work
COPY apko.yaml .
ARG TARGETARCH
RUN apko build --sbom=false --arch "$TARGETARCH" apko.yaml apko-alpine:edge /work/image.tar \
 && mkdir -p /work/oci /rootfs \
 && tar -xf /work/image.tar -C /work/oci \
 && layer="" \
 && for f in $(find /work/oci -type f); do \
      if tar -tzf "$f" >/dev/null 2>&1; then layer="$f"; fi; \
    done \
 && test -n "$layer" \
 && tar -xzf "$layer" -C /rootfs

# Cross-compile metal-boot on the build platform and compress it with UPX.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
RUN apk add --no-cache upx
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG TARGETOS TARGETARCH
ARG GITREV=unknown
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -trimpath -ldflags "-s -w -X main.GitRev=${GITREV}" \
      -o /out/metal-boot ./cmd/metal-boot \
 && upx --best --lzma /out/metal-boot

# Assemble: apko rootfs + metal-boot on scratch. The image config that apko
# would have written (user, workdir, stop signal) is restated here because
# only the filesystem survives the tar round-trip.
FROM scratch
COPY --from=rootfs /rootfs/ /
COPY --from=build /out/metal-boot /usr/bin/metal-boot
COPY configs/config.example.yaml /config/config.yaml

LABEL org.opencontainers.image.title="metal-boot" \
      org.opencontainers.image.description="Metal Boot with Ironic all-in-one" \
      org.opencontainers.image.source="https://github.com/metal3-community/metal-boot"

USER ironic
WORKDIR /var/lib/ironic
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/bin/metal-boot"]
