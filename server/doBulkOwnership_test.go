package server_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/protocol"
)

func testBulkOwnershipCall(t *testing.T, clientid int, inObjects []protocol.ObjectVersioned, newOwner string) {
	uri := host + config.NginxRootURL + "/objects/owner/" + newOwner
	jsonBody, err := json.MarshalIndent(inObjects, "", "  ")
	failNowOnErr(t, err, "Unable to marshal request")
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	failNowOnErr(t, err, "Unable to set up request")
	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName:       "Bulk ownership change",
			RequestDescription:  "Files owned by tester10",
			ResponseDescription: "Are now owned by tester09",
		},
	)

	res, err := clients[clientid].Client.Do(req)
	failNowOnErr(t, err, "Unable to do request")
	trafficLogs[APISampleFile].Response(t, res)

	statusMustBe(t, 200, res, "update failed")

	var objectErrors []protocol.ObjectError
	objectErrorsBytes, err := ioutil.ReadAll(res.Body)
	failNowOnErr(t, err, "Could not read all bytes")
	err = json.Unmarshal(objectErrorsBytes, &objectErrors)
	failNowOnErr(t, err, "update failed")
	for i := 0; i < len(objectErrors); i++ {
		if objectErrors[i].Code != 200 {
			t.Logf("some objects were not updated: %s", string(objectErrorsBytes))
			t.FailNow()
		}
	}
}

func TestBulkOwnership(t *testing.T) {
	clientid := 0

	nextUser := "user/cn=test tester09,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"

	//  Create a few objects
	var inObjects []protocol.ObjectVersioned
	for i := 0; i < 5; i++ {
		o := makeFolderViaJSON("Test Folder for Ownership ", clientid, t)
		inObject := protocol.ObjectVersioned{
			ObjectID:    o.ID,
			ChangeToken: o.ChangeToken,
		}
		inObjects = append(inObjects, inObject)
		log.Printf("transfer %s to user %s", o.ID, nextUser)
	}

	testBulkOwnershipCall(
		t,
		clientid,
		inObjects,
		nextUser,
	)
}
