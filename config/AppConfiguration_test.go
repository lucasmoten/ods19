package config_test

import (
	"os"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/config"
)

func TestCascadeStringSlice_EmptyVarYieldsZeroLenSlice(t *testing.T) {

	// Set up TEST_VAR with empty string
	os.Setenv("TEST_VAR", "")

	var empty []string

	result := config.CascadeStringSlice("TEST_VAR", empty, empty)

	if len(result) != 0 {
		t.Errorf("Expected len 0 for string slice, got %v", len(result))
	}
}
