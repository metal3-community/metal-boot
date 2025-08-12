# Builder
# Build the Go binary
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ./cmd/pibmc


FROM scratch
COPY --from=builder /app/pibmc /
COPY --from=builder /app/config.example.yaml /config/config.yaml
ENTRYPOINT ["/pibmc"]
