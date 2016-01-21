package config

import (
	"os"
	"strings"
	"testing"
)

// TestStructWithNoFields has no fields
type TestStructWithNoFields struct {
}

// TestStructWithOneStringField has a single field name field
type TestStructWithOneStringField struct {
	Field string
}

// TestStructWithOneIntField has a single field name field
type TestStructWithOneIntField struct {
	Field int
}

// TestStructWithTwoStringFields has two string fields named field1, field2
type TestStructWithTwoStringFields struct {
	Field1 string
	Field2 string
}

// TestStructWithReferencedStruct embeds the TestStructWithOneStringField
type TestStructWithReferencedStruct struct {
	TestStructWithOneStringField
}

// TestStructWithNestedStruct embeds the TestStructWithOneIntField and
// references the TestStructWithTwoStringFields
type TestStructWithNestedStruct struct {
	TestStructWithOneIntField
	Nested TestStructWithTwoStringFields
}

// TestStructWithArrayOfNestedStruct includes an array of TestStructWithTwoStringFields
type TestStructWithArrayOfNestedStruct struct {
	Nested []TestStructWithTwoStringFields
}

func TestExpandEnv(t *testing.T) {
	start := "$GOPATH/src/decipher.com"
	changed := os.ExpandEnv(start)
	if start == changed {
		t.Failed()
	}
}

func TestOSEnvOnTestStructWithNoFields(t *testing.T) {
	s := TestStructWithNoFields{}
	ExpandEnvironmentVariables(&s)
}

func TestOSEnvOnTestStructWithOneStringField(t *testing.T) {
	s := TestStructWithOneStringField{}
	s.Field = "ABC"
	ExpandEnvironmentVariables(&s)
	if s.Field != "ABC" {
		t.Failed()
	}
}

func TestOSEnvOnTestStructWithOneStringFieldExpandGOPATH(t *testing.T) {
	s := TestStructWithOneStringField{}
	s.Field = "$GOPATH"
	ExpandEnvironmentVariables(&s)
	if !strings.HasPrefix(s.Field, "/") {
		t.Failed()
	}
}

func TestOSEnvOnTestSructWithOneIntField(t *testing.T) {
	s := TestStructWithOneIntField{1}
	ExpandEnvironmentVariables(&s)
	if s.Field != 1 {
		t.Failed()
	}
}

func TestOSEnvOnTestStructWithTwoStringFields(t *testing.T) {
	s := TestStructWithTwoStringFields{"$GOPATH", "$OD_ROOT"}
	ExpandEnvironmentVariables(&s)
	if s.Field1 == s.Field2 {
		t.Failed()
	}
	if s.Field1 == "$GOPATH" {
		t.Failed()
	}
	if s.Field2 == "$OD_ROOT" {
		t.Failed()
	}
	if !strings.HasPrefix(s.Field1, "/") {
		t.Failed()
	}
	if !strings.HasPrefix(s.Field2, "/") {
		t.Failed()
	}
}

func TestOSEnvOnTestStrutWithReferencedStruct(t *testing.T) {
	s := TestStructWithReferencedStruct{}
	s.Field = "$GOPATH"
	ExpandEnvironmentVariables(&s)
	if s.Field == "$GOPATH" {
		t.Failed()
	}
	if !strings.HasPrefix(s.Field, "/") {
		t.Failed()
	}
}

func TestOSEnvOnTestStructWithNestedStruct(t *testing.T) {
	s := TestStructWithNestedStruct{}
	s.Field = 123
	s.Nested.Field1 = "$OD_ROOT"
	s.Nested.Field2 = "$PATH"
	ExpandEnvironmentVariables(&s)
	if s.Field != 123 {
		t.Failed()
	}
	if !strings.HasPrefix(s.Nested.Field1, "/") {
		t.Failed()
	}
	if s.Nested.Field1 == "$OD_ROOT" {
		t.Failed()
	}
	if !strings.HasPrefix(s.Nested.Field2, "/") {
		t.Failed()
	}
	if s.Nested.Field2 == "$PATH" {
		t.Failed()
	}
	if !strings.HasPrefix(s.Nested.Field2, "/") {
		t.Failed()
	}
}

func TestOSEnvOnTestStructWithArrayOfNestedStruct(t *testing.T) {
	s := TestStructWithArrayOfNestedStruct{}
	s.Nested = make([]TestStructWithTwoStringFields, 2)
	s.Nested[0].Field1 = "$GOPATH"
	s.Nested[1].Field2 = "$OD_ROOT"
}
