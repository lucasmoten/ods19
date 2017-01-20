package util

import (
	"regexp"
	"testing"
)

func TestGetRegexCaptureGroups(t *testing.T) {

	pattern := "/static/(?P<path>.*)"
	s := "/static/js/listObjects.js"
	re := regexp.MustCompile(pattern)
	result := GetRegexCaptureGroups(s, re)

	if result["path"] != "js/listObjects.js" {
		t.Fail()
	}

	if item := result["foo"]; item == "" {
		t.Log("Foo not found in map.")
	}
}

func TestSanitizePath(t *testing.T) {
	okayPath := "/var/www/static/app.js"
	err := SanitizePath("/var/www", okayPath)
	if err != nil {
		t.Logf("Expected this path to PASS sanitize test: %s\n", okayPath)
		t.Fail()
	}

	badPath := "/going/to/hack/../you/now.js"
	err = SanitizePath("/var/www", badPath)
	if err == nil {
		t.Logf("Expected this path to FAIL sanitize test: %s\n", badPath)
		t.Fail()
	}
}
