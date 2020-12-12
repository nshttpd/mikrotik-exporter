package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type ipsecPeersCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newIPsecPeersCollector() routerOSCollector {
	c := &ipsecPeersCollector{}
	c.init()
	return c
}

func (c *ipsecPeersCollector) init() {
	c.props = []string{"local-address", "remote-address", "state", "side", "uptime", "rx-bytes", "rx-packets", "tx-bytes", "tx-packets"}

	labelNames := []string{"devicename", "local_address", "remote_address", "state", "side"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[4:] {
		c.descriptions[p] = descriptionForPropertyName("ipsec_peers", p, labelNames)
	}
}

func (c *ipsecPeersCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *ipsecPeersCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectMetrics(re, ctx)
	}

	return nil
}

func (c *ipsecPeersCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/ip/ipsec/active-peers/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching interface metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *ipsecPeersCollector) collectMetrics(re *proto.Sentence, ctx *collectorContext) {
	for _, p := range c.props[4:] {
		c.collectMetricForProperty(p, re, ctx)
	}
}

func (c *ipsecPeersCollector) collectMetricForProperty(property string, re *proto.Sentence, ctx *collectorContext) {
	desc := c.descriptions[property]
	if value := re.Map[property]; value != "" {
		var v float64
		var err error
		switch property {
		case "uptime":
			v, err = parseDuration(re.Map["uptime"])
			if err != nil {
				log.WithFields(log.Fields{
					"device":   ctx.device.Name,
					"property": property,
					"value":    value,
					"error":    err,
				}).Error("error parsing ipsec peers metric value")
				return
			}
		default:
			v, err = strconv.ParseFloat(value, 64)
			if err != nil {
				log.WithFields(log.Fields{
					"device":   ctx.device.Name,
					"property": property,
					"value":    value,
					"error":    err,
				}).Error("error parsing ipsec peers metric value")
				return
			}
		}

		ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, ctx.device.Name, re.Map["local-address"], re.Map["remote-address"], re.Map["state"], re.Map["side"])
	}
}
