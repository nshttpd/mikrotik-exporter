FROM golang:alpine as builder
RUN \
  cd / && \
  apk add --no-cache git ca-certificates make && \
  git clone https://github.com/nshttpd/mikrotik-exporter.git && \
  cd /mikrotik-exporter && \
  go get -d -v && \
  make build


FROM alpine
COPY scripts/start.sh /app/
COPY --from=builder /mikrotik-exporter/mikrotik-exporter /app/mikrotik-exporter
RUN chmod 755 /app/*

USER nobody
EXPOSE 9436

ENTRYPOINT ["/app/start.sh"]
