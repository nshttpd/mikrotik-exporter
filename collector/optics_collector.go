package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type opticsCollector struct {
	rxStatusDesc    *prometheus.Desc
	txStatusDesc    *prometheus.Desc
	rxPowerDesc     *prometheus.Desc
	txPowerDesc     *prometheus.Desc
	temperatureDesc *prometheus.Desc
	txBiasDesc      *prometheus.Desc
	voltageDesc     *prometheus.Desc
	props           []string
}

func newOpticsCollector() routerOSCollector {
	const prefix = "optics"

	labelNames := []string{"name", "address", "interface"}
	return &opticsCollector{
		rxStatusDesc:    description(prefix, "rx_status", "RX status (1 = no loss)", labelNames),
		txStatusDesc:    description(prefix, "tx_status", "TX status (1 = no faults)", labelNames),
		rxPowerDesc:     description(prefix, "rx_power_dbm", "RX power in dBM", labelNames),
		txPowerDesc:     description(prefix, "tx_power_dbm", "TX power in dBM", labelNames),
		temperatureDesc: description(prefix, "temperature_celsius", "temperature in degree celsius", labelNames),
		txBiasDesc:      description(prefix, "tx_bias_ma", "bias is milliamps", labelNames),
		voltageDesc:     description(prefix, "voltage_volt", "volage in volt", labelNames),
		props:           []string{"sfp-rx-loss", "sfp-tx-fault", "sfp-temperature", "sfp-supply-voltage", "sfp-tx-bias-current", "sfp-tx-power", "sfp-rx-power"},
	}
}

func (c *opticsCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.rxStatusDesc
	ch <- c.txStatusDesc
	ch <- c.rxPowerDesc
	ch <- c.txPowerDesc
	ch <- c.temperatureDesc
	ch <- c.txBiasDesc
	ch <- c.voltageDesc
}

func (c *opticsCollector) collect(ctx *collectorContext) error {
	reply, err := ctx.client.Run("/interface/ethernet/print", "=.proplist=name")
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching interface metrics")
		return err
	}

	ifaces := make([]string, 0)
	for _, iface := range reply.Re {
		n := iface.Map["name"]
		if strings.HasPrefix(n, "sfp") {
			ifaces = append(ifaces, n)
		}
	}

	if len(ifaces) == 0 {
		return nil
	}

	return c.collectOpticalMetricsForInterfaces(ifaces, ctx)
}

func (c *opticsCollector) collectOpticalMetricsForInterfaces(ifaces []string, ctx *collectorContext) error {
	reply, err := ctx.client.Run("/interface/ethernet/monitor",
		"=numbers="+strings.Join(ifaces, ","),
		"=once=",
		"=.proplist=name,"+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching interface monitor metrics")
		return err
	}

	for _, se := range reply.Re {
		name, ok := se.Map["name"]
		if !ok {
			continue
		}

		c.collectMetricsForInterface(name, se, ctx)
	}

	return nil
}

func (c *opticsCollector) collectMetricsForInterface(name string, se *proto.Sentence, ctx *collectorContext) {
	for _, prop := range c.props {
		v, ok := se.Map[prop]
		if !ok {
			continue
		}

		value, err := c.valueForKey(prop, v)
		if err != nil {
			log.WithFields(log.Fields{
				"device":    ctx.device.Name,
				"interface": name,
				"property":  prop,
				"error":     err,
			}).Error("error parsing interface monitor metric")
			return
		}

		ctx.ch <- prometheus.MustNewConstMetric(c.descForKey(prop), prometheus.GaugeValue, value, ctx.device.Name, ctx.device.Address, name)
	}
}

func (c *opticsCollector) valueForKey(name, value string) (float64, error) {
	if name == "sfp-rx-loss" || name == "sfp-tx-fault" {
		status := float64(1)
		if value == "true" {
			status = float64(0)
		}

		return status, nil
	}

	return strconv.ParseFloat(value, 64)
}

func (c *opticsCollector) descForKey(name string) *prometheus.Desc {
	switch name {
	case "sfp-rx-loss":
		return c.rxStatusDesc
	case "sfp-tx-fault":
		return c.txStatusDesc
	case "sfp-temperature":
		return c.temperatureDesc
	case "sfp-supply-voltage":
		return c.voltageDesc
	case "sfp-tx-bias-current":
		return c.txBiasDesc
	case "sfp-tx-power":
		return c.txPowerDesc
	case "sfp-rx-power":
		return c.rxPowerDesc
	}

	return nil
}
