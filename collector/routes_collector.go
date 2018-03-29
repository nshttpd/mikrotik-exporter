package collector

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
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

func (c *routesCollector) collect(ctx *collectorContext) error {
	c.colllectForIPVersion(ctx, "4", "ip")
	c.colllectForIPVersion(ctx, "6", "ipv6")
	return nil
}

func (c *routesCollector) colllectForIPVersion(ctx *collectorContext, ipVersion, topic string) {
	c.colllectCount(ctx, ipVersion, topic)

	for _, p := range routesProtocols {
		c.colllectCountProtcol(ctx, ipVersion, topic, p)
	}
}

func (c *routesCollector) colllectCount(ctx *collectorContext, ipVersion, topic string) {
	reply, err := ctx.client.Run(fmt.Sprintf("/%s/route/print", topic), "?disabled=false", "=count-only=")
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"device":     ctx.device.Name,
			"error":      err,
		}).Error("error fetching routes metrics")
		return
	}

	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"device":     ctx.device.Name,
			"error":      err,
		}).Error("error parsing routes metrics")
		return
	}

	ctx.ch <- prometheus.MustNewConstMetric(routesTotalDesc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, ipVersion)
}

func (c *routesCollector) colllectCountProtcol(ctx *collectorContext, ipVersion, topic, protocol string) {
	reply, err := ctx.client.Run(fmt.Sprintf("/%s/route/print", topic), "?disabled=false", fmt.Sprintf("?%s", protocol), "=count-only=")
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"protocol":   protocol,
			"device":     ctx.device.Name,
			"error":      err,
		}).Error("error fetching routes metrics")
		return
	}

	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"protocol":   protocol,
			"device":     ctx.device.Name,
			"error":      err,
		}).Error("error parsing routes metrics")
		return
	}

	ctx.ch <- prometheus.MustNewConstMetric(routesProtocolDesc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, ipVersion, protocol)
}
