package server_test

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"testing"

	cfg "decipher.com/oduploader/config"

	"decipher.com/oduploader/protocol"
)

func TestQuery(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		t.Logf("(Verbose Mode) Using client id %d", clientid)
	}

	// URL
	uri := host + cfg.RootURL + "/query/test"

	// Body
	paging := protocol.PagingRequest{}
	paging.PageNumber = 1
	paging.PageSize = 2
	jsonBody, err := json.Marshal(paging)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}

	// Request
	req, err := http.NewRequest("GET", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpclients[clientid].Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		t.Logf("bad status: %s", res.Status)
		t.FailNow()
	}
	decoder := json.NewDecoder(res.Body)
	var listOfObjects protocol.ObjectResultset
	err = decoder.Decode(&listOfObjects)
	if err != nil {
		t.Logf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}
	if verboseOutput {
		t.Logf("Total Rows: %d", listOfObjects.TotalRows)

		jsonData, err := json.MarshalIndent(listOfObjects, "", "  ")
		if err != nil {
			t.Logf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		t.Logf("Here is the response body:")
		t.Logf(string(jsonData))
	}
}

func TestQuerySortBySizeDescending(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		t.Logf("(Verbose Mode) Using client id %d", clientid)
	}

	// Depends on 2 or more objects with name or description containing 'gettysburg' of varying size
	searchPhrase := "gettysburg"

	// URL
	uri := host + cfg.RootURL + "/query/" + searchPhrase + "?sortField=contentSize&sortAscending=false&PageSize=2&PageNumber=1"

	// Request
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	res, err := httpclients[clientid].Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		t.Logf("bad status: %s", res.Status)
		t.FailNow()
	}
	if verboseOutput {
		t.Logf("Status: %s", res.Status)
	}
	decoder := json.NewDecoder(res.Body)
	var listOfObjects protocol.ObjectResultset
	err = decoder.Decode(&listOfObjects)
	if err != nil {
		t.Logf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}
	if verboseOutput {
		t.Logf("Total Rows: %d", listOfObjects.TotalRows)
		t.Logf("Page Count: %d", listOfObjects.PageCount)
		t.Logf("Page Size: %d", listOfObjects.PageSize)
		jsonData, err := json.MarshalIndent(listOfObjects, "", "  ")
		if err != nil {
			t.Logf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		t.Logf("Here is the response body:")
		t.Logf(string(jsonData))
	}

	// Check that there are enough rows
	if listOfObjects.TotalRows < 2 {
		t.Logf("Not enough rows for this test. Modify the test to force creation of objects, or run full tests in autopilot to populate records this test depends on")
		t.FailNow()
	}

	// Get size of first and last item in resultset
	size1 := listOfObjects.Objects[0].ContentSize
	size2 := listOfObjects.Objects[1].ContentSize
	// If there are more pages, go fetch the last
	if listOfObjects.TotalRows > 2 {
		uri := host + cfg.RootURL + "/query/" + searchPhrase + "?sortField=contentSize&sortAscending=false&PageSize=2&PageNumber=" + strconv.Itoa(listOfObjects.PageCount)
		if err != nil {
			t.Logf("Unable to marshal json for request:%v", err)
			t.FailNow()
		}
		// Request
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			t.Logf("Error setting up HTTP Request: %v", err)
			t.FailNow()
		}
		res, err := httpclients[clientid].Do(req)
		if err != nil {
			t.Logf("Unable to do request:%v", err)
			t.FailNow()
		}
		// Response validation
		if res.StatusCode != http.StatusOK {
			t.Logf("bad status: %s", res.Status)
			t.FailNow()
		}
		if verboseOutput {
			t.Logf("Status: %s", res.Status)
		}
		decoder := json.NewDecoder(res.Body)
		var listOfObjects protocol.ObjectResultset
		err = decoder.Decode(&listOfObjects)
		if err != nil {
			t.Logf("Error decoding json to ObjectResultset: %v", err)
			t.FailNow()
		}
		if verboseOutput {
			t.Logf("Page %d: size %d, rows %d", listOfObjects.PageNumber, listOfObjects.PageSize, listOfObjects.PageRows)
			for _, obj := range listOfObjects.Objects {
				log.Printf("- object.name: %s", obj.Name)
			}
		}
		// Get size of last row
		size2 = listOfObjects.Objects[listOfObjects.PageRows-1].ContentSize
	}

	if size1 <= size2 {
		t.Logf("The size of the first object returned is smaller than or equal to the size of the last object")
		t.Logf("First object size: %d  -- Last object size: %d", size1, size2)
		t.FailNow()
	}
}
