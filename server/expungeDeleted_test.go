package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/server"
	"decipher.com/object-drive-server/util"
)

func testExpungeDeletedCreateObject(t *testing.T, clientid int) *protocol.Object {
	return makeFolderViaJSON("Test Folder for Update ", clientid, t)
}

func countTrash(t *testing.T, clientID int) int {
	trashURI := mountPoint + "/trashed?pageNumber=1&pageSize=1000"

	trashReq, err := http.NewRequest("GET", trashURI, nil)
	if err != nil {
		t.Errorf("Could not create trashReq: %v\n", err)
	}
	trashResp, err := clients[clientID].Client.Do(trashReq)
	if err != nil {
		t.Errorf("Unable to do trash request:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(trashResp.Body)

	var trashResponse protocol.ObjectResultset
	err = util.FullDecode(trashResp.Body, &trashResponse)
	if err != nil {
		t.Errorf("Could not decode listObjectsTrashed ObjectResultset response.")
	}
	return trashResponse.TotalRows
}

func testExpungeDeletedDeleteObject(t *testing.T, clientid int, obj *protocol.Object) {
	deleteuri := mountPoint + "/objects/" + obj.ID + "/trash"
	objChangeToken := protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = obj.ChangeToken
	jsonBody, err := json.Marshal(objChangeToken)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("POST", deleteuri, bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}

	if res.StatusCode != 200 {
		t.Logf("delete failed: %d %s", res.StatusCode, res.Status)
		t.FailNow()
	}
}

func TestExpungeDeleted(t *testing.T) {
	clientid := 0

	folder := testExpungeDeletedCreateObject(t, clientid)
	t.Logf("created folder parent id: %s", folder.ParentID)
	testExpungeDeletedDeleteObject(t, clientid, folder)

	countTrashBefore := countTrash(t, clientid)

	// Clean the trash
	trashuri := mountPoint + "/trashed"

	req, err := http.NewRequest("DELETE", trashuri, nil)
	// do the request
	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName:       "Take out the trash",
			RequestDescription:  "Simple clean trash request (expunge deleted objects)",
			ResponseDescription: "The response may be partial, so we keep doing until trash is gone if we need to",
		},
	)
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	trafficLogs[APISampleFile].Response(t, res)
	defer util.FinishBody(res.Body)
	// process Response
	if res.StatusCode != http.StatusOK {
		t.Logf("bad status: %s", res.Status)
		t.FailNow()
	}

	var stats server.ExpungedStats
	util.FullDecode(res.Body, &stats)

	t.Logf("expunged: %d", stats.ExpungedCount)
	if stats.ExpungedCount == 0 {
		t.Logf("we should have expunged at least one object")
		t.FailNow()
	}

	countTrashAfter := countTrash(t, clientid)

	t.Logf("we had %d trash items before empty trash, but %d after", countTrashBefore, countTrashAfter)
	if countTrashBefore > 0 && countTrashAfter > 0 {
		t.FailNow()
	}
}
