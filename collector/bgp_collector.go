package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

var (
	bgpabelNames    = []string{"name", "address", "session", "asn"}
	bgpProps        = []string{"name", "remote-as", "state", "prefix-count", "updates-sent", "updates-received", "withdrawn-sent", "withdrawn-received"}
	bgpDescriptions map[string]*prometheus.Desc
)

func init() {
	bgpDescriptions = make(map[string]*prometheus.Desc)
	bgpDescriptions["state"] = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "bgp", "up"),
		"BGP session is established (up = 1)",
		bgpabelNames,
		nil,
	)
	for _, p := range bgpProps[3:] {
		bgpDescriptions[p] = descriptionForPropertyName("bgp", p, bgpabelNames)
	}
}

type bgpCollector struct {
}

func (c *bgpCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range bgpDescriptions {
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
	reply, err := ctx.client.Run("/routing/bgp/peer/print", "=.proplist="+strings.Join(bgpProps, ","))
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
	var session, asn string
	for _, p := range bgpProps {
		if p == "name" {
			session = re.Map[p]
		} else if p == "remote-as" {
			asn = re.Map[p]
		} else {
			c.collectMetricForProperty(p, session, asn, re, ctx)
		}
	}
}

func (c *bgpCollector) collectMetricForProperty(property, session, asn string, re *proto.Sentence, ctx *collectorContext) {
	desc := bgpDescriptions[property]
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
