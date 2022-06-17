package config

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestShouldParse(t *testing.T) {
	b := loadTestFile(t)
	c, err := Load(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("could not parse: %v", err)
	}

	if len(c.Devices) != 2 {
		t.Fatalf("expected 2 devices, got %v", len(c.Devices))
	}

	assertDevice("test1", "192.168.1.1", "foo", "bar", c.Devices[0], t)
	assertDevice("test2", "192.168.2.1", "test", "123", c.Devices[1], t)
	assertDeviceConnection("8728", false, false, c.Devices[0], t)
	assertDeviceConnection("324", true, true, c.Devices[1], t)
	assertFeature("BGP", c.Features.BGP, t)
	assertFeature("Conntrack", c.Features.Conntrack, t)
	assertFeature("DHCP", c.Features.DHCP, t)
	assertFeature("DHCPv6", c.Features.DHCPv6, t)
	assertFeature("Pools", c.Features.Pools, t)
	assertFeature("Routes", c.Features.Routes, t)
	assertFeature("Optics", c.Features.Optics, t)
	assertFeature("WlanSTA", c.Features.WlanSTA, t)
	assertFeature("WlanIF", c.Features.WlanIF, t)
	assertFeature("Ipsec", c.Features.Ipsec, t)
	assertFeature("Lte", c.Features.Lte, t)
	assertFeature("Netwatch", c.Features.Netwatch, t)
}

func loadTestFile(t *testing.T) []byte {
	b, err := ioutil.ReadFile("config.test.yml")
	if err != nil {
		t.Fatalf("could not load config: %v", err)
	}

	return b
}

func assertDevice(name, address, user, password string, c Device, t *testing.T) {
	if c.Name != name {
		t.Fatalf("expected name %s, got %s", name, c.Name)
	}

	if c.Address != address {
		t.Fatalf("expected address %s, got %s", address, c.Address)
	}

	if c.User != user {
		t.Fatalf("expected user %s, got %s", user, c.User)
	}

	if c.Password != password {
		t.Fatalf("expected password %s, got %s", password, c.Password)
	}
}

func assertDeviceConnection(port string, tls, insecure bool, c Device, t *testing.T) {
	if c.Port != port {
		t.Fatalf("expected port %s, got %s", port, c.Port)
	}

	if c.EnableTLS != tls {
		t.Fatalf("expected tls %t, got %t", tls, c.EnableTLS)
	}
	if c.InsecureTLS != insecure {
		t.Fatalf("expected insecure %t, got %t", insecure, c.InsecureTLS)
	}
}

func assertFeature(name string, v bool, t *testing.T) {
	if !v {
		t.Fatalf("exprected feature %s to be enabled", name)
	}
}
