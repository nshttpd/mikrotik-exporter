package collector

import (
	"github.com/prometheus/client_golang/prometheus"
)

type routerOSCollector interface {
	describe(ch chan<- *prometheus.Desc)
	collect(ctx *collectorContext) error
}
