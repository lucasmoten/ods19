package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

// TestChangeOwner validates that change owner is implemented, ownership changes, and parent set to root
func TestChangeOwner(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	clientid := 0
	originalOwner := "user/" + fakeDN0
	newOwner := "user/" + fakeDN1

	t.Logf("* Creating 2 folders under root")
	folder1 := makeFolderViaJSON("Test ChangeOwner Folder 1 ", clientid, t)
	folder2 := makeFolderViaJSON("Test ChangeOwner Folder 2 ", clientid, t)
	t.Logf("* Verifying owner of both folders as %s", originalOwner)
	if folder1.OwnedBy != originalOwner {
		t.Logf("Owner for folder1 is %s expected %s", folder1.OwnedBy, originalOwner)
		t.FailNow()
	}
	if folder2.OwnedBy != originalOwner {
		t.Logf("Owner for folder2 is %s expected %s", folder2.OwnedBy, originalOwner)
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
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
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
	t.Logf("* Verifying owner of moved folder is still %s", originalOwner)
	if updatedFolder.OwnedBy != originalOwner {
		t.Logf("Owner for folder2 is %s expected %s", updatedFolder.OwnedBy, originalOwner)
		t.FailNow()
	}

	t.Logf("* Changing owner of folder 2 to %s", newOwner)
	changeowneruri := host + config.NginxRootURL + "/objects/" + folder2.ID + "/owner/" + newOwner
	objChangeToken.ChangeToken = updatedFolder.ChangeToken
	changeOwnerRequest := makeHTTPRequestFromInterface(t, "POST", changeowneruri, objChangeToken)
	changeOwnerResponse, err := clients[clientid].Client.Do(changeOwnerRequest)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, changeOwnerResponse, "Bad status when changing owner")
	var updatedObject protocol.Object
	err = util.FullDecode(changeOwnerResponse.Body, &updatedObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Verifying owner changed")
	if updatedObject.OwnedBy != newOwner {
		t.Logf("Owner for folder2 is %s expected %s", updatedObject.OwnedBy, newOwner)
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
	newownernodisplayname := "group/dctc/dctc/odrive_g1"
	changeowneruri := host + config.NginxRootURL + "/objects/" + createdObject.ID + "/owner/" + newowner
	objChangeToken := protocol.ChangeTokenStruct{ChangeToken: createdObject.ChangeToken}
	changeOwnerRequest := makeHTTPRequestFromInterface(t, "POST", changeowneruri, objChangeToken)
	trafficLogs[APISampleFile].Request(t, changeOwnerRequest,
		&TrafficLogDescription{
			OperationName:       "Change Owner for Object",
			RequestDescription:  "Transfer object ownership to a different user or group",
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

func TestChangeOwnerToGroupWithClient(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	clientid := 0

	t.Logf("* Creating a private object")
	createObjectRequest := protocol.CreateObjectRequest{}
	createObjectRequest.Name = "Test ChangeOwner to Group"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = testhelpers.ValidACMUnclassifiedFOUOSharedToTester10
	createdObject, err := clients[clientid].C.CreateObject(createObjectRequest, nil)
	if err != nil {
		t.Logf("Error creating object: %v", err)
		t.FailNow()
	}

	t.Logf("* Changing ownership to group")
	newowner := "group/dctc/dctc/odrive_g1"
	changeOwnerRequest := protocol.ChangeOwnerRequest{}
	changeOwnerRequest.ID = createdObject.ID
	changeOwnerRequest.ChangeToken = createdObject.ChangeToken
	changeOwnerRequest.NewOwner = newowner
	changedObject, err := clients[clientid].C.ChangeOwner(changeOwnerRequest)
	if err != nil {
		t.Logf("Error changing owner: %v", err)
		t.FailNow()
	}
	if changedObject.OwnedBy != newowner {
		t.Logf("Owner of changed object is '%s', expected '%s'", changedObject.OwnedBy, newowner)
		t.Fail()
	}
}

func TestChangeOwnerToNonCachedUser(t *testing.T) {
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

	t.Logf("* Changing ownership to another user")
	rand.Seed(time.Now().UTC().UnixNano())
	randomnum := rand.Int()
	newowner := fmt.Sprintf("user/cn=uncached user%6d,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us", randomnum)
	changeowneruri := host + config.NginxRootURL + "/objects/" + createdObject.ID + "/owner/" + newowner
	objChangeToken := protocol.ChangeTokenStruct{ChangeToken: createdObject.ChangeToken}
	changeOwnerRequest := makeHTTPRequestFromInterface(t, "POST", changeowneruri, objChangeToken)
	changeOwnerResponse, err := clients[clientid].Client.Do(changeOwnerRequest)
	failNowOnErr(t, err, "Unable to do request")
	if changeOwnerResponse.StatusCode != http.StatusOK {
		t.Fail()
		buf := new(bytes.Buffer)
		buf.ReadFrom(changeOwnerResponse.Body)
		bodytext := buf.String()
		t.Logf("Failed to change owner to %s", newowner)
		t.Logf("Response status = %d, body follows", changeOwnerResponse.StatusCode)
		t.Logf(bodytext)
		ioutil.ReadAll(changeOwnerResponse.Body)
		t.FailNow()
	}
	statusMustBe(t, 200, changeOwnerResponse, "Bad status when changing owner")
	var updatedObject protocol.Object
	err = util.FullDecode(changeOwnerResponse.Body, &updatedObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Verifying owner changed")
	if updatedObject.OwnedBy != newowner {
		t.Logf("Owner for folder2 is %s expected %s", updatedObject.OwnedBy, newowner)
		t.FailNow()
	}
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
	statusMustBe(t, http.StatusPreconditionRequired, changeOwnerResponse, "Bad status when changing owner")
	defer util.FinishBody(changeOwnerResponse.Body)
}

func TestChangeOwnerRecursive(t *testing.T) {
	randomName := func(name string) string {
		s, _ := util.NewGUID()
		return name + s
	}
	clientid := 0
	root, child1, child2, child3 := randomName("root"), randomName("child1"), randomName("child2"), randomName("child3")

	t.Logf("Create object hierarchy:\n root: %s\n child1: %s\n child2: %s\n child3: %s\n", root, child1, child2, child3)
	cor := protocol.CreateObjectRequest{
		NamePathDelimiter: ":::",
		Name:              strings.Join([]string{root, child1, child2, child3}, ":::"),
		RawAcm:            testhelpers.ValidACMUnclassifiedFOUOSharedToTester10,
	}
	child3Obj, err := clients[clientid].C.CreateObject(cor, bytes.NewBuffer([]byte("testvalue")))
	failNowOnErr(t, err, "unable to do request")

	// We want to change child1 owner. Get at child1 by walking up ParentIDs.
	child2Obj, err := clients[clientid].C.GetObject(child3Obj.ParentID)
	failNowOnErr(t, err, "unable to do get child2")
	child1Obj, err := clients[clientid].C.GetObject(child2Obj.ParentID)
	failNowOnErr(t, err, "unable to do get child1")

	// tester01 try to get child1Obj; We expect failure, but need to make a
	// request to guarantee that the user exists.
	_, err = clients[1].C.GetObject(child1Obj.ID)
	if err != nil {
		t.Logf("Expected error: %v", err)
	}

	// ChangeOwnerRequest for child1 with ApplyRecursively true.
	newOwner := "user/" + fakeDN1
	chor := protocol.ChangeOwnerRequest{
		ChangeToken:      child1Obj.ChangeToken,
		ApplyRecursively: true,
		ID:               child1Obj.ID,
		NewOwner:         newOwner,
	}
	t.Logf("ChangeOwnerRequest: %v\n", chor)

	_, err = clients[clientid].C.ChangeOwner(chor)
	if err != nil {
		t.Errorf("change owner did not succeed: %v", err)
		t.FailNow()
	}

	tries := 0
	for {
		// We must retry because this is an async operation on the server.
		final, err := clients[1].C.GetObject(child3Obj.ID)
		if err != nil {
			if tries < 50 {
				tries++
				t.Logf("Sleeping 50 ms. Tries %v", tries)
				time.Sleep(50 * time.Millisecond)
				continue
			}
			t.Errorf("GetObject should succeed for tester1 on child3: %v", err)
			t.FailNow()
		}
		if newOwner != final.OwnedBy {
			t.Errorf("expected %s got %s", newOwner, final.OwnedBy)
			t.FailNow()
		}
		if final.ChangeCount != 1 {
			t.Errorf("expected exactly 1 update; got: %v", final.ChangeCount)
			t.FailNow()
		}
		break
	}

	t.Log("Attempting to retrieve stream as tester1")
	stream, err := clients[1].C.GetObjectStream(child3Obj.ID)
	if err != nil {
		t.Errorf("getting object stream failed: %v", err)
		t.FailNow()
	}

	data, err := ioutil.ReadAll(stream)
	if err != nil {
		t.Errorf("could not read returned stream: %v", err)
		t.FailNow()
	}
	t.Log("Stream data received:")
	t.Log(string(data))
	if string(data) != "testvalue" {
		t.Errorf("expected stream to contain 'testvalue', but got: %s", string(data))
	}

}
