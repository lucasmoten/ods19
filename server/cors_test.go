package server_test

import (
	"net/http"
	"strings"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/util"
)

func TestCors(t *testing.T) {
	//Preflight request for a POST is like this:

	origin := "https://proxier" + ":" + cfg.Port
	req, err := http.NewRequest("OPTIONS", mountPoint+"/objects/0123456789abcdef0123456789abcdef", nil)
	if err != nil {
		t.Errorf("Unable to generate request:%v", err)
		t.FailNow()
	}
	//The whole point is to reflect back these headers:
	headers := "content-type, x-requested-with"
	method := "POST"
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", method)
	req.Header.Set("Access-Control-Request-Headers", headers)

	// Do the request:
	clientid := 0
	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName:       "CORS Pre-flight check",
			RequestDescription:  "Do the request, and specify the headers to be reflected back if ok",
			ResponseDescription: "See the headers with values reflected back",
		},
	)
	res1, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v", err)
		t.FailNow()
	}
	trafficLogs[APISampleFile].Response(t, res1)
	defer util.FinishBody(res1.Body)
	if res1.StatusCode != 204 {
		t.Errorf("Unexpected status %s for creator", res1.Status)
		t.FailNow()
	}
	// We are expecting simple reflection right now:
	if res1.Header.Get("Access-Control-Allow-Origin") != origin {
		t.Errorf("Origin mismatch: %s vs %s", origin, res1.Header.Get("Access-Control-Allow-Origin"))
		t.FailNow()
	}
	if !strings.Contains(res1.Header.Get("Access-Control-Allow-Methods"), method) {
		t.Errorf("method mismatch: %s vs %s", origin, res1.Header.Get("Access-Control-Allow-Methods"))
		t.FailNow()
	}
	if !strings.Contains(res1.Header.Get("Access-Control-Allow-Headers"), headers) {
		t.Errorf("method mismatch: %s vs %s", origin, res1.Header.Get("Access-Control-Allow-Headers"))
		t.FailNow()
	}

	// Also check that normal methods get origin checks:
	// Make an arbitrary request, where we set origin and get it reflected back as allowed
	req, err = http.NewRequest("GET", mountPoint+"/userstats", nil)
	if err != nil {
		t.Errorf("Unable to generate request:%v", err)
		t.FailNow()
	}
	req.Header.Set("Origin", origin)
	res1, err = clients[clientid].Client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res1.Body)
	if res1.StatusCode != http.StatusOK {
		t.Errorf("Unexpected status %s for creator", res1.Status)
		t.FailNow()
	}
	// We are expecting simple reflection right now:
	if res1.Header.Get("Access-Control-Allow-Origin") != origin {
		t.Errorf("Origin mismatch: %s vs %s", origin, res1.Header.Get("Access-Control-Allow-Origin"))
		t.FailNow()
	}
}
