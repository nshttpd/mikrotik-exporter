FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG SHORT_SHA=unknown
RUN CGO_ENABLED=0 go build -ldflags "-X main.appVersion=${VERSION} -X main.shortSha=${SHORT_SHA}" -o mikrotik-exporter .

FROM alpine:3.21

EXPOSE 9436

COPY scripts/start.sh /app/
COPY --from=builder /build/mikrotik-exporter /app/mikrotik-exporter

RUN chmod 755 /app/*

ENTRYPOINT ["/app/start.sh"]
