package collector

import (
	"sync"
	"time"

	"github.com/nshttpd/mikrotik-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	routeros "gopkg.in/routeros.v2"
)

const (
	namespace = "mikrotik"
	apiPort   = ":8728"
)

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

type collector struct {
	devices    []config.Device
	collectors []metricCollector
}

func WithBGP() CollectorOption {
	return func(c *collector) {
		c.collectors = append(c.collectors, &bgpCollector{})
	}
}

// CollectorOption applies options to collector
type CollectorOption func(*collector)

// NewCollector creates a collector instance
func NewCollector(cfg *config.Config, opts ...CollectorOption) (*collector, error) {
	log.WithFields(log.Fields{
		"numDevices": len(cfg.Devices),
	}).Info("setting up collector for devices")

	c := &collector{
		devices: cfg.Devices,
		collectors: []metricCollector{
			&interfaceCollector{},
			&resourceCollector{},
		},
	}

	for _, o := range opts {
		o(c)
	}

	return c, nil
}

// Describe implements the prometheus.Collector interface.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc

	for _, co := range c.collectors {
		co.describe(ch)
	}
}

// Collect implements the prometheus.Collector interface.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(c.devices))

	for _, dev := range c.devices {
		go func(d *config.Device) {
			c.collectForDevice(d, ch)
			wg.Done()
		}(&dev)
	}

	wg.Wait()
}

func (c *collector) collectForDevice(d *config.Device, ch chan<- prometheus.Metric) {
	begin := time.Now()

	err := c.connectAndCollect(d, ch)

	duration := time.Since(begin)
	var success float64
	if err != nil {
		log.Errorf("ERROR: %s collector failed after %fs: %s", d.Name, duration.Seconds(), err)
		success = 0
	} else {
		log.Debugf("OK: %s collector succeeded after %fs.", d.Name, duration.Seconds())
		success = 1
	}

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), d.Name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, d.Name)
}

func (c *collector) connectAndCollect(d *config.Device, ch chan<- prometheus.Metric) error {
	cl, err := routeros.Dial(d.Address+apiPort, d.User, d.Password)
	if err != nil {
		log.WithFields(log.Fields{
			"device": d.Name,
			"error":  err,
		}).Error("error dialing device")
		return err
	}
	defer cl.Close()

	for _, co := range c.collectors {
		err = co.collect(ch, d, cl)
		if err != nil {
			return err
		}
	}

	return nil
}
