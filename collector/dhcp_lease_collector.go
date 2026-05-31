package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type dhcpLeaseCollector struct {
	props              []string
	leaseInfoDesc      *prometheus.Desc
	leaseExpiresDesc   *prometheus.Desc
}

func (c *dhcpLeaseCollector) init() {
	c.props = []string{"active-mac-address", "server", "status", "expires-after", "active-address", "host-name"}

	const prefix = "dhcp"

	// Info metric with all lease details as labels
	infoLabelNames := []string{"name", "address", "mac", "ip", "hostname", "server", "status"}
	c.leaseInfoDesc = description(prefix, "lease_info", "DHCP lease information (value is always 1)", infoLabelNames)

	// Separate metric for lease expiration time
	expiresLabelNames := []string{"name", "address", "mac", "ip"}
	c.leaseExpiresDesc = description(prefix, "lease_expires_seconds", "Seconds until DHCP lease expires", expiresLabelNames)
}

func newDHCPLCollector() routerOSCollector {
	c := &dhcpLeaseCollector{}
	c.init()
	return c
}

func (c *dhcpLeaseCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.leaseInfoDesc
	ch <- c.leaseExpiresDesc
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
		// Handle empty response as "no active leases" (not an error)
		if isEmptyResponse(err) {
			log.WithFields(log.Fields{
				"device": ctx.device.Name,
			}).Debug("no active DHCP leases")
			return []*proto.Sentence{}, nil
		}
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching DHCP leases metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *dhcpLeaseCollector) collectMetric(ctx *collectorContext, re *proto.Sentence) {
	mac := re.Map["active-mac-address"]
	server := re.Map["server"]
	status := re.Map["status"]
	ip := re.Map["active-address"]
	// QuoteToASCII because of broken DHCP clients, then remove surrounding quotes
	hostname := strconv.QuoteToASCII(re.Map["host-name"])
	hostname = strings.Trim(hostname, "\"")

	// Parse lease expiration time
	expiresSeconds, err := parseDuration(re.Map["expires-after"])
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"property": "expires-after",
			"value":    re.Map["expires-after"],
			"error":    err,
		}).Warn("error parsing lease expiration, using 0")
		expiresSeconds = 0
	}

	// Emit info metric (value=1, labels contain all info)
	infoMetric, err := prometheus.NewConstMetric(
		c.leaseInfoDesc,
		prometheus.GaugeValue,
		1,
		ctx.device.Name, ctx.device.Address, mac, ip, hostname, server, status,
	)
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"mac":    mac,
			"error":  err,
		}).Error("error creating dhcp lease info metric")
	} else {
		ctx.ch <- infoMetric
	}

	// Emit expires metric (value=seconds until expiration)
	expiresMetric, err := prometheus.NewConstMetric(
		c.leaseExpiresDesc,
		prometheus.GaugeValue,
		expiresSeconds,
		ctx.device.Name, ctx.device.Address, mac, ip,
	)
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"mac":    mac,
			"error":  err,
		}).Error("error creating dhcp lease expires metric")
	} else {
		ctx.ch <- expiresMetric
	}
}
