package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type capsmanCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newCapsmanCollector() routerOSCollector {
	c := &capsmanCollector{}
	c.init()
	return c
}

func (c *capsmanCollector) init() {
	//"rx-signal", "tx-signal",
	c.props = []string{"interface", "mac-address", "ssid", "uptime", "tx-signal", "rx-signal", "packets", "bytes"}
	labelNames := []string{"name", "address", "interface", "mac_address", "ssid"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[3 : len(c.props)-2] {
		c.descriptions[p] = descriptionForPropertyName("capsman_station", p, labelNames)
	}
	for _, p := range c.props[len(c.props)-2:] {
		c.descriptions["tx_"+p] = descriptionForPropertyName("capsman_station", "tx_"+p, labelNames)
		c.descriptions["rx_"+p] = descriptionForPropertyName("capsman_station", "rx_"+p, labelNames)
	}
}

func (c *capsmanCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *capsmanCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *capsmanCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/caps-man/registration-table/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching wlan station metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *capsmanCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	iface := re.Map["interface"]
	mac := re.Map["mac-address"]
	ssid := re.Map["ssid"]

	for _, p := range c.props[3 : len(c.props)-2] {
		c.collectMetricForProperty(p, iface, mac, ssid, re, ctx)
	}
	for _, p := range c.props[len(c.props)-2:] {
		c.collectMetricForTXRXCounters(p, iface, mac, ssid, re, ctx)
	}
}

func (c *capsmanCollector) collectMetricForProperty(property, iface, mac, ssid string, re *proto.Sentence, ctx *collectorContext) {
	if re.Map[property] == "" {
		return
	}
	p := re.Map[property]
	i := strings.Index(p, "@")
	if i > -1 {
		p = p[:i]
	}
	var v float64
	var err error
	if property != "uptime" {
		v, err = strconv.ParseFloat(p, 64)
	} else {
		v, err = parseDuration(p)
	}
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing capsman station metric value")
		return
	}

	desc := c.descriptions[property]
	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, iface, mac, ssid)
}

func (c *capsmanCollector) collectMetricForTXRXCounters(property, iface, mac, ssid string, re *proto.Sentence, ctx *collectorContext) {
	tx, rx, err := splitStringToFloats(re.Map[property])
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing capsman station metric value")
		return
	}
	desc_tx := c.descriptions["tx_"+property]
	desc_rx := c.descriptions["rx_"+property]
	ctx.ch <- prometheus.MustNewConstMetric(desc_tx, prometheus.CounterValue, tx, ctx.device.Name, ctx.device.Address, iface, mac, ssid)
	ctx.ch <- prometheus.MustNewConstMetric(desc_rx, prometheus.CounterValue, rx, ctx.device.Name, ctx.device.Address, iface, mac, ssid)
}
