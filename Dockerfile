# Build
FROM golang:alpine AS build

RUN apk add --update git make build-base && \
    rm -rf /var/cache/apk/*

WORKDIR /go/src/github.com/nshttpd/mikrotik-exporter
COPY . /go/src/github.com/nshttpd/mikrotik-exporter
RUN make

# Runtime
FROM alpine:3.6

EXPOSE 9436

COPY scripts/start.sh /app/
COPY --from=build /go/src/github.com/nshttpd/mikrotik-exporter/mikrotik-exporter /app/

RUN chmod 755 /app/*

ENTRYPOINT ["/app/start.sh"]
