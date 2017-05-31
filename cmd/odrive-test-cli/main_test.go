package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseTestString tests the mapping of string testers
// into the appropriate values needed.
func TestParseTestString(t *testing.T) {
	truthMappings := map[string]string{
		"0":     "00",
		"1":     "01",
		"2":     "02",
		"3":     "03",
		"4":     "04",
		"5":     "05",
		"6":     "06",
		"7":     "07",
		"8":     "08",
		"9":     "09",
		"10":    "10",
		"01":    "01",
		"005":   "05",
		"00010": "10",
	}

	for input, truth := range truthMappings {
		output, err := parseTesterString(input)
		assert.Nil(t, err, fmt.Sprintf("error parsing string: %s", err))
		if output != truth {
			t.Error("Failed to parse ", input, " into ", truth)
		}
	}
}
