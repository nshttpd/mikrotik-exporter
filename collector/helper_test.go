package collector

import (
	"math"
	"testing"
	"unicode/utf8"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestCleanLabelValue(t *testing.T) {
	// "künden." encoded as Windows-1252/Latin-1, as returned by some RouterOS devices.
	invalid := "k\xfcnden."

	assert.False(t, utf8.ValidString(invalid))

	cleaned := cleanLabelValue(invalid)
	assert.True(t, utf8.ValidString(cleaned))
	assert.Equal(t, "künden.", cleanLabelValue("künden."))

	desc := prometheus.NewDesc("test_metric", "", []string{"comment"}, nil)
	assert.NotPanics(t, func() {
		prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, 1, cleaned)
	})
}

func TestSplitStringToFloats(t *testing.T) {
	var testCases = []struct {
		input    string
		expected struct {
			f1 float64
			f2 float64
		}
		isNaN    bool
		hasError bool
	}{
		{
			"1.2,2.1",
			struct {
				f1 float64
				f2 float64
			}{
				1.2,
				2.1,
			},
			false,
			false,
		},
		{
			input:    "1.2,",
			isNaN:    true,
			hasError: true,
		},
		{
			input:    ",2.1",
			isNaN:    true,
			hasError: true,
		},
		{
			"1.2,2.1,3.2",
			struct {
				f1 float64
				f2 float64
			}{
				1.2,
				2.1,
			},
			false,
			false,
		},
		{
			input:    "",
			isNaN:    true,
			hasError: true,
		},
	}

	for _, testCase := range testCases {
		f1, f2, err := splitStringToFloats(testCase.input)

		switch testCase.hasError {
		case true:
			assert.Error(t, err)
		case false:
			assert.NoError(t, err)
		}

		switch testCase.isNaN {
		case true:
			assert.True(t, math.IsNaN(f1))
			assert.True(t, math.IsNaN(f2))
		case false:
			assert.Equal(t, testCase.expected.f1, f1)
			assert.Equal(t, testCase.expected.f2, f2)
		}
	}
}

func TestParseDuration(t *testing.T) {
	var testCases = []struct {
		input    string
		output   float64
		hasError bool
	}{
		{
			"3d3h42m53s",
			272573,
			false,
		},
		{
			"15w3d3h42m53s",
			9344573,
			false,
		},
		{
			"42m53s",
			2573,
			false,
		},
		{
			"7w6d9h34m",
			4786440,
			false,
		},
		{
			"59",
			0,
			true,
		},
		{
			"s",
			0,
			false,
		},
		{
			"",
			0,
			false,
		},
	}

	for _, testCase := range testCases {
		f, err := parseDuration(testCase.input)

		switch testCase.hasError {
		case true:
			assert.Error(t, err)
		case false:
			assert.NoError(t, err)
		}

		assert.Equal(t, testCase.output, f)
	}
}
