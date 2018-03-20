package collector

import (
	"fmt"
	"strconv"

	"github.com/nshttpd/mikrotik-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2"
)

const routesPrefiix = "routes"

var (
	routesProtocols = []string{"bgp", "static", "ospf", "dynamic", "connect"}
)

var (
	routesTotalDesc    *prometheus.Desc
	routesProtocolDesc *prometheus.Desc
)

func init() {
	l := []string{"name", "address", "ip_version"}
	routesTotalDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, routesPrefiix, "total_count"),
		"number of routes in RIB",
		l,
		nil,
	)
	routesProtocolDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, routesPrefiix, "protocol_count"),
		"number of routes per protocol in RIB",
		append(l, "protocol"),
		nil,
	)
}

type routesCollector struct {
}

func (c *routesCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- routesTotalDesc
	ch <- routesProtocolDesc
}

func (c *routesCollector) collect(ch chan<- prometheus.Metric, device *config.Device, client *routeros.Client) error {
	c.colllectForIPVersion(client, device, ch, "4", "ip")
	c.colllectForIPVersion(client, device, ch, "6", "ipv6")
	return nil
}

func (c *routesCollector) colllectForIPVersion(client *routeros.Client, device *config.Device, ch chan<- prometheus.Metric, ipVersion, topic string) {
	c.colllectCount(client, device, ch, ipVersion, topic)

	for _, p := range routesProtocols {
		c.colllectCountProtcol(client, device, ch, ipVersion, topic, p)
	}
}

func (c *routesCollector) colllectCount(client *routeros.Client, device *config.Device, ch chan<- prometheus.Metric, ipVersion, topic string) {
	reply, err := client.Run(fmt.Sprintf("/%s/route/print", topic), "?disabled=false", "=count-only=")
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"device":     device.Name,
			"error":      err,
		}).Error("error fetching routes metrics")
		return
	}

	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"device":     device.Name,
			"error":      err,
		}).Error("error parsing routes metrics")
		return
	}

	ch <- prometheus.MustNewConstMetric(routesTotalDesc, prometheus.GaugeValue, v, device.Name, device.Address, ipVersion)
}

func (c *routesCollector) colllectCountProtcol(client *routeros.Client, device *config.Device, ch chan<- prometheus.Metric, ipVersion, topic, protocol string) {
	reply, err := client.Run(fmt.Sprintf("/%s/route/print", topic), "?disabled=false", fmt.Sprintf("?%s", protocol), "=count-only=")
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"protocol":   protocol,
			"device":     device.Name,
			"error":      err,
		}).Error("error fetching routes metrics")
		return
	}

	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"protocol":   protocol,
			"device":     device.Name,
			"error":      err,
		}).Error("error parsing routes metrics")
		return
	}

	ch <- prometheus.MustNewConstMetric(routesProtocolDesc, prometheus.GaugeValue, v, device.Name, device.Address, ipVersion, protocol)
}
