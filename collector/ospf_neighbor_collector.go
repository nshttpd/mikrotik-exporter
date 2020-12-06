package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type ospfNeighborCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newOSPFNeighborCollector() routerOSCollector {
	c := &ospfNeighborCollector{}
	c.init()
	return c
}

func (c *ospfNeighborCollector) init() {
	c.props = []string{"instance", "router-id", "address", "interface", "state", "state-changes"}

	const prefix = "ospf_neighbor"
	labelNames := []string{"name", "address", "instance", "router_id", "neighbor_address", "interface", "state"}

	c.descriptions = make(map[string]*prometheus.Desc)
	c.descriptions["state-changes"] = description(prefix, "state_changes", "OSPF neighbor state changes counter", labelNames)
}

func (c *ospfNeighborCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *ospfNeighborCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *ospfNeighborCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/routing/ospf/neighbor/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching ospf neighbor metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *ospfNeighborCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	instance := re.Map["instance"]
	routerID := re.Map["router-id"]
	neighborAddress := re.Map["address"]
	neighborInterface := re.Map["interface"]
	state := re.Map["state"]

	for _, p := range c.props[5:] {
		c.collectMetricForProperty(p, instance, routerID, neighborAddress, neighborInterface, state, re, ctx)
	}
}

func (c *ospfNeighborCollector) collectMetricForProperty(property, instance, routerID, neighborAddress, neighborInterface, state string, re *proto.Sentence, ctx *collectorContext) {
	desc := c.descriptions[property]
	v, err := c.parseValueForProperty(property, re.Map[property])
	if err != nil {
		log.WithFields(log.Fields{
			"device":    ctx.device.Name,
			"router_id": routerID,
			"property":  property,
			"value":     re.Map[property],
			"error":     err,
		}).Error("error parsing ospf neighbor metric value")
		return
	}

	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, instance, routerID, neighborAddress, neighborInterface, state)
}

func (c *ospfNeighborCollector) parseValueForProperty(property, value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}
