package collector

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type cloudCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newCloudCollector() routerOSCollector {
	c := &cloudCollector{}
	c.init()
	return c
}

func (c *cloudCollector) init() {
	c.props = []string{"public-address", "ddns-enabled"}
	labelNames := []string{"name", "address", "public_address"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[1:] {
		c.descriptions[p] = descriptionForPropertyName("cloud", p, labelNames)
	}
}

func (c *cloudCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *cloudCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *cloudCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/ip/cloud/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching cloud metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *cloudCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	publicIp := re.Map["public-address"]

	for _, p := range c.props[1:] {
		c.collectMetricForProperty(p, publicIp, re, ctx)
	}
}

func (c *cloudCollector) collectMetricForProperty(property, publicIp string, re *proto.Sentence, ctx *collectorContext) {
	desc := c.descriptions[property]
	if value := re.Map[property]; value != "" {
		var numericValue float64
		switch value {
		case "false":
			numericValue = 0
		case "true":
			numericValue = 1
		default:
			log.WithFields(log.Fields{
				"device":         ctx.device.Name,
				"public-address": publicIp,
				"property":       property,
				"value":          value,
				"error":          fmt.Errorf("unexpected cloud ddns-enabled value"),
			}).Error("error parsing cloud metric value")
		}

		ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, numericValue, ctx.device.Name, ctx.device.Address, publicIp)
	}
}
