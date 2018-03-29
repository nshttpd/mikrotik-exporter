package collector

import (
	"github.com/prometheus/client_golang/prometheus"
)

type metricCollector interface {
	describe(ch chan<- *prometheus.Desc)
	collect(ctx *collectorContext) error
}
