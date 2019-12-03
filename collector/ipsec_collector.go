package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type ipsecCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newIpsecCollector() routerOSCollector {
	c := &ipsecCollector{}
	c.init()
	return c
}

func (c *ipsecCollector) init() {
	c.props = []string{"src-address", "dst-address", "ph2-state", "invalid", "active", "comment"}

	labelNames := []string{"devicename", "srcdst", "comment"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[1:] {
		c.descriptions[p] = descriptionForPropertyName("ipsec", p, labelNames)
	}
}

func (c *ipsecCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *ipsecCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *ipsecCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/ip/ipsec/policy/print", "?disabled=false", "?dynamic=false", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching interface metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *ipsecCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	srcdst := re.Map["src-address"] + "-" + re.Map["dst-address"]
	comment := re.Map["comment"]

	for _, p := range c.props[2:] {
		c.collectMetricForProperty(p, srcdst, comment, re, ctx)
	}
}

func (c *ipsecCollector) collectMetricForProperty(property, srcdst, comment string, re *proto.Sentence, ctx *collectorContext) {
	desc := c.descriptions[property]
	if value := re.Map[property]; value != "" {
		var v float64
		var err error
		v, err = strconv.ParseFloat(value, 64)

		switch property {
		case "ph2-state":
			if value == "established" {
				v, err = 1, nil
			} else {
				v, err = 0, nil
			}
		case "active", "invalid":
			if value == "true" {
				v, err = 1, nil
			} else {
				v, err = 0, nil
			}
		case "comment":
			return
		}

		if err != nil {
			log.WithFields(log.Fields{
				"device":   ctx.device.Name,
				"srcdst":   srcdst,
				"property": property,
				"value":    value,
				"error":    err,
			}).Error("error parsing ipsec metric value")
			return
		}
		ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, ctx.device.Name, srcdst, comment)
	}
}
