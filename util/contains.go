package util

import "strings"

// ContainsAny indicates if the value of msg is present in any of the values of the string array
func ContainsAny(msg string, a []string) bool {
	for _, s := range a {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// FirstMatch yields msg if it is present in the string array, otherwise returns empty string
func FirstMatch(msg string, a []string) string {
	for _, s := range a {
		if strings.Contains(msg, s) {
			return s
		}
	}
	return ""
}
