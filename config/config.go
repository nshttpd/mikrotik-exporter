package config

import (
	"io"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// Features represents the collector features that can be enabled/disabled
type Features struct {
	BGP       *bool `yaml:"bgp,omitempty"`
	Conntrack *bool `yaml:"conntrack,omitempty"`
	DHCP      *bool `yaml:"dhcp,omitempty"`
	DHCPL     *bool `yaml:"dhcpl,omitempty"`
	DHCPv6    *bool `yaml:"dhcpv6,omitempty"`
	Firmware  *bool `yaml:"firmware,omitempty"`
	Health    *bool `yaml:"health,omitempty"`
	Routes    *bool `yaml:"routes,omitempty"`
	POE       *bool `yaml:"poe,omitempty"`
	Pools     *bool `yaml:"pools,omitempty"`
	Optics    *bool `yaml:"optics,omitempty"`
	W60G      *bool `yaml:"w60g,omitempty"`
	WlanSTA   *bool `yaml:"wlansta,omitempty"`
	Capsman   *bool `yaml:"capsman,omitempty"`
	WlanIF    *bool `yaml:"wlanif,omitempty"`
	Monitor   *bool `yaml:"monitor,omitempty"`
	Ipsec     *bool `yaml:"ipsec,omitempty"`
	Lte       *bool `yaml:"lte,omitempty"`
	Netwatch  *bool `yaml:"netwatch,omitempty"`
	Cloud     *bool `yaml:"cloud,omitempty"`
}

// Config represents the configuration for the exporter
type Config struct {
	Devices  []Device `yaml:"devices"`
	Features Features `yaml:"features,omitempty"`
}

// Device represents a target device
type Device struct {
	Name      string    `yaml:"name"`
	Address   string    `yaml:"address,omitempty"`
	Srv       SrvRecord `yaml:"srv,omitempty"`
	User      string    `yaml:"user"`
	Password  string    `yaml:"password"`
	Port      string    `yaml:"port"`
	Wifiwave2 bool      `yaml:"wifiwave2"`
	Features  Features  `yaml:"features,omitempty"` // Per-device feature overrides
}

// IsFeatureEnabled checks if a feature is enabled for a device
// Per-device settings override global settings
func (d *Device) IsFeatureEnabled(globalFeatures *Features, featureName string) bool {
	// Check per-device override first
	deviceVal := d.Features.getFeatureValue(featureName)
	if deviceVal != nil {
		return *deviceVal
	}
	// Fall back to global setting
	globalVal := globalFeatures.getFeatureValue(featureName)
	if globalVal != nil {
		return *globalVal
	}
	return false
}

func (f *Features) getFeatureValue(name string) *bool {
	if f == nil {
		return nil
	}
	switch name {
	case "bgp":
		return f.BGP
	case "conntrack":
		return f.Conntrack
	case "dhcp":
		return f.DHCP
	case "dhcpl":
		return f.DHCPL
	case "dhcpv6":
		return f.DHCPv6
	case "firmware":
		return f.Firmware
	case "health":
		return f.Health
	case "routes":
		return f.Routes
	case "poe":
		return f.POE
	case "pools":
		return f.Pools
	case "optics":
		return f.Optics
	case "w60g":
		return f.W60G
	case "wlansta":
		return f.WlanSTA
	case "capsman":
		return f.Capsman
	case "wlanif":
		return f.WlanIF
	case "monitor":
		return f.Monitor
	case "ipsec":
		return f.Ipsec
	case "lte":
		return f.Lte
	case "netwatch":
		return f.Netwatch
	case "cloud":
		return f.Cloud
	}
	return nil
}

type SrvRecord struct {
	Record string    `yaml:"record"`
	Dns    DnsServer `yaml:"dns,omitempty"`
}
type DnsServer struct {
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

// Load reads YAML from reader and unmashals in Config
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
