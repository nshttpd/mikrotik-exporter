FROM golang:1.25 AS builder
WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go mod tidy && go build -v -o /usr/local/bin/mikrotik-exporter

FROM debian:12.13-slim
EXPOSE 9436
COPY scripts/start.sh /app/
COPY --from=builder /usr/local/bin/mikrotik-exporter /app/mikrotik-exporter
RUN chmod 755 /app/*
ENTRYPOINT ["/app/start.sh"]
