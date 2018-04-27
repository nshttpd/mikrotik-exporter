package collector

import "github.com/prometheus/client_golang/prometheus"

type opticsCollector struct {
}

func newOpticsCollector() routerOSCollector {
	return &opticsCollector{}
}

func (c *opticsCollector) describe(ch chan<- *prometheus.Desc) {

}

func (c *opticsCollector) collect(ctx *collectorContext) error {
	return nil
}
