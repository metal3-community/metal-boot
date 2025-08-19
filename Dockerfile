# Builder
# Build the Go binary
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o metal-boot ./cmd/metal-boot

FROM scratch
COPY --from=builder /app/metal-boot /
COPY configs/config.example.yaml /config/config.yaml
ENTRYPOINT ["/metal-boot"]
