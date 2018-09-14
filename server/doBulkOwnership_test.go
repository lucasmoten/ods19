package server_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/protocol"
)

func testBulkOwnershipCall(t *testing.T, clientid int, inObjects []protocol.ObjectVersioned, newOwner string) {
	uri := mountPoint + "/objects/owner/" + newOwner
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
	for _, oe := range objectErrors {
		if oe.Code != 200 {
			t.Logf("some objects were not updated: %s", string(objectErrorsBytes))
			t.FailNow()
		}
	}
}

func bulkOwnershipTo(t *testing.T, clientid int, nextUser string) {
	//  Create a few objects
	var inObjects []protocol.ObjectVersioned
	for i := 0; i < 5; i++ {
		o := makeFolderViaJSON("Test Folder for Ownership ", clientid, t)
		inObject := protocol.ObjectVersioned{
			ObjectID:    o.ID,
			ChangeToken: o.ChangeToken,
		}
		inObjects = append(inObjects, inObject)
		t.Logf("transfer %s to user %s", o.ID, nextUser)
	}

	testBulkOwnershipCall(
		t,
		clientid,
		inObjects,
		nextUser,
	)
}

func TestBulkOwnership(t *testing.T) {
	nextUser := ""

	// transfer to self
	nextUser = "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	bulkOwnershipTo(t, 0, nextUser)

	// transfer to other user
	nextUser = "user/cn=test tester09,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	bulkOwnershipTo(t, 0, nextUser)

	// transfer to group
	nextUser = "group/dctc_odrive"
	bulkOwnershipTo(t, 0, nextUser)

}
