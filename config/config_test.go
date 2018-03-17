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
}

func loadTestFile(t *testing.T) []byte {
	b, err := ioutil.ReadFile("test/config.test.yml")
	if err != nil {
		t.Fatalf("could not load config: %v", err)
	}

	return b
}
