package util

import (
	"fmt"
	"regexp"
	"strings"
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

// SanitizePath is for files that came from hostile input.  it's more restrictive than all allowed files
// (ie: Arabic names, things that really do need percent in the name - it may be other people's files that we did not invent the name for),
// since we not only ensure that we are under the root, but disallow percentages as if they are escapes.
// it's not appropriate for everything, but it is good for files that we control the names of.
func SanitizePath(root, path string) error {
	// The path must really begin with the root, and not use .. tricks to apparently have the prefix, but get out.
	if strings.HasPrefix(path, root) == false {
		return fmt.Errorf("normalized path is not in root.  definite attack attempt: %s", path)
	}
	attackPattern := `\.{2,}`
	re := regexp.MustCompile(attackPattern)
	if re.MatchString(path) {
		return fmt.Errorf("relative path detected. possible attack. path string: %s", path)
	}
	attackPattern = `\%`
	re = regexp.MustCompile(attackPattern)
	if re.MatchString(path) {
		return fmt.Errorf("encoding metacharacter detected. possible attack. path string: %s", path)
	}
	return nil
}
