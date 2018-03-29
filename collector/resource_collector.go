package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
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

func (c *resourceCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *resourceCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/system/resource/print", "=.proplist="+strings.Join(resourceProps, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching system resource metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *resourceCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	for _, p := range resourceProps {
		c.collectMetricForProperty(p, re, ctx)
	}
}

func (c *resourceCollector) collectMetricForProperty(property string, re *proto.Sentence, ctx *collectorContext) {
	v, err := strconv.ParseFloat(re.Map[property], 64)
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing system resource metric value")
		return
	}

	desc := resourceDescriptions[property]
	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, ctx.device.Name, ctx.device.Address)
}
