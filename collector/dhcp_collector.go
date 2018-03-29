package collector

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const dhcpPrefiix = "dhcp"

var (
	dhcpLeasesActiveCountDesc *prometheus.Desc
)

func init() {
	l := []string{"name", "address", "server"}
	dhcpLeasesActiveCountDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, dhcpPrefiix, "leases_active_count"),
		"number of active leases per DHCP server",
		l,
		nil,
	)
}

type dhcpCollector struct {
}

func (c *dhcpCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- dhcpLeasesActiveCountDesc
}

func (c *dhcpCollector) collect(ctx *collectorContext) error {
	names, err := c.fetchDHCPServerNames(ctx)
	if err != nil {
		return err
	}

	for _, n := range names {
		err := c.colllectForDHCPServer(ctx, n)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *dhcpCollector) fetchDHCPServerNames(ctx *collectorContext) ([]string, error) {
	reply, err := ctx.client.Run("/ip/dhcp-server/print", "=.proplist=name")
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching DHCP server names")
		return nil, err
	}

	names := []string{}
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *dhcpCollector) colllectForDHCPServer(ctx *collectorContext, dhcpServer string) error {
	reply, err := ctx.client.Run("/ip/dhcp-server/lease/print", fmt.Sprintf("?server=%s", dhcpServer), "=active=", "=count-only=")
	if err != nil {
		log.WithFields(log.Fields{
			"dhcp_server": dhcpServer,
			"device":      ctx.device.Name,
			"error":       err,
		}).Error("error fetching DHCP lease counts")
		return err
	}

	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		log.WithFields(log.Fields{
			"dhcp_server": dhcpServer,
			"device":      ctx.device.Name,
			"error":       err,
		}).Error("error parsing DHCP lease counts")
		return err
	}

	ctx.ch <- prometheus.MustNewConstMetric(dhcpLeasesActiveCountDesc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, dhcpServer)
	return nil
}
