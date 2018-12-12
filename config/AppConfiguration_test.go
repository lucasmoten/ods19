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

func TestCascadeBoolFromString1(t *testing.T) {
	os.Setenv("TEST_BOOL", "")

	var empty string

	result := config.CascadeBoolFromString("TEST_BOOL", empty, true)

	if !result {
		t.Errorf("Expected true for default when env var not set, got %v", result)
	}
}

func TestCascadeBoolFromString2(t *testing.T) {
	os.Setenv("TEST_BOOL", "false")

	var empty string

	result := config.CascadeBoolFromString("TEST_BOOL", empty, true)

	if result {
		t.Errorf("Expected false because env var should override, got %v", result)
	}
}

func TestCascadeBoolFromString3(t *testing.T) {
	os.Setenv("TEST_BOOL", "false")

	var empty = "true"

	result := config.CascadeBoolFromString("TEST_BOOL", empty, true)

	if result {
		t.Errorf("Expected false because env var should override, got %v", result)
	}
}

func TestCascadeBoolFromString4(t *testing.T) {
	os.Setenv("TEST_BOOL", "")

	var empty string

	result := config.CascadeBoolFromString("TEST_BOOL", empty, true)

	if !result {
		t.Errorf("Expected true because of default, got %v", result)
	}
}

func TestCascadeBoolFromString5(t *testing.T) {
	os.Setenv("TEST_BOOL", "")

	result := config.CascadeBoolFromString("TEST_BOOL", "true", false)

	if !result {
		t.Errorf("Expected true because of file value, got %v", result)
	}
}
