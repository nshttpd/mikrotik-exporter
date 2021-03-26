package collector

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type lteCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newLteCollector() routerOSCollector {
	c := &lteCollector{}
	c.init()
	return c
}

func (c *lteCollector) init() {
	c.props = []string{"current-cellid", "primary-band" ,"ca-band", "rssi", "rsrp", "rsrq", "sinr"}
	labelNames := []string{"name", "address", "interface", "cellid", "primaryband", "caband"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props {
		c.descriptions[p] = descriptionForPropertyName("lte_interface", p, labelNames)
	}
}

func (c *lteCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *lteCollector) collect(ctx *collectorContext) error {
	names, err := c.fetchInterfaceNames(ctx)
	if err != nil {
		return err
	}

	for _, n := range names {
		err := c.collectForInterface(n, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *lteCollector) fetchInterfaceNames(ctx *collectorContext) ([]string, error) {
	reply, err := ctx.client.Run("/interface/lte/print", "?disabled=false", "=.proplist=name")
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching lte interface names")
		return nil, err
	}

	names := []string{}
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *lteCollector) collectForInterface(iface string, ctx *collectorContext) error {
	reply, err := ctx.client.Run("/interface/lte/info", fmt.Sprintf("=number=%s", iface), "=once=", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"interface": iface,
			"device":    ctx.device.Name,
			"error":     err,
		}).Error("error fetching interface statistics")
		return err
	}

	for _, p := range c.props[3:] {
		// there's always going to be only one sentence in reply, as we
		// have to explicitly specify the interface
		c.collectMetricForProperty(p, iface, reply.Re[0], ctx)
	}

	return nil
}

func (c *lteCollector) collectMetricForProperty(property, iface string, re *proto.Sentence, ctx *collectorContext) {
	desc := c.descriptions[property]
	current_cellid := re.Map["current-cellid"]
	// get only band and its width, drop earfcn and phy-cellid info
	primaryband := re.Map["primary-band"]
	if primaryband != "" {
		primaryband = strings.Fields(primaryband)[0]
	}
	caband := re.Map["ca-band"]
	if caband != "" {
		caband = strings.Fields(caband)[0]
	}

	if re.Map[property] == "" {
		return
	}
	v, err := strconv.ParseFloat(re.Map[property], 64)
	if err != nil {
		log.WithFields(log.Fields{
			"property":  property,
			"interface": iface,
			"device":    ctx.device.Name,
			"error":     err,
		}).Error("error parsing interface metric value")
		return
	}

	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, iface, current_cellid, primaryband, caband)
}
