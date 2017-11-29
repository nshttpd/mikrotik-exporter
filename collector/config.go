package collector

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

type Config struct {
	Devices []Device
	Metrics PromMetrics
}

func (c *Config) FromFlags(device, address, user, password *string) error {
	if *device == "" || *address == "" || *user == "" || *password == "" {
		return fmt.Errorf("missing required param for single device configuration")
	}

	d := &Device{
		address:  *address,
		name:     *device,
		user:     *user,
		password: *password,
		iDesc:    map[string]*prometheus.Desc{},
		rDesc:    map[string]*prometheus.Desc{},
	}

	*c = Config{
		Devices: []Device{*d},
	}

	return nil

}
