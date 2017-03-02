package util

import (
	"strings"
	"testing"
)

type testDelimiter struct {
	value     string
	delimiter string
	part      string
	remainder string
}

func TestGetNextDelimitedPart(t *testing.T) {

	subtests := []testDelimiter{}
	subtests = append(subtests, testDelimiter{value: "abcdef", delimiter: "/", part: "", remainder: "abcdef"})                        // delimiter not found
	subtests = append(subtests, testDelimiter{value: "abc/def", delimiter: "/", part: "abc", remainder: "def"})                       // delimiter found, one part, with one part remainder
	subtests = append(subtests, testDelimiter{value: "abc/def/ghi", delimiter: "/", part: "abc", remainder: "def/ghi"})               // delimiter found, one part, remainder recomposed
	subtests = append(subtests, testDelimiter{value: "abc///def///ghi", delimiter: "///", part: "abc", remainder: "def///ghi"})       // delimiter with length > 1 found
	subtests = append(subtests, testDelimiter{value: "///abc///def///ghi", delimiter: "///", part: "abc", remainder: "def///ghi"})    // delimiter found, and prefix gets removed
	subtests = append(subtests, testDelimiter{value: "///abc///def///ghi///", delimiter: "///", part: "abc", remainder: "def///ghi"}) // delimiter found and suffix gets removed
	subtests = append(subtests, testDelimiter{value: strings.Join([]string{"TestDBMigration20161230", "path", "delimiters"}, string(rune(30))), delimiter: string(rune(30)), part: "TestDBMigration20161230", remainder: strings.Join([]string{"path", "delimiters"}, string(rune(30)))})

	for testIdx, subtest := range subtests {
		value := subtest.value
		delimiter := subtest.delimiter
		t.Logf("Subtest %d: %s with delimiter %s", testIdx, value, delimiter)
		part, remainder := GetNextDelimitedPart(value, delimiter)
		if part != subtest.part {
			t.Logf("[x] Expected part to be (%s), but got (%s)", subtest.part, part)
			t.Fail()
			continue
		}
		if remainder != subtest.remainder {
			t.Logf("[x] Expected remainder to be (%s), but got (%s)", subtest.remainder, remainder)
			t.Fail()
			continue
		}
		t.Logf("OK! part and remainder returned were as expected!")
	}
}
