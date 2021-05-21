FROM golang:1.13 AS builder
WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

FROM scratch
COPY --from=builder /go/src/app/mikrotik-exporter /mikrotik-exporter
ENTRYPOINT ["/mikrotik-exporter"]