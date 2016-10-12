package config_test

import (
	"os"
	"testing"

	configx "decipher.com/object-drive-server/configx"
)

func TestCascadeStringSlice_EmptyVarYieldsZeroLenSlice(t *testing.T) {

	// Set up TEST_VAR with empty string
	os.Setenv("TEST_VAR", "")

	var empty []string

	result := configx.CascadeStringSlice("TEST_VAR", empty, empty)

	if len(result) != 0 {
		t.Errorf("Expected len 0 for string slice, got %v", len(result))
	}
}
