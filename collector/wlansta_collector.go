package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type wlanSTACollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newWlanSTACollector() routerOSCollector {
	c := &wlanSTACollector{}
	c.init()
	return c
}

func (c *wlanSTACollector) init() {
	c.props = []string{"interface", "mac-address", "signal-to-noise", "signal-strength-ch0", "packets", "bytes", "frames"}
	labelNames := []string{"name", "address", "interface", "mac_address"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[:len(c.props)-3] {
		c.descriptions[p] = descriptionForPropertyName("wlan_station", p, labelNames)
	}
	for _, p := range c.props[len(c.props)-3:] {
		c.descriptions["tx_"+p] = descriptionForPropertyName("wlan_station", "tx_"+p, labelNames)
		c.descriptions["rx_"+p] = descriptionForPropertyName("wlan_station", "rx_"+p, labelNames)
	}
}

func (c *wlanSTACollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *wlanSTACollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *wlanSTACollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/interface/wireless/registration-table/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching wlan station metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *wlanSTACollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	iface := re.Map["interface"]
	mac := re.Map["mac-address"]

	for _, p := range c.props[2 : len(c.props)-3] {
		c.collectMetricForProperty(p, iface, mac, re, ctx)
	}
	for _, p := range c.props[len(c.props)-3:] {
		c.collectMetricForTXRXCounters(p, iface, mac, re, ctx)
	}
}

func (c *wlanSTACollector) collectMetricForProperty(property, iface, mac string, re *proto.Sentence, ctx *collectorContext) {
	if re.Map[property] == "" {
		return
	}
	v, err := strconv.ParseFloat(re.Map[property], 64)
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing wlan station metric value")
		return
	}

	desc := c.descriptions[property]
	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, iface, mac)
}

func (c *wlanSTACollector) collectMetricForTXRXCounters(property, iface, mac string, re *proto.Sentence, ctx *collectorContext) {
	tx, rx, err := splitStringToFloats(re.Map[property])
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing wlan station metric value")
		return
	}
	desc_tx := c.descriptions["tx_"+property]
	desc_rx := c.descriptions["rx_"+property]
	ctx.ch <- prometheus.MustNewConstMetric(desc_tx, prometheus.CounterValue, tx, ctx.device.Name, ctx.device.Address, iface, mac)
	ctx.ch <- prometheus.MustNewConstMetric(desc_rx, prometheus.CounterValue, rx, ctx.device.Name, ctx.device.Address, iface, mac)
}
