package collector

import (
	"github.com/Visteras/mikrotik-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/routeros.v2"
)

type collectorContext struct {
	ch     chan<- prometheus.Metric
	device *config.Device
	client *routeros.Client
}
