FROM alpine:3.6

EXPOSE 9436

COPY scripts/start.sh /app/
COPY dist/mikrotik-exporter_linux_amd64 /app/mikrotik-exporter

RUN chmod 755 /app/*

ENTRYPOINT ["/app/start.sh"]