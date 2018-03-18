package collector

import (
	"github.com/nshttpd/mikrotik-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/routeros.v2"
)

var (
	resourceLabelNames   = []string{"name", "address"}
	resourceProps        = []string{"free-memory", "total-memory", "cpu-load", "free-hdd-space", "total-hdd-space"}
	resourceDescriptions map[string]*prometheus.Desc
)

func init() {
	resourceDescriptions = make(map[string]*prometheus.Desc)
	for _, p := range resourceProps[1:] {
		resourceDescriptions[p] = descriptionForPropertyName("resource", p, interfaceLabelNames)
	}
}

type resourceCollector struct {
}

func (c *resourceCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range interfaceDescriptions {
		ch <- d
	}
}

func (c *resourceCollector) collect(ch chan<- prometheus.Metric, device *config.Device, client *routeros.Client) error {
	return nil
}
