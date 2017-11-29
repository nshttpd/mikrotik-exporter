package collector

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

const namespace = "mikrotik"

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"mikrotik_exporter: duration of a collector scrape",
		[]string{"device"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"mikrotik_exporter: whether a collector succeeded",
		[]string{"device"},
		nil,
	)
)

type deviceCollector struct {
	Devices []Device
}

func NewDeviceCollector(cfg Config) (*deviceCollector, error) {
	devices := make([]Device, len(cfg.Devices))

	cfg.Logger.Info("setting up collector for devices",
		"numDevices", len(cfg.Devices),
	)

	copy(devices, cfg.Devices)

	return &deviceCollector{Devices: devices}, nil
}

// Describe implements the prometheus.Collector interface.
func (d deviceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (d deviceCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(d.Devices))
	for _, device := range d.Devices {
		go func(d Device) {
			execute(d, ch)
			wg.Done()
		}(device)
	}
	wg.Wait()
}

func execute(d Device, ch chan<- prometheus.Metric) {
	begin := time.Now()
	err := d.Update(ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		log.Errorf("ERROR: %s collector failed after %fs: %s", d.name, duration.Seconds(), err)
		success = 0
	} else {
		log.Debugf("OK: %s collector succeeded after %fs.", d.name, duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), d.name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, d.name)
}
