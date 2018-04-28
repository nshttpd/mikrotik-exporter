FROM alpine:3.6

EXPOSE 9436

COPY scripts/start.sh /app/
COPY mikrotik-exporter /app/

RUN chmod 755 /app/*

ENTRYPOINT ["/app/start.sh"]