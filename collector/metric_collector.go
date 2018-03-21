package collector

import (
	"github.com/nshttpd/mikrotik-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	routeros "gopkg.in/routeros.v2"
)

type metricCollector interface {
	describe(ch chan<- *prometheus.Desc)
	collect(ch chan<- prometheus.Metric, device *config.Device, client *routeros.Client) error
}
