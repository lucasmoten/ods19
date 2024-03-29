package server_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/protocol"
)

func testBulkDeleteCall(t *testing.T, clientid int, inObjects []protocol.ObjectVersioned, expectedItems int, expectedFailures int) {
	deleteuri := mountPoint + "/objects"
	jsonBody, err := json.Marshal(inObjects)
	failNowOnErr(t, err, "Unable to marshal json")
	req, err := http.NewRequest("DELETE", deleteuri, bytes.NewBuffer(jsonBody))
	failNowOnErr(t, err, "Cannot setup http request")
	res, err := clients[clientid].Client.Do(req)
	failNowOnErr(t, err, "Unable to do request")

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Logf("unable to read body: %v", err)
		t.FailNow()
	} else {
		statusMustBe(t, 200, res, "delete failed")
		var bulkResponse []protocol.ObjectError
		err = json.Unmarshal(bytes, &bulkResponse)
		if err != nil {
			t.Logf("unable to parse body: %v", err)
			t.FailNow()
		}
		failedCount := 0
		responses := len(bulkResponse)
		if responses != expectedItems {
			t.Logf("wrong number of items in response: %v", responses)
			t.FailNow()
		}
		for i := 0; i < responses; i++ {
			if bulkResponse[i].Code != 200 {
				failedCount++
			}
		}
		if failedCount != expectedFailures {
			t.Logf("expected 2 failures, but got %d", failedCount)
			t.FailNow()
		}
	}
}

func TestBulkDelete(t *testing.T) {
	clientid := 0
	clientid2 := 2
	//  Create a few objects we can read
	var inObjects []protocol.ObjectVersioned
	for i := 0; i < 5; i++ {
		o := makeFolderViaJSON("Test Folder for Update ", clientid, t)
		inObject := protocol.ObjectVersioned{
			ObjectID:    o.ID,
			ChangeToken: o.ChangeToken,
		}
		inObjects = append(inObjects, inObject)
	}
	// Create a few objects we cannot delete
	for i := 0; i < 2; i++ {
		o := makeFolderViaJSON("Test Folder Not Ours for Update ", clientid2, t)
		inObject := protocol.ObjectVersioned{
			ObjectID:    o.ID,
			ChangeToken: o.ChangeToken,
		}
		inObjects = append(inObjects, inObject)
	}

	// Delete them in bulk
	testBulkDeleteCall(t, clientid, inObjects, 7, 2)
}

func TestBulkDelete1000(t *testing.T) {
	clientid := 0
	var inObjects []protocol.ObjectVersioned
	o := makeFolderViaJSON("Test BulkDelete1000", clientid, t)
	// Add the same item to be deleted 1000 times
	for i := 0; i < 1000; i++ {
		inObject := protocol.ObjectVersioned{
			ObjectID:    o.ID,
			ChangeToken: o.ChangeToken,
		}
		inObjects = append(inObjects, inObject)
	}
	// Delete them in bulk
	testBulkDeleteCall(t, clientid, inObjects, 1000, 0)
}
func TestBulkDelete1001(t *testing.T) {
	clientid := 0
	var inObjects []protocol.ObjectVersioned
	o := makeFolderViaJSON("Test BulkDelete1001", clientid, t)
	// Add the same item to be deleted 1001 times
	for i := 0; i < 1001; i++ {
		inObject := protocol.ObjectVersioned{
			ObjectID:    o.ID,
			ChangeToken: o.ChangeToken,
		}
		inObjects = append(inObjects, inObject)
	}
	// Delete them in bulk
	deleteuri := mountPoint + "/objects"
	jsonBody, err := json.Marshal(inObjects)
	failNowOnErr(t, err, "Unable to marshal json")
	req, err := http.NewRequest("DELETE", deleteuri, bytes.NewBuffer(jsonBody))
	failNowOnErr(t, err, "Cannot setup http request")
	res, err := clients[clientid].Client.Do(req)
	failNowOnErr(t, err, "Unable to do request")
	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		t.Logf("unable to read body: %v", err)
		t.FailNow()
	} else {
		statusMustBe(t, http.StatusBadRequest, res, "")
	}
	// ok, now remove an item and run it through for cleanup
	inObjects = inObjects[1:]
	testBulkDeleteCall(t, clientid, inObjects, 1000, 0)

}
