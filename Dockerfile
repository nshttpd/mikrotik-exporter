# Builder stage
FROM golang:1.25.1-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go test ./...

# Build the binary
RUN CGO_ENABLED=0 go build -o /app/mikrotik-exporter .

# Final stage
FROM debian:12.12-slim

WORKDIR /app

EXPOSE 9436

# Copy the binary from the builder stage
COPY --from=builder /app/mikrotik-exporter /app/mikrotik-exporter

# Copy the start.sh script
COPY scripts/start.sh /app/

RUN chmod 755 /app/*

ENTRYPOINT ["/app/start.sh"]
