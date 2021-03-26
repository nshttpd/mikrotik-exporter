package collector

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type firmwareCollector struct {
	props       []string
	description *prometheus.Desc
}

func newFirmwareCollector() routerOSCollector {
	c := &firmwareCollector{}
	c.init()
	return c
}

func (c *firmwareCollector) init() {
	labelNames := []string{"devicename", "name", "disabled", "version", "build_time"}
	c.description = description("system", "package", "system packages version", labelNames)
}

func (c *firmwareCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.description
}

func (c *firmwareCollector) collect(ctx *collectorContext) error {
	reply, err := ctx.client.Run("/system/package/getall")
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		})
		return err
	}

	pkgs := reply.Re

	for _, pkg := range pkgs {
		v := 1.0
		if strings.EqualFold(pkg.Map["disabled"], "true") {
			v = 0.0
		}
		ctx.ch <- prometheus.MustNewConstMetric(c.description, prometheus.GaugeValue, v, ctx.device.Name, pkg.Map["name"], pkg.Map["disabled"], pkg.Map["version"], pkg.Map["build-time"])
	}

	return nil
}
