package main

import (
	"log"
	"net/http"
	"testing"

	"decipher.com/object-drive-server/autopilot"
)

/*
Things common between all tests
*/

var userID0 = 0
var userID1 = 1

func resErrCheck(t *testing.T, res *http.Response, err error) {
	if err != nil {
		log.Printf("error came back:%v", err)
		t.FailNow()
		return
	}
	if res == nil {
		log.Printf("we got a null result back")
		t.FailNow()
		return
	}
	if res.StatusCode != http.StatusOK {
		log.Printf("http status must be ok.  we got %d.  %s", res.StatusCode, res.Status)
		t.FailNow()
		return
	}
}

func init() {
	autopilot.Init()
}
