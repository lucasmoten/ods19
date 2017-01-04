package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/protocol"
)

func testBulkDeleteCall(t *testing.T, clientid int, inObjects []protocol.ObjectVersioned) {
	deleteuri := host + config.NginxRootURL + "/objects"
	jsonBody, err := json.Marshal(inObjects)
	failNowOnErr(t, err, "Unable to marshal json")
	req, err := http.NewRequest("DELETE", deleteuri, bytes.NewBuffer(jsonBody))
	failNowOnErr(t, err, "Cannot setup http request")
	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName:       "Bulk delete",
			RequestDescription:  "A list of object ids with change token",
			ResponseDescription: "Any errors that happened",
		},
	)
	res, err := clients[clientid].Client.Do(req)
	failNowOnErr(t, err, "Unable to do request")
	trafficLogs[APISampleFile].Response(t, res)

	statusMustBe(t, 200, res, "delete failed")
}

func TestBulkDelete(t *testing.T) {
	clientid := 0

	//  Create a few objects
	var inObjects []protocol.ObjectVersioned
	for i := 0; i < 5; i++ {
		o := makeFolderViaJSON("Test Folder for Update ", clientid, t)
		inObject := protocol.ObjectVersioned{
			ObjectID:    o.ID,
			ChangeToken: o.ChangeToken,
		}
		inObjects = append(inObjects, inObject)
	}

	// Delete them in bulk
	testBulkDeleteCall(t, clientid, inObjects)
}
