package collector

import (
	"strings"

	"fmt"

	"strconv"

	"github.com/prometheus/client_golang/prometheus"
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
	// clean up logging later TODO(smb)
	//l.Debugw("fetching interface metrics",
	//	"device", d.name,
	//)

	// grab a connection to the device
	c, err := routeros.Dial(d.address+apiPort, d.user, d.password)
	if err != nil {
		// clean up logging later TODO(smb)
		//l.Errorw("error dialing device",
		//	"device", d.name,
		//	"error", err,
		//)
		return nil, err
	}
	defer c.Close()

	reply, err := c.Run("/interface/print", "?disabled=false",
		"?running=true", "=.proplist="+strings.Join(InterfaceProps, ","))
	if err != nil {
		// do some logging here about an error when we redo all the logging TODO(smb)
		return nil, err
	}

	return reply.Re, nil

	//for _, re := range reply.Re {
	//	var name string
	//	// name should always be first element on the array
	//	for _, p := range InterfaceProps {
	//		if p == "name" {
	//			name = re.Map[p]
	//		} else {
	//			v, err := strconv.ParseFloat(re.Map[p], 64)
	//			if err != nil {
	//				l.Errorw("error parsing value to float",
	//					"device", d.name,
	//					"property", p,
	//					"value", re.Map[p],
	//					"error", err,
	//				)
	//			}
	//			m.IncrementInterface(p, d.name, d.address, name, v)
	//		}
	//	}
	//}
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
					} // add an else with logging here when logging is re done TODO(smb)
				}
			}
		}
	}
	return nil
}
