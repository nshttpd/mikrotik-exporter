package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

// from https://forum.mikrotik.com/viewtopic.php?t=195124#p999722:
// wifiwave2 is an implementation of drivers from the manufacturer of the
// chipset, rather than an in-house written driver (which wireless is). So
// there are many small details that are missing or incomplete...

type wlanSTACollector struct {
	// Both wifiwave2 and wireless have a similar, yet different API. They also
	// expose a slightly different set of properties.
	props               []string
	propsWirelessExtra  []string
	propsWirelessRXTX   []string
	propsWifiwave2Extra []string
	propsWifiwave2RXTX  []string
	descriptions        map[string]*prometheus.Desc
}

func newWlanSTACollector() routerOSCollector {
	c := &wlanSTACollector{}
	c.init()
	return c
}

func (c *wlanSTACollector) init() {
	// common properties
	c.props = []string{"interface", "mac-address"}
	// wifiwave2 doesn't expose SNR, and uses different name for signal-strength
	c.propsWirelessExtra = []string{"signal-to-noise", "signal-strength"}
	// wireless exposes extra field "frames", not available in wifiwave2
	c.propsWirelessRXTX = []string{"packets", "bytes", "frames"}
	c.propsWifiwave2Extra = []string{"signal"}
	c.propsWifiwave2RXTX = []string{"packets", "bytes"}
	// all metrics have the same label names
	labelNames := []string{"name", "address", "interface", "mac_address"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.propsWirelessExtra {
		c.descriptions[p] = descriptionForPropertyName("wlan_station", p, labelNames)
	}
	// normalize the metric name 'signal-strength' for the property "signal", so that dashboards
	// that capture both wireless and wifiwave2 devices don't need to normalize
	c.descriptions["signal"] = descriptionForPropertyName("wlan_station", "signal-strength", labelNames)
	for _, p := range c.propsWirelessRXTX {
		c.descriptions["tx_"+p] = descriptionForPropertyName("wlan_station", "tx_"+p, labelNames)
		c.descriptions["rx_"+p] = descriptionForPropertyName("wlan_station", "rx_"+p, labelNames)
	}
	for _, p := range c.propsWifiwave2RXTX {
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
	var cmd []string
	var props []string = c.props
	if ctx.device.Wifiwave2 {
		props = append(props, c.propsWifiwave2Extra...)
		props = append(props, c.propsWifiwave2RXTX...)
		cmd = []string{
			"/interface/wifiwave2/registration-table/print",
			"=.proplist=" + strings.Join(props, ","),
		}
	} else {
		props = append(props, c.propsWirelessExtra...)
		props = append(props, c.propsWirelessRXTX...)
		cmd = []string{
			"/interface/wireless/registration-table/print",
			"=.proplist=" + strings.Join(props, ","),
		}
	}
	reply, err := ctx.client.Run(cmd...)
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

	if ctx.device.Wifiwave2 {
		for _, p := range c.propsWifiwave2Extra {
			c.collectMetricForProperty(p, iface, mac, re, ctx)
		}
		for _, p := range c.propsWifiwave2RXTX {
			c.collectMetricForTXRXCounters(p, iface, mac, re, ctx)
		}
	} else {
		for _, p := range c.propsWirelessExtra {
			c.collectMetricForProperty(p, iface, mac, re, ctx)
		}
		for _, p := range c.propsWirelessRXTX {
			c.collectMetricForTXRXCounters(p, iface, mac, re, ctx)
		}
	}
}

func (c *wlanSTACollector) collectMetricForProperty(property, iface, mac string, re *proto.Sentence, ctx *collectorContext) {
	if re.Map[property] == "" {
		return
	}
	p := re.Map[property]
	i := strings.Index(p, "@")
	if i > -1 {
		p = p[:i]
	}
	v, err := strconv.ParseFloat(p, 64)
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
