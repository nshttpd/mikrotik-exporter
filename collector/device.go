package collector

import (
	"strings"

	"fmt"

	"strconv"

	"github.com/nshttpd/mikrotik-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2"
	"gopkg.in/routeros.v2/proto"
)

const (
	apiPort = ":8728"
)

var (
	interfaceLabelNames = []string{"name", "address", "interface"}
	interfaceProps      = []string{"name", "rx-byte", "tx-byte", "rx-packet", "tx-packet", "rx-error", "tx-error", "rx-drop", "tx-drop"}
	resourceLabelNames  = []string{"name", "address"}
	resourceProps       = []string{"free-memory", "total-memory", "cpu-load", "free-hdd-space", "total-hdd-space"}
)

type device struct {
	config.Device
	interfaceDesc map[string]*prometheus.Desc
	ressourceDesc map[string]*prometheus.Desc
}

func devicesForConfig(cfg *config.Config) []*device {
	devices := make([]*device, len(cfg.Devices))
	for i, d := range cfg.Devices {
		devices[i] = &device{d, make(map[string]*prometheus.Desc), make(map[string]*prometheus.Desc)}
	}

	return devices
}

func metricStringCleanup(in string) string {
	return strings.Replace(in, "-", "_", -1)
}

func (d *device) fetchInterfaceMetrics() ([]*proto.Sentence, error) {
	log.WithFields(log.Fields{
		"device": d.Name,
	}).Debug("fetching interface metrics")

	c, err := routeros.Dial(d.Address+apiPort, d.User, d.Password)
	if err != nil {
		log.WithFields(log.Fields{
			"device": d.Name,
			"error":  err,
		}).Error("error dialing device")
		return nil, err
	}
	defer c.Close()

	reply, err := c.Run("/interface/print", "?disabled=false",
		"?running=true", "=.proplist="+strings.Join(interfaceProps, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": d.Name,
			"error":  err,
		}).Error("error fetching interface metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (d *device) Update(ch chan<- prometheus.Metric) error {
	stats, err := d.fetchInterfaceMetrics()
	if err != nil {
		return err
	}

	for _, re := range stats {
		d.updateForStat(re, ch)
	}

	return nil
}

func (d *device) updateForStat(re *proto.Sentence, ch chan<- prometheus.Metric) {
	var intf string
	for _, p := range interfaceProps {
		if p == "name" {
			intf = re.Map[p]
		} else {
			d.updateWithProperty(p, intf, re, ch)
		}
	}
}

func (d *device) updateWithProperty(property, intf string, re *proto.Sentence, ch chan<- prometheus.Metric) {
	desc := d.descriptionForPropery(property)
	v, err := strconv.ParseFloat(re.Map[property], 64)
	if err != nil {
		log.WithFields(log.Fields{
			"device":    d.Name,
			"interface": intf,
			"property":  property,
			"value":     re.Map[property],
			"error":     err,
		}).Error("error parsing interface metric value")
		return
	}

	ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, d.Name, d.Address, intf)
}

func (d *device) descriptionForPropery(property string) *prometheus.Desc {
	desc, ok := d.interfaceDesc[property]
	if ok {
		return desc
	}

	desc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "interface", metricStringCleanup(property)),
		fmt.Sprintf("interface property statistic %s", property),
		interfaceLabelNames,
		nil,
	)
	d.interfaceDesc[property] = desc
	return desc
}
