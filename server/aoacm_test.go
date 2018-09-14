package server_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/util"

	"bitbucket.di2e.net/dime/object-drive-server/protocol"
)

func TestAOACMPerformance(t *testing.T) {
	// only run this manually. its designed to take some time and will generally timeout on circle and regular testing
	if isCircleCI() {
		t.Skip()
	}

	if testing.Short() {
		t.Skip()
	}

	testTime := time.Now()
	// The following are merely for convenience to target this manual test against a different server
	//basehost = "https://bedrock.363-283.io"
	//basehost = "https://chm.363-283.io"
	var uris []string
	uris = append(uris, "/objects")
	uris = append(uris, "/shares")
	uris = append(uris, "/shared")
	uris = append(uris, "/sharedpublic")
	uris = append(uris, "/search/AA")
	uris = append(uris, "/trashed")

	for i := 0; i <= 1; i++ {
		t.Logf("%s", clients[i].Name)
		if time.Since(testTime).Seconds() > 550 {
			t.Logf("out of time. skipping")
			break
		}
		t.Logf("----------------------------------------------------------------------------")
		t.Logf("operation      matches    1       2       3       4       5     average time")
		t.Logf("----------------------------------------------------------------------------")
		for _, operation := range uris {
			uri := mountPoint + operation
			var responseTimes []float64
			var responseTimesString []string
			responseTimeSuccess := 0
			totalTime := float64(0)
			matches := -1
			for r := 1; r <= 5; r++ {
				if time.Since(testTime).Seconds() > 530 {
					responseTimesString = append(responseTimesString, "SKIPPED")
					continue
				}
				req, err := http.NewRequest("GET", uri, nil)
				if err != nil {
					t.Logf("Error setting up HTTP Request: %v", err)
					t.FailNow()
				}
				req.Header.Set("Content-Type", "application/json")
				timeStart := time.Now()
				res, err := clients[i].Client.Do(req)
				if err != nil {
					t.Logf("Unable to do request: %v", err)
					t.FailNow()
				}
				duration := time.Since(timeStart).Seconds()
				totalTime = totalTime + duration
				responseTimes = append(responseTimes, duration)
				switch res.StatusCode {
				case 200:
					if matches == -1 {
						var listOfObjects protocol.ObjectResultset
						rawString, err := util.FullDecodeRawString(res.Body, &listOfObjects)
						if err != nil {
							t.Logf("error decoding body: %v", err)
							t.Logf("%s", rawString)
							t.FailNow()
						}
						matches = listOfObjects.TotalRows
					}
					responseTimesString = append(responseTimesString, fmt.Sprintf("%7.3f", duration))
					responseTimeSuccess++
				case 504:
					responseTimesString = append(responseTimesString, "TIMEOUT")
				default:
					responseTimesString = append(responseTimesString, fmt.Sprintf("ERR %d", res.StatusCode))
				}
				util.FinishBody(res.Body)
			}
			avgTimeString := "NO DATA"
			if responseTimeSuccess > 0 {
				avgTime := float64(totalTime) / float64(responseTimeSuccess)
				avgTimeString = fmt.Sprintf("%10.4f", avgTime)
			}
			t.Logf("%-14s %7d %7s %7s %7s %7s %7s %10s", operation, matches, responseTimesString[0], responseTimesString[1], responseTimesString[2], responseTimesString[3], responseTimesString[4], avgTimeString)
		}
		t.Logf("")
		t.Logf("")
	}
}
