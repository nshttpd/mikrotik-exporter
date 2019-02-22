package collector

import (
	"testing"
)

func TestParseUptime(t *testing.T) {

	uptimes := []struct {
		u string
		v float64
	}{
		{"3d3h42m53s", 272573},
		{"15w3d3h42m53s", 9344573},
		{"42m53s", 2573},
		{"7w6d9h34m", 4786440},
	}

	for _, uptime := range uptimes {
		seconds, err := parseUptime(uptime.u)
		if err != nil {
			t.Error(err)
		}
		if seconds != uptime.v {
			t.Errorf("seconds : %f != v : %f\n", seconds, uptime.v)
		}
	}
}
