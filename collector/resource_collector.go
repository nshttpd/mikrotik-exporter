package collector

import (
	"strconv"
	"strings"

	"github.com/nshttpd/mikrotik-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2"
	"gopkg.in/routeros.v2/proto"
)

var (
	resourceLabelNames   = []string{"name", "address"}
	resourceProps        = []string{"free-memory", "total-memory", "cpu-load", "free-hdd-space", "total-hdd-space"}
	resourceDescriptions map[string]*prometheus.Desc
)

func init() {
	resourceDescriptions = make(map[string]*prometheus.Desc)
	for _, p := range resourceProps {
		resourceDescriptions[p] = descriptionForPropertyName("system", p, resourceLabelNames)
	}
}

type resourceCollector struct {
}

func (c *resourceCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range resourceDescriptions {
		ch <- d
	}
}

func (c *resourceCollector) collect(ch chan<- prometheus.Metric, device *config.Device, client *routeros.Client) error {
	stats, err := c.fetch(client, device)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, device, ch)
	}

	return nil
}

func (c *resourceCollector) fetch(client *routeros.Client, device *config.Device) ([]*proto.Sentence, error) {
	reply, err := client.Run("/system/resource/print", "=.proplist="+strings.Join(resourceProps, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": device.Name,
			"error":  err,
		}).Error("error fetching system resource metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *resourceCollector) collectForStat(re *proto.Sentence, device *config.Device, ch chan<- prometheus.Metric) {
	for _, p := range resourceProps {
		c.collectMetricForProperty(p, device, re, ch)
	}
}

func (c *resourceCollector) collectMetricForProperty(property string, device *config.Device, re *proto.Sentence, ch chan<- prometheus.Metric) {
	v, err := strconv.ParseFloat(re.Map[property], 64)
	if err != nil {
		log.WithFields(log.Fields{
			"device":   device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing system resource metric value")
		return
	}

	desc := resourceDescriptions[property]
	ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, device.Name, device.Address)
}
