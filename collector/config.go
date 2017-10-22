package exporter

import (
	"fmt"
	"go.uber.org/zap"
)

type Config struct {
	Devices []Device
	Logger  *zap.SugaredLogger
	Metrics PromMetrics
}

func (c *Config) FromFlags(device, address, user, password *string) error {
	if *device == "" || *address == "" || *user == "" || *password == "" {
		return fmt.Errorf("missing required param for single device configuration")
	}

	d := &Device{
		Address:  *address,
		Name:     *device,
		User:     *user,
		Password: *password,
	}

	*c = Config{
		Devices: []Device{*d},
	}

	return nil

}
