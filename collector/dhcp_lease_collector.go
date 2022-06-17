package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type dhcpLeaseCollector struct {
	props        []string
	descriptions *prometheus.Desc
}

func (c *dhcpLeaseCollector) init() {
	c.props = []string{"active-mac-address", "server", "status", "expires-after", "active-address", "host-name"}

	labelNames := []string{"name", "address", "activemacaddress", "server", "status", "expiresafter", "activeaddress", "hostname"}
	c.descriptions = description("dhcp", "leases_metrics", "number of metrics", labelNames)

}

func newDHCPLCollector() routerOSCollector {
	c := &dhcpLeaseCollector{}
	c.init()
	return c
}

func (c *dhcpLeaseCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.descriptions
}

func (c *dhcpLeaseCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectMetric(ctx, re)
	}

	return nil
}

func (c *dhcpLeaseCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/ip/dhcp-server/lease/print", "?status=bound", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching DHCP leases metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *dhcpLeaseCollector) collectMetric(ctx *collectorContext, re *proto.Sentence) {
	v := 1.0

	f, err := parseDuration(re.Map["expires-after"])
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"property": "expires-after",
			"value":    re.Map["expires-after"],
			"error":    err,
		}).Error("error parsing duration metric value")
		return
	}

	activemacaddress := re.Map["active-mac-address"]
	server := re.Map["server"]
	status := re.Map["status"]
	activeaddress := re.Map["active-address"]
	// QuoteToASCII because of broken DHCP clients
	hostname := strconv.QuoteToASCII(re.Map["host-name"])

	metric, err := prometheus.NewConstMetric(c.descriptions, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, activemacaddress, server, status, strconv.FormatFloat(f, 'f', 0, 64), activeaddress, hostname)
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error parsing dhcp lease")
		return
	}
	ctx.ch <- metric
}
