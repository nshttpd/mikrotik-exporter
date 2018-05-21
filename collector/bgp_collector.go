package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type bgpCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newBGPCollector() routerOSCollector {
	c := &bgpCollector{}
	c.init()
	return c
}

func (c *bgpCollector) init() {
	c.props = []string{"name", "remote-as", "state", "prefix-count", "updates-sent", "updates-received", "withdrawn-sent", "withdrawn-received"}

	const prefix = "bgp"
	labelNames := []string{"name", "address", "session", "asn"}

	c.descriptions = make(map[string]*prometheus.Desc)
	c.descriptions["state"] = description(prefix, "up", "BGP session is established (up = 1)", labelNames)

	for _, p := range c.props[3:] {
		c.descriptions[p] = descriptionForPropertyName(prefix, p, labelNames)
	}
}

func (c *bgpCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *bgpCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *bgpCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/routing/bgp/peer/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching bgp metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *bgpCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	asn := re.Map["remote-as"]
	session := re.Map["name"]

	for _, p := range c.props[2:] {
		c.collectMetricForProperty(p, session, asn, re, ctx)
	}
}

func (c *bgpCollector) collectMetricForProperty(property, session, asn string, re *proto.Sentence, ctx *collectorContext) {
	desc := c.descriptions[property]
	v, err := c.parseValueForProperty(property, re.Map[property])
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"session":  session,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing bgp metric value")
		return
	}

	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, session, asn)
}

func (c *bgpCollector) parseValueForProperty(property, value string) (float64, error) {
	if property == "state" {
		if value == "established" {
			return 1, nil
		}

		return 0, nil
	}

	if value == "" {
		return 0, nil
	}

	return strconv.ParseFloat(value, 64)
}
