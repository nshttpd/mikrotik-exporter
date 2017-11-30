package collector

import (
	"strings"

	"fmt"

	"strconv"

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
	InterfaceProps      = []string{"name", "rx-byte", "tx-byte", "rx-packet", "tx-packet", "rx-error", "tx-error", "rx-drop", "tx-drop"}
	resourceLabelNames  = []string{"name", "address"}
	ResourceProps       = []string{"free-memory", "total-memory", "cpu-load", "free-hdd-space", "total-hdd-space"}
)

type Device struct {
	address  string
	name     string
	user     string
	password string
	iDesc    map[string]*prometheus.Desc // interface level descriptions for device
	rDesc    map[string]*prometheus.Desc // resource level descriptions for device
}

func metricStringCleanup(in string) string {
	return strings.Replace(in, "-", "_", -1)
}

func (d *Device) fetchInterfaceMetrics() ([]*proto.Sentence, error) {

	log.WithFields(log.Fields{
		"device": d.name,
	}).Debug("fetching interface metrics")

	// grab a connection to the device
	c, err := routeros.Dial(d.address+apiPort, d.user, d.password)
	if err != nil {
		log.WithFields(log.Fields{
			"device": d.name,
			"error":  err,
		}).Error("error dialing device")
		return nil, err
	}
	defer c.Close()

	reply, err := c.Run("/interface/print", "?disabled=false",
		"?running=true", "=.proplist="+strings.Join(InterfaceProps, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": d.name,
			"error":  err,
		}).Error("error fetching interface metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (d *Device) Update(ch chan<- prometheus.Metric) error {

	stats, err := d.fetchInterfaceMetrics()
	// if there is no error, deal with the response
	if err == nil {
		for _, re := range stats {
			var intf string
			for _, p := range InterfaceProps {
				if p == "name" {
					intf = re.Map[p]
				} else {
					desc, ok := d.iDesc[p]
					if !ok {
						desc = prometheus.NewDesc(
							prometheus.BuildFQName(namespace, "interface", metricStringCleanup(p)),
							fmt.Sprintf("interface property statistic %s", p),
							interfaceLabelNames,
							nil,
						)
						d.iDesc[p] = desc
					}
					v, err := strconv.ParseFloat(re.Map[p], 64)
					if err == nil {
						ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, d.name, d.address, intf)
					} else {
						log.WithFields(log.Fields{
							"device":    d.name,
							"interface": intf,
							"property":  p,
							"value":     re.Map[p],
							"error":     err,
						}).Error("error parsing interface metric value")
					}
				}
			}
		}
	}
	return nil
}
