package exporter

import (
	"go.uber.org/zap"
	"gopkg.in/routeros.v2"
	"strconv"
	"strings"
)

const (
	apiPort = ":8728"
)

type Device struct {
	Address  string
	Name     string
	User     string
	Password string
}

func (d *Device) fetchInterfaceMetrics(c *routeros.Client, m PromMetrics, l *zap.SugaredLogger) error {
	l.Debugw("fetching interface metrics",
		"device", d.Name,
	)

	reply, err := c.Run("/interface/print", "?disabled=false",
		"?running=true", "=.proplist="+strings.Join(InterfaceProps, ","))
	if err != nil {
		return err
	}

	for _, re := range reply.Re {
		var name string
		// name should always be first element on the array
		for _, p := range InterfaceProps {
			if p == "name" {
				name = re.Map[p]
			} else {
				v, err := strconv.ParseFloat(re.Map[p], 64)
				if err != nil {
					l.Errorw("error parsing value to float",
						"device", d.Name,
						"property", p,
						"value", re.Map[p],
						"error", err,
					)
				}
				m.IncrementInterface(p, d.Name, d.Address, name, v)
			}
		}
	}

	l.Debugw("done fetching interface metrics",
		"device", d.Name,
	)

	return nil
}

func (d *Device) CollectMetrics(p PromMetrics, l *zap.SugaredLogger) error {

	c, err := routeros.Dial(d.Address+apiPort, d.User, d.Password)
	if err != nil {
		l.Errorw("error dialing device",
			"device", d.Name,
			"error", err,
		)
		return err
	}
	defer c.Close()

	if err := d.fetchInterfaceMetrics(c, p, l); err != nil {
		l.Errorw("error fetching interface metrics",
			"device", d.Name,
			"error", err,
		)
		return err
	}

	return nil
}
