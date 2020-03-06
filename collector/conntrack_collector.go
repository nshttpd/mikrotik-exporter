package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type conntrackCollector struct {
	props            []string
	totalEntriesDesc *prometheus.Desc
	maxEntriesDesc   *prometheus.Desc
}

func newConntrackCollector() routerOSCollector {
	const prefix = "conntrack"

	labelNames := []string{"name", "address"}
	return &conntrackCollector{
		props:            []string{"total-entries", "max-entries"},
		totalEntriesDesc: description(prefix, "entries", "Number of tracked connections", labelNames),
		maxEntriesDesc:   description(prefix, "max_entries", "Conntrack table capacity", labelNames),
	}
}

func (c *conntrackCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalEntriesDesc
	ch <- c.maxEntriesDesc
}

func (c *conntrackCollector) collect(ctx *collectorContext) error {
	reply, err := ctx.client.Run("/ip/firewall/connection/tracking/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching conntrack table metrics")
		return err
	}

	for _, re := range reply.Re {
		c.collectMetricForProperty("total-entries", c.totalEntriesDesc, re, ctx)
		c.collectMetricForProperty("max-entries", c.maxEntriesDesc, re, ctx)
	}

	return nil
}

func (c *conntrackCollector) collectMetricForProperty(property string, desc *prometheus.Desc, re *proto.Sentence, ctx *collectorContext) {
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
		}).Error("error parsing conntrack metric value")
		return
	}

	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address)
}
