package collector

import (
	"strconv"
	"strings"

	"github.com/nshttpd/mikrotik-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	routeros "gopkg.in/routeros.v2"
	"gopkg.in/routeros.v2/proto"
)

var (
	bgpabelNames    = []string{"name", "address", "session", "asn"}
	bgpProps        = []string{"name", "remote-as", "state"}
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
}

type bgpCollector struct {
}

func (c *bgpCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range bgpDescriptions {
		ch <- d
	}
}

func (c *bgpCollector) collect(ch chan<- prometheus.Metric, device *config.Device, client *routeros.Client) error {
	stats, err := c.fetch(client, device)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, device, ch)
	}

	return nil
}

func (c *bgpCollector) fetch(client *routeros.Client, device *config.Device) ([]*proto.Sentence, error) {
	reply, err := client.Run("/routing/bgp/peer/print", "=.proplist="+strings.Join(bgpProps, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": device.Name,
			"error":  err,
		}).Error("error fetching bgp metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *bgpCollector) collectForStat(re *proto.Sentence, device *config.Device, ch chan<- prometheus.Metric) {
	var session, asn string
	for _, p := range bgpProps {
		if p == "name" {
			session = re.Map[p]
		} else if p == "remote-as" {
			asn = re.Map[p]
		} else {
			c.collectMetricForProperty(p, session, asn, device, re, ch)
		}
	}
}

func (c *bgpCollector) collectMetricForProperty(property, session, asn string, device *config.Device, re *proto.Sentence, ch chan<- prometheus.Metric) {
	desc := bgpDescriptions[property]
	v, err := c.parseValueForProperty(property, re.Map[property])
	if err != nil {
		log.WithFields(log.Fields{
			"device":   device.Name,
			"session":  session,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing bgp metric value")
		return
	}

	ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, device.Name, device.Address, session, asn)
}

func (c *bgpCollector) parseValueForProperty(property, value string) (float64, error) {
	if property == "state" {
		if value == "established" {
			return 1, nil
		}

		return 0, nil
	}

	return strconv.ParseFloat(value, 64)
}
