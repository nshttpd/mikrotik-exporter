#!/bin/sh

if [ ! -x /app/mikrotik-exporter ]; then
  chmod 755 /app/mikrotik-exporter
fi

if [ -z "$CONFIG_FILE" ]
then
    /app/mikrotik-exporter -device $DEVICE -address $ADDRESS -user $USER -password $PASSWORD
else
    /app/mikrotik-exporter -config-file $CONFIG_FILE
fi
