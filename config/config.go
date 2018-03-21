package config

import (
	"io"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Devices []Device `yaml:"devices"`
}

type Device struct {
	Name     string `yaml:"name"`
	Address  string `yaml:"address"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func Load(r io.Reader) (*Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
