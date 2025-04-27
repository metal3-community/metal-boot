# Builder
# Build the Go binary
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ./pibmc


FROM scratch
COPY --from=builder /app/pibmc /
COPY --from=builder /app/redfish.example.yaml /config/redfish.yaml
ENTRYPOINT ["/pibmc"]
