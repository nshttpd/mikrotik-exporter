package collector

import (
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
	c.props = []string{"local-address", "remote-address", "state", "side", "uptime"}

	labelNames := []string{"devicename", "local_address", "remote_address", "state", "side"}
	c.descriptions = make(map[string]*prometheus.Desc)
	c.descriptions["uptime"] = descriptionForPropertyName("ipsec_peers", "uptime", labelNames)
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
		c.collectMetric(re, ctx)
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

func (c *ipsecPeersCollector) collectMetric(re *proto.Sentence, ctx *collectorContext) {
	v, err := parseDuration(re.Map["uptime"])
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"property": "uptime",
			"value":    re.Map["uptime"],
			"error":    err,
		}).Error("error parsing duration metric value")
		return
	}

	localAddress := re.Map["local-address"]
	remoteAddress := re.Map["remote-address"]
	state := re.Map["state"]
	side := re.Map["side"]

	ctx.ch <- prometheus.MustNewConstMetric(c.descriptions["uptime"], prometheus.CounterValue, v, ctx.device.Name, localAddress, remoteAddress, state, side)
}
