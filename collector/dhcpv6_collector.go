package collector

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const dhcpv6Prefiix = "dhcpv6"

var (
	dhcpv6bindingCountDesc *prometheus.Desc
)

func init() {
	l := []string{"name", "address", "server"}
	dhcpv6bindingCountDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, dhcpv6Prefiix, "binding_count"),
		"number of active bindings per DHCPv6 server",
		l,
		nil,
	)
}

type dhcpv6Collector struct {
}

func (c *dhcpv6Collector) describe(ch chan<- *prometheus.Desc) {
	ch <- dhcpv6bindingCountDesc
}

func (c *dhcpv6Collector) collect(ctx *collectorContext) error {
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

func (c *dhcpv6Collector) fetchDHCPServerNames(ctx *collectorContext) ([]string, error) {
	reply, err := ctx.client.Run("/ipv6/dhcp-server/print", "=.proplist=name")
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching DHCPv6 server names")
		return nil, err
	}

	names := []string{}
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *dhcpv6Collector) colllectForDHCPServer(ctx *collectorContext, dhcpServer string) error {
	reply, err := ctx.client.Run("/ipv6/dhcp-server/binding/print", fmt.Sprintf("?server=%s", dhcpServer), "=count-only=")
	if err != nil {
		log.WithFields(log.Fields{
			"dhcpv6_server": dhcpServer,
			"device":        ctx.device.Name,
			"error":         err,
		}).Error("error fetching DHCPv6 binding counts")
		return err
	}

	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		log.WithFields(log.Fields{
			"dhcpv6_server": dhcpServer,
			"device":        ctx.device.Name,
			"error":         err,
		}).Error("error parsing DHCPv6 binding counts")
		return err
	}

	ctx.ch <- prometheus.MustNewConstMetric(dhcpv6bindingCountDesc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, dhcpServer)
	return nil
}
