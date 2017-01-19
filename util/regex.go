package util

import (
	"fmt"
	"regexp"
)

// GetRegexCaptureGroups takes a string and a compiled RegExp, and returns
// a map of capture group name to the captured value. Map may be empty, and
// expected keys may not be present. Test for empty string values when
// attempting to get values from the resulting map.
func GetRegexCaptureGroups(s string, re *regexp.Regexp) map[string]string {
	result := make(map[string]string)
	match := re.FindStringSubmatch(s)
	for i, name := range re.SubexpNames() {
		if i != 0 {
			result[name] = match[i]
		}
	}
	return result
}

func SanitizePath(path string) error {

	attackPattern := `\.{2,}`
	re := regexp.MustCompile(attackPattern)
	if re.MatchString(path) {
		return fmt.Errorf("Relative path detected. Possible attack. Path string: %s\n", path)
	}
	attackPattern = `\%`
	re = regexp.MustCompile(attackPattern)
	if re.MatchString(path) {
		return fmt.Errorf("Encoding metacharacter detected. Possible attack. Path string: %s\n", path)
	}
	return nil
}
