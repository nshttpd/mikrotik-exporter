FROM arm32v7/busybox:1.27.2

EXPOSE 9090

COPY scripts/start.sh /app/
COPY dist/mikrotik-exporter_linux_arm /app/mikrotik-exporter

RUN chmod 755 /app/*

ENTRYPOINT ["/app/start.sh"]