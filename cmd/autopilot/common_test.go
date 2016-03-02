package main

import (
	"decipher.com/oduploader/autopilot"
	"log"
	"net/http"
	"testing"
)

/*
Things common between all tests
*/

var userID0 = 0
var userID1 = 1

func resErrCheck(t *testing.T, res *http.Response, err error) {
	if err != nil {
		log.Printf("error came back:%v", err)
		t.Fail()
	}
	if res.StatusCode != http.StatusOK {
		log.Printf("http status must be ok.  we got %d.  %s",res.StatusCode, res.Status)
		t.Fail()
	}
}

func init() {
	autopilot.Init()
}
