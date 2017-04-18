package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"

	"decipher.com/object-drive-server/protocol"
)

// TestChangeOwner validates that change owner is implemented, ownership changes, and parent set to root
func TestChangeOwner(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	clientid := 0
	fakeDN0Owner := "user/" + fakeDN0
	fakeDN1Owner := "user/" + fakeDN1

	t.Logf("* Creating 2 folders under root")
	folder1 := makeFolderViaJSON("Test ChangeOwner Folder 1 ", clientid, t)
	folder2 := makeFolderViaJSON("Test ChangeOwner Folder 2 ", clientid, t)
	t.Logf("* Verifying owner of both folders as %s", fakeDN0Owner)
	if folder1.OwnedBy != fakeDN0Owner {
		t.Logf("Owner for folder1 is %s expected %s", folder1.OwnedBy, fakeDN0Owner)
		t.FailNow()
	}
	if folder2.OwnedBy != fakeDN0Owner {
		t.Logf("Owner for folder2 is %s expected %s", folder2.OwnedBy, fakeDN0Owner)
		t.FailNow()
	}
	t.Logf("* Moving folder 2 under folder 1")
	moveuri := host + config.NginxRootURL + "/objects/" + folder2.ID + "/move/" + folder1.ID
	objChangeToken := protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = folder2.ChangeToken
	jsonBody, err := json.Marshal(objChangeToken)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("POST", moveuri, bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	// do the request
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// process Response
	if res.StatusCode != http.StatusOK {
		t.Logf("bad status: %s", res.Status)
		t.FailNow()
	}
	var updatedFolder protocol.Object
	err = util.FullDecode(res.Body, &updatedFolder)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Verifying owner of moved folder is still %s", fakeDN0Owner)
	if updatedFolder.OwnedBy != fakeDN0Owner {
		t.Logf("Owner for folder2 is %s expected %s", updatedFolder.OwnedBy, fakeDN0Owner)
		t.FailNow()
	}

	newowner := fakeDN1Owner
	t.Logf("* Changing owner of folder 2 to %s", newowner)
	changeowneruri := host + config.NginxRootURL + "/objects/" + folder2.ID + "/owner/" + newowner
	objChangeToken.ChangeToken = updatedFolder.ChangeToken
	changeOwnerRequest := makeHTTPRequestFromInterface(t, "POST", changeowneruri, objChangeToken)
	changeOwnerResponse, err := clients[clientid].Client.Do(changeOwnerRequest)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, changeOwnerResponse, "Bad status when changing owner")
	var updatedObject protocol.Object
	err = util.FullDecode(changeOwnerResponse.Body, &updatedObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Verifying owner changed")
	if updatedObject.OwnedBy != newowner {
		t.Logf("Owner for folder2 is %s expected %s", updatedObject.OwnedBy, newowner)
		t.FailNow()
	}

	t.Logf("* Verifying object moved to root")
	if updatedObject.ParentID != "" {
		t.Logf("folder 2 parent is %s, expected it to be moved to root", updatedObject.ParentID)
		t.FailNow()
	}
}

func TestChangeOwnerToGroup(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	clientid := 0

	t.Logf("* Creating a private object")
	myobject := protocol.CreateObjectRequest{Name: "Test ChangeOwner to Group", TypeName: "Folder", RawAcm: testhelpers.ValidACMUnclassifiedFOUOSharedToTester10}
	newobjuri := host + config.NginxRootURL + "/objects"
	createObjectReq := makeHTTPRequestFromInterface(t, "POST", newobjuri, myobject)
	createObjectRes, err := clients[clientid].Client.Do(createObjectReq)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createObjectRes, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(createObjectRes.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Verify the right users can read")
	shouldHaveReadForObjectID(t, createdObject.ID, 0)
	shouldNotHaveReadForObjectID(t, createdObject.ID, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	if t.Failed() {
		t.FailNow()
	}

	t.Logf("* Changing ownership to group")
	newowner := "group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1"
	newownernodisplayname := "group/dctc/DCTC/ODrive_G1"
	changeowneruri := host + config.NginxRootURL + "/objects/" + createdObject.ID + "/owner/" + newowner
	objChangeToken := protocol.ChangeTokenStruct{ChangeToken: createdObject.ChangeToken}
	changeOwnerRequest := makeHTTPRequestFromInterface(t, "POST", changeowneruri, objChangeToken)
	trafficLogs[APISampleFile].Request(t, changeOwnerRequest,
		&TrafficLogDescription{
			OperationName:       "Change Owner for Object",
			RequestDescription:  "Transfer object ownershpi to a different user or group",
			ResponseDescription: "Ownership changed to designated resource, and moved to root",
		},
	)
	changeOwnerResponse, err := clients[clientid].Client.Do(changeOwnerRequest)
	trafficLogs[APISampleFile].Response(t, changeOwnerResponse)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, changeOwnerResponse, "Bad status when changing owner")
	var updatedObject protocol.Object
	err = util.FullDecode(changeOwnerResponse.Body, &updatedObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Verifying owner changed")
	if updatedObject.OwnedBy != newownernodisplayname {
		t.Logf("Owner for folder2 is %s expected %s", updatedObject.OwnedBy, newowner)
		t.FailNow()
	}
	t.Logf("* Verify the right users can read")
	shouldHaveReadForObjectID(t, updatedObject.ID, 0, 6, 7, 8, 9)
	shouldNotHaveReadForObjectID(t, updatedObject.ID, 1, 2, 3, 4, 5)
}

func TestChangeOwnerToEveryoneDisallowed(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	clientid := 0

	t.Logf("* Creating folders under root")
	folder1 := makeFolderViaJSON("Test ChangeOwner to Everyone", clientid, t)

	t.Logf("* Attempting to change ownership to everyone")
	newowner := "group/-Everyone"
	changeowneruri := host + config.NginxRootURL + "/objects/" + folder1.ID + "/owner/" + newowner
	objChangeToken := protocol.ChangeTokenStruct{ChangeToken: folder1.ChangeToken}
	changeOwnerRequest := makeHTTPRequestFromInterface(t, "POST", changeowneruri, objChangeToken)
	changeOwnerResponse, err := clients[clientid].Client.Do(changeOwnerRequest)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 400, changeOwnerResponse, "Bad status when changing owner")
	defer util.FinishBody(changeOwnerResponse.Body)
}
