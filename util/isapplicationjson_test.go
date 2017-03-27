package util_test

import "testing"
import "decipher.com/object-drive-server/util"

func TestIsApplicationJSON(t *testing.T) {
	if util.IsApplicationJSON("text/plain") {
		t.FailNow()
	}
	if !util.IsApplicationJSON("application/json") {
		t.FailNow()
	}
	if util.IsApplicationJSON("APPLICATION/JSON") {
		t.FailNow()
	}
	if !util.IsApplicationJSON("application/json ;charset=UTF-8") {
		t.FailNow()
	}
}
