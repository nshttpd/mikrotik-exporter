#!/bin/sh

if [ -z "$CONFIG_FILE" ]
then
    /app/mikrotik-exporter -device $DEVICE -address $ADDRESS -user $USER -password $PASSWORD
else
    /app/mikrotik-exporter -config-file $CONFIG_FILE
fi
