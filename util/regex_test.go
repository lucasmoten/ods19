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

	// TODO handle the case where path is missing.
}

func TestSanitizePath(t *testing.T) {
	okayPath := "/var/www/static/app.js"
	err := SanitizePath(okayPath)
	if err != nil {
		t.Logf("Expected this path to PASS sanitize test: %s\n", okayPath)
		t.Fail()
	}

	badPath := "/going/to/hack/../you/now.js"
	err = SanitizePath(badPath)
	if err == nil {
		t.Logf("Expected this path to FAIL sanitize test: %s\n", badPath)
		t.Fail()
	}
}
