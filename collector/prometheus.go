package exporter

import (
	"strings"

	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
)

const (
	promNamespace = "mikrotik"
)

var (
	interfaceLabelNames = []string{"name", "address", "interface"}
	InterfaceProps      = []string{"name", "rx-byte", "tx-byte", "rx-packet", "tx-packet", "rx-error", "tx-error", "rx-drop", "tx-drop"}
	resourceLabelNames  = []string{"name", "address"}
	ResourceProps       = []string{"free-memory", "total-memory", "cpu-load", "free-hdd-space", "total-hdd-space"}
)

type PromMetrics struct {
	InterfaceMetrics map[string]*prometheus.CounterVec
	ResourceMetrics  map[string]*prometheus.GaugeVec
}

func metricStringCleanup(in string) string {
	return strings.Replace(in, "-", "_", -1)
}

func (p *PromMetrics) makeLabels(name, address string) prometheus.Labels {
	labels := make(prometheus.Labels)
	labels["name"] = metricStringCleanup(name)
	labels["address"] = metricStringCleanup(address)
	return labels
}

func (p *PromMetrics) makeInterfaceLabels(name, address, intf string) prometheus.Labels {
	l := p.makeLabels(name, address)
	l["interface"] = intf
	return l
}

func (p *PromMetrics) SetupPrometheus(l zap.SugaredLogger) (http.Handler, error) {

	p.InterfaceMetrics = make(map[string]*prometheus.CounterVec)
	p.ResourceMetrics = make(map[string]*prometheus.GaugeVec)

	for _, v := range InterfaceProps {
		n := metricStringCleanup(v)
		c := prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: promNamespace,
			Subsystem: "interface",
			Name:      n,
			Help:      fmt.Sprintf("Interface %s counter", v),
		}, interfaceLabelNames)

		if err := prometheus.Register(c); err != nil {
			l.Errorw("error creating interface counter vector",
				"property", v,
				"error", err,
			)
			return nil, err
		}

		p.InterfaceMetrics[v] = c

	}

	for _, v := range ResourceProps {
		n := metricStringCleanup(v)
		c := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: promNamespace,
			Subsystem: "resource",
			Name:      n,
			Help:      fmt.Sprintf("Resource %s counter", v),
		}, resourceLabelNames)

		if err := prometheus.Register(c); err != nil {
			l.Errorw("error creating resource counter vec",
				"property", v,
				"error", err,
			)
			return nil, err
		}
		p.ResourceMetrics[v] = c
	}

	return promhttp.Handler(), nil

}

func (p *PromMetrics) IncrementInterface(prop, name, address, intf string, cnt float64) {
	l := p.makeInterfaceLabels(name, address, intf)
	p.InterfaceMetrics[prop].With(l).Add(cnt)
}

func (p *PromMetrics) UpdateResource(res, name, address string, v float64) {
	l := p.makeLabels(name, address)
	p.ResourceMetrics[res].With(l).Set(v)
}
