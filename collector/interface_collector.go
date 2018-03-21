package collector

import (
	"strconv"
	"strings"

	"github.com/nshttpd/mikrotik-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2"
	"gopkg.in/routeros.v2/proto"
)

var (
	interfaceLabelNames   = []string{"name", "address", "interface"}
	interfaceProps        = []string{"name", "rx-byte", "tx-byte", "rx-packet", "tx-packet", "rx-error", "tx-error", "rx-drop", "tx-drop"}
	interfaceDescriptions map[string]*prometheus.Desc
)

func init() {
	interfaceDescriptions = make(map[string]*prometheus.Desc)
	for _, p := range interfaceProps[1:] {
		interfaceDescriptions[p] = descriptionForPropertyName("interface", p, interfaceLabelNames)
	}
}

type interfaceCollector struct {
}

func (c *interfaceCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range interfaceDescriptions {
		ch <- d
	}
}

func (c *interfaceCollector) collect(ch chan<- prometheus.Metric, device *config.Device, client *routeros.Client) error {
	stats, err := c.fetch(client, device)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, device, ch)
	}

	return nil
}

func (c *interfaceCollector) fetch(client *routeros.Client, device *config.Device) ([]*proto.Sentence, error) {
	reply, err := client.Run("/interface/print", "?disabled=false",
		"?running=true", "=.proplist="+strings.Join(interfaceProps, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": device.Name,
			"error":  err,
		}).Error("error fetching interface metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *interfaceCollector) collectForStat(re *proto.Sentence, device *config.Device, ch chan<- prometheus.Metric) {
	var iface string
	for _, p := range interfaceProps {
		if p == "name" {
			iface = re.Map[p]
		} else {
			c.collectMetricForProperty(p, iface, device, re, ch)
		}
	}
}

func (c *interfaceCollector) collectMetricForProperty(property, iface string, device *config.Device, re *proto.Sentence, ch chan<- prometheus.Metric) {
	desc := interfaceDescriptions[property]
	v, err := strconv.ParseFloat(re.Map[property], 64)
	if err != nil {
		log.WithFields(log.Fields{
			"device":    device.Name,
			"interface": iface,
			"property":  property,
			"value":     re.Map[property],
			"error":     err,
		}).Error("error parsing interface metric value")
		return
	}

	ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, device.Name, device.Address, iface)
}
