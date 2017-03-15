package util

import "strings"

// DefaultPathDelimiter indicates a default path delimiter to use if none specified.
const DefaultPathDelimiter = string(rune(30)) // 30 = record separator

// GetNextDelimitedPart removes delimiter from beginning and ending of passed in value and
// then looks to see if the resultant value contains it. If it does, the first part is
// return along with assembeled remainder.  Otherwise, an empty string is given indicating
// there are no more delimited parts
func GetNextDelimitedPart(value string, delimiter string) (part string, remainder string) {
	if len(delimiter) == 0 {
		delimiter = DefaultPathDelimiter
	}
	cleansed := value
	for strings.HasPrefix(cleansed, delimiter) {
		cleansed = cleansed[len(delimiter):]
	}
	for strings.HasSuffix(cleansed, delimiter) {
		cleansed = cleansed[:len(cleansed)-len(delimiter)]
	}
	res := strings.Split(cleansed, delimiter)
	if len(res) > 1 {
		return res[0], strings.Join(res[1:], delimiter)
	}
	return "", cleansed
}
