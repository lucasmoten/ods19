package server_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/deciphernow/object-drive-server/protocol"
)

func testBulkMoveCall(t *testing.T, clientid int, inObjects []protocol.MoveObjectRequest) {
	deleteuri := mountPoint + "/objects/move"
	jsonBody, err := json.Marshal(inObjects)
	failNowOnErr(t, err, "Unable to marshal request")
	req, err := http.NewRequest("POST", deleteuri, bytes.NewBuffer(jsonBody))
	failNowOnErr(t, err, "Unable to set up request")
	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName:       "Bulk move",
			RequestDescription:  "A list of object ids with change token and new parent",
			ResponseDescription: "Any errors that happened",
		},
	)

	res, err := clients[clientid].Client.Do(req)
	failNowOnErr(t, err, "Unable to do request")
	trafficLogs[APISampleFile].Response(t, res)

	statusMustBe(t, 200, res, "move failed")

	var objectErrors []protocol.ObjectError
	objectErrorsBytes, err := ioutil.ReadAll(res.Body)
	failNowOnErr(t, err, "Could not read all bytes")
	err = json.Unmarshal(objectErrorsBytes, &objectErrors)
	failNowOnErr(t, err, "move failed")
	for i := 0; i < len(objectErrors); i++ {
		if objectErrors[i].Code != 200 {
			t.Logf("some objects were not moved: %s", string(objectErrorsBytes))
			t.FailNow()
		}

		objectProperties, err := clients[clientid].C.GetObject(objectErrors[i].ObjectID)
		if err != nil {
			t.Logf("error getting properties to validate object moved: %s", err.Error())
			t.FailNow()
		}
		if inObjects[i].ParentID != objectProperties.ParentID {
			t.Logf("Validation on object move failed. Expected Object '%s' to be moved to parent '%s'. Actual parent id for object '%s' is reported to be '%s'", inObjects[i].ID, inObjects[i].ParentID, objectProperties.ID, objectProperties.ParentID)
			t.FailNow()
		}
	}

}

func TestBulkMove(t *testing.T) {
	clientid := 0

	to := makeFolderViaJSON("Test Folder for Update ", clientid, t)

	//  Create a few objects
	var inObjects []protocol.MoveObjectRequest
	for i := 0; i < 5; i++ {
		o := makeFolderViaJSON("Test Folder for Update ", clientid, t)
		inObject := protocol.MoveObjectRequest{
			ID:          o.ID,
			ChangeToken: o.ChangeToken,
			ParentID:    to.ID,
		}
		inObjects = append(inObjects, inObject)
		t.Logf("moving %s to parent %s", o.ID, to.ID)
	}

	// Delete them in bulk
	testBulkMoveCall(t, clientid, inObjects)
}
