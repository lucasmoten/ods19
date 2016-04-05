package server_test

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"testing"
	"time"

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
	uri := host + cfg.RootURL + "/search/test"

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

func TestQuerySortByVersionDescending(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		t.Logf("(Verbose Mode) Using client id %d", clientid)
	}

	searchPhrase := "QuerySortByVersionDescending"

	// Create 2 folders under root
	folder1, err := makeFolderViaJSON("Test Folder 1 "+searchPhrase+" "+strconv.FormatInt(time.Now().Unix(), 10), clientid)
	if err != nil {
		log.Printf("Error making folder 1: %v", err)
		t.FailNow()
	}
	folder2, err := makeFolderViaJSON("Test Folder 2 "+searchPhrase+" "+strconv.FormatInt(time.Now().Unix(), 10), clientid)
	if err != nil {
		log.Printf("Error making folder 2: %v", err)
		t.FailNow()
	}
	// Modify the 1st folder
	updateuri := host + cfg.RootURL + "/objects/" + folder1.ID + "/properties"
	updateObjectRequest := protocol.UpdateObjectRequest{}
	updateObjectRequest.Name = folder1.Name
	updateObjectRequest.Description = "The folder has been changed once"
	updateObjectRequest.ChangeToken = folder1.ChangeToken
	jsonBody, err := json.Marshal(updateObjectRequest)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req1, err := http.NewRequest("POST", updateuri, bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	res1, err := httpclients[clientid].Do(req1)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	// process Response
	if res1.StatusCode != http.StatusOK {
		log.Printf("bad status modifying folder 1 first time: %s", res1.Status)
		t.FailNow()
	}
	decoder := json.NewDecoder(res1.Body)
	var updatedFolder protocol.Object
	err = decoder.Decode(&updatedFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	updateObjectRequest.ChangeToken = updatedFolder.ChangeToken
	updateObjectRequest.Description = "The folder has been changed twice"
	// Modify the 1st folder again
	updateuri = host + cfg.RootURL + "/objects/" + folder1.ID + "/properties"
	jsonBody, err = json.Marshal(updateObjectRequest)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req2, err := http.NewRequest("POST", updateuri, bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	res2, err := httpclients[clientid].Do(req2)
	if err != nil {
		log.Printf("Unable to do request to modify folder again:%v", err)
		t.FailNow()
	}
	// process Response
	if res2.StatusCode != http.StatusOK {
		log.Printf("bad status modifying folder 1 second time: %s", res2.Status)
		t.FailNow()
	}
	decoder = json.NewDecoder(res2.Body)
	err = decoder.Decode(&updatedFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	folder1.ChangeToken = updatedFolder.ChangeToken

	// URL
	uri := host + cfg.RootURL + "/search/" + searchPhrase + "?sortField=version&sortAscending=false&PageSize=2&PageNumber=1"

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
		t.Logf("bad status searching: %s", res.Status)
		t.FailNow()
	}
	if verboseOutput {
		t.Logf("Status: %s", res.Status)
	}
	decoder = json.NewDecoder(res.Body)
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

	// Get changes  of first and last item in resultset
	changes1 := listOfObjects.Objects[0].ChangeCount
	changes2 := listOfObjects.Objects[1].ChangeCount
	// If there are more pages, go fetch the last
	if listOfObjects.TotalRows > 2 {
		uri := host + cfg.RootURL + "/search/" + searchPhrase + "?sortField=version&sortAscending=false&PageSize=2&PageNumber=" + strconv.Itoa(listOfObjects.PageCount)
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
			t.Logf("bad status searching page 2: %s", res.Status)
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
		// Get changes of last row
		changes2 = listOfObjects.Objects[listOfObjects.PageRows-1].ChangeCount
	}

	if changes1 <= changes2 {
		t.Logf("The change count of the first object returned is smaller than or equal to the change count of the last object")
		t.Logf("First object change count: %d  -- Last object change count: %d", changes1, changes2)
		t.FailNow()
	}

	// Cleanup
	// Now delete the first folder
	deleteuri := host + cfg.RootURL + "/objects/" + folder1.ID + "/trash"
	objChangeToken := protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = folder1.ChangeToken
	jsonBody, err = json.Marshal(objChangeToken)
	if err != nil {
		log.Printf("deleting folder Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req3, err := http.NewRequest("POST", deleteuri, bytes.NewBuffer(jsonBody))
	req3.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("deleting folder Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	res3, err := httpclients[clientid].Do(req3)
	if err != nil {
		log.Printf("deleting folder Unable to do request:%v", err)
		t.FailNow()
	}
	// process Response
	if res3.StatusCode != http.StatusOK {
		log.Printf("deleting folder bad status: %s", res3.Status)
		t.FailNow()
	}
	decoder = json.NewDecoder(res3.Body)
	var deletedFolder1 protocol.DeletedObjectResponse
	err = decoder.Decode(&deletedFolder1)
	if err != nil {
		log.Printf("deleting folder Error decoding json to Object 1: %v", err)
		log.Println()
		t.FailNow()
	}

	// Now delete the second folder
	deleteuri = host + cfg.RootURL + "/objects/" + folder2.ID + "/trash"
	objChangeToken = protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = folder2.ChangeToken
	jsonBody, err = json.Marshal(objChangeToken)
	if err != nil {
		log.Printf("deleting folder Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req4, err := http.NewRequest("POST", deleteuri, bytes.NewBuffer(jsonBody))
	req4.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("deleting folder Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	res4, err := httpclients[clientid].Do(req4)
	if err != nil {
		log.Printf("deleting folder Unable to do request:%v", err)
		t.FailNow()
	}
	// process Response
	if res4.StatusCode != http.StatusOK {
		log.Printf("deleting folder bad status: %s", res4.Status)
		t.FailNow()
	}
	decoder = json.NewDecoder(res4.Body)
	var deletedFolder2 protocol.DeletedObjectResponse
	err = decoder.Decode(&deletedFolder2)
	if err != nil {
		log.Printf("deleting folder Error decoding json to Object 2: %v", err)
		log.Println()
		t.FailNow()
	}
}
