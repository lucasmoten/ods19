package util

import "strings"

// IsApplicationJSON examines a passed in contentType and indicates whether it is an application/json conforming to RFC 2046
func IsApplicationJSON(contentType string) bool {
	return strings.TrimSpace(strings.Split(contentType, ";")[0]) == "application/json"
}
