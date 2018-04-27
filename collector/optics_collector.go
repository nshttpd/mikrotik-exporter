package collector

import "github.com/prometheus/client_golang/prometheus"

type opticsCollector struct {
	rxStatusDesc    *prometheus.Desc
	txStatusDesc    *prometheus.Desc
	rxPowerDesc     *prometheus.Desc
	txPowerDesc     *prometheus.Desc
	temperatureDesc *prometheus.Desc
	txBiasDesc      *prometheus.Desc
	voltageDesc     *prometheus.Desc
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
	return nil
}
