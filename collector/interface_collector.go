package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type interfaceCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newInterfaceCollector() routerOSCollector {
	c := &interfaceCollector{}
	c.init()
	return c
}

func (c *interfaceCollector) init() {
	c.props = []string{"name", "comment", "rx-byte", "tx-byte", "rx-packet", "tx-packet", "rx-error", "tx-error", "rx-drop", "tx-drop"}

	labelNames := []string{"name", "address", "interface", "comment"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[1:] {
		c.descriptions[p] = descriptionForPropertyName("interface", p, labelNames)
	}
}

func (c *interfaceCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *interfaceCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *interfaceCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/interface/print", "?disabled=false", "?running=true", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching interface metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *interfaceCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	name := re.Map["name"]
	comment := re.Map["comment"]

	for _, p := range c.props[2:] {
		c.collectMetricForProperty(p, name, comment, re, ctx)
	}
}

func (c *interfaceCollector) collectMetricForProperty(property, iface, comment string, re *proto.Sentence, ctx *collectorContext) {
	desc := c.descriptions[property]
	if value := re.Map[property]; value != "" {
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.WithFields(log.Fields{
				"device":    ctx.device.Name,
				"interface": iface,
				"property":  property,
				"value":     value,
				"error":     err,
			}).Error("error parsing interface metric value")
			return
		}
		ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, ctx.device.Name, ctx.device.Address, iface, comment)
	}
}
