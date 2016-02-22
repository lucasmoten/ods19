package main

import (
	"testing"
)

func TestAutopilot(t *testing.T) {
	if testing.Short() == true {
		doMainDefault()
	}
}

func init() {
}
