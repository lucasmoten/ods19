package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

// TestAcmWithoutShare - User T1 creates object O1 with ACM having no share.
// Verify T1..T10, and another known DN have access to read the object
// (by virtue of it shared to everyone)
func TestAcmWithoutShare(t *testing.T) {

	t.Logf("* Create object O1 as tester1")
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O1"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"]}`
	createObjectRequest.ContentSize = 0
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := mountPoint + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateResponse, err := clients[tester1].Client.Do(httpCreate)

	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, 200, httpCreateResponse, "Bad status when creating object")

	var createdObject protocol.Object

	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object: %v")

	// ### Verify all clients can read it
	testers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	shouldHaveReadForObjectID(t, createdObject.ID, testers...)
	shouldNotHaveReadForObjectID(t, createdObject.ID, 10)
}

// TestAcmWithShareForODrive - User T1 creates object O2 with ACM having share
// for group ODrive. Verify T1..T10 but no other DNs have access since only
// T1..T10 are in that group
func TestAcmWithShareForODrive(t *testing.T) {

	// ### Create object O2 as tester1
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O2"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive"]}}}}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := mountPoint + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateResponse, err := clients[tester1].Client.Do(httpCreate)
	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, 200, httpCreateResponse, "Bad status when creating object")

	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	shouldHaveReadForObjectID(t, createdObject.ID, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	shouldNotHaveReadForObjectID(t, createdObject.ID, 10)
}

// TestAcmWithShareCreatorIsNotInWillForceThemIntoShare - User T1 creates
// object O3 with ACM having share for group ODrive G1. T1 is not in this
// group and AAC would reject it on its own. Verify that the object is still
// created and T1 can read it, and that the resultant share includes
// group ODrive G1 as well as user T1
func TestAcmWithShareCreatorIsNotInWillForceThemIntoShare(t *testing.T) {
	t.Logf("### Create object O3 as tester1")
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O3"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive_G1"]}}}}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	t.Logf("prep http request")
	uriCreate := mountPoint + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	t.Logf("exec and get response")
	httpCreateResponse, err := clients[tester1].Client.Do(httpCreate)
	failNowOnErr(t, err, "unable to do request")
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, 200, httpCreateResponse, "Bad status when creating object")

	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")
	t.Logf("check permissions")
	for _, p := range createdObject.Permissions {
		t.Logf("%s", p)
	}

	shouldHaveReadForObjectID(t, createdObject.ID, 1, 6, 7, 8, 9, 0)
	shouldNotHaveReadForObjectID(t, createdObject.ID, 2, 3, 4, 5, 10)

}

// TestAcmWithShareForODriveG1Allowed - User T10 creates object O4 with ACM
// having share for group ODrive G1. Verify that this is created, and
// accessible by T6..T10, but not T1..T5 or another DN
func TestAcmWithShareForODriveG1Allowed(t *testing.T) {
	// ### Create object O4 as tester10
	tester10 := 0
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O4"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive_G1"]}}}}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := mountPoint + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateResponse, err := clients[tester10].Client.Do(httpCreate)
	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, 200, httpCreateResponse, "Bad status when creating object")

	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	failNowOnErr(t, err, "error decoding json to Object")

	// ### Verify tester 6-10 can read it, but not 1-5 or other certs
	shouldHaveReadForObjectID(t, createdObject.ID, 6, 7, 8, 9, 0)
	shouldNotHaveReadForObjectID(t, createdObject.ID, 1, 2, 3, 4, 5)
}

// TestAcmWithShareForODriveG2Allowed - User T1 creates object O5 with ACM
// having share for group ODrive G2. Verify that this is created, and
// accessible by T1..T5, but not T6..T10 or another DN
func TestAcmWithShareForODriveG2Allowed(t *testing.T) {
	// ### Create object O5 as tester1
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O5"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive_G2"]}}}}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := mountPoint + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateResponse, err := clients[tester1].Client.Do(httpCreate)
	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, 200, httpCreateResponse, "Bad status when creating object")

	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	failNowOnErr(t, err, "error decoding json to Object")

	// ### Verify tester 1-5 can read it, but not 6-10 or other certs
	shouldHaveReadForObjectID(t, createdObject.ID, 1, 2, 3, 4, 5)
	shouldNotHaveReadForObjectID(t, createdObject.ID, 6, 7, 8, 9, 0)

}

// TestAddReadShareForUser - User T1 creates object O7 with ACM having share
// for group ODrive G2. This is created and accessible by T1..T5, but not by
// T6..T10 or other users not in the group. Then add share to T10 allowRead
// and verify that T10 is then able to read it.
func TestAddReadShareForUser(t *testing.T) {
	t.Logf("TestAddReadShareForUser - ### Create object O7 as tester1")
	tester1 := 1
	t.Logf("TestAddReadShareForUser - prep object")
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O7"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive_G2"]}}}}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := mountPoint + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	t.Logf("TestAddReadShareForUser - exec and get response")
	httpCreateResponse, err := clients[tester1].Client.Do(httpCreate)
	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, 200, httpCreateResponse, "Bad status when creating object")

	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	failNowOnErr(t, err, "error decoding json to Object")

	t.Logf("TestAddReadShareForUser - ### Add a share for tester 10 to be able to read the object")
	// prep share
	var createShareRequest protocol.ObjectShare
	createShareRequest.AllowRead = true
	createShareRequest.Share = makeUserShare("cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us")
	// jsonify it
	jsonBody, _ = json.Marshal(createShareRequest)
	t.Logf("TestAddReadShareForUser - prep http request")
	uriShare := mountPoint + "/shared/" + createdObject.ID
	httpCreateShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateShare.Header.Set("Content-Type", "application/json")
	t.Logf("TestAddReadShareForUser - exec and get response")
	httpCreateShareResponse, err := clients[tester1].Client.Do(httpCreateShare)
	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, 200, httpCreateShareResponse, "Bad status when creating share")

	var updatedObject protocol.Object
	err = util.FullDecode(httpCreateShareResponse.Body, &updatedObject)
	failNowOnErr(t, err, "error decoding json to Object")

	t.Logf("TestAddReadShareForUser - ### Verify tester 1-5 can read it, as well as 10, but not 6-9 or other certs")
	shouldHaveReadForObjectID(t, createdObject.ID, 1, 2, 3, 4, 5, 0)
	shouldNotHaveReadForObjectID(t, createdObject.ID, 6, 7, 8, 9)
}

// TestAddReadAndUpdateShareForUser - User T1 creates object O8 with ACM having
// share for group ODrive G2. This is created and accessible by T1..T5, but not
// by T6..T10 or other users not in the group. Then add share to T10 allowRead
// and verify that T10 is then able to read it, but T6..T9 still cannot. Next,
// add share to group G1 allowRead, allowUpdate. Verify that T1..T10 can read.
// Verify that T9 can update it by changing the name
func TestAddReadAndUpdateShareForUser(t *testing.T) {
	// ### Create object O8 as tester1
	t.Logf("Creating object O8 as tester1 with ACM share for G2")
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O8"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive_G2"]}}}}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := mountPoint + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")

	// exec and get response
	httpCreateResponse, err := clients[tester1].Client.Do(httpCreate)
	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, 200, httpCreateResponse, "Bad status when creating object")

	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	failNowOnErr(t, err, "error decoding json to Object")

	// ### Add a share for tester 10 to be able to read the object
	t.Logf("Adding share for tester 10 for read")
	// prep share
	var createShareRequest protocol.ObjectShare
	createShareRequest.AllowRead = true
	createShareRequest.Share = makeUserShare("cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us")
	// jsonify it
	jsonBody, _ = json.Marshal(createShareRequest)
	// prep http request
	uriShare := mountPoint + "/shared/" + createdObject.ID
	// prep http request
	httpCreateShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateShare.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateShareResponse, err := clients[tester1].Client.Do(httpCreateShare)
	failNowOnErr(t, err, "unable to do create share request")
	defer util.FinishBody(httpCreateShareResponse.Body)
	statusMustBe(t, 200, httpCreateShareResponse, "bad status when creating share")

	// parse back to object
	var updatedObject protocol.Object
	err = util.FullDecode(httpCreateShareResponse.Body, &updatedObject)
	failNowOnErr(t, err, "error decoding json to Object")

	// ### Verify tester 1-5 can read it, as well as 10, but not 6-9 or other certs
	t.Logf("Verify 1-5 can read, as well as 10, but not 6-9")
	shouldHaveReadForObjectID(t, createdObject.ID, 1, 2, 3, 4, 5, 0)
	shouldNotHaveReadForObjectID(t, createdObject.ID, 6, 7, 8, 9)

	// ### Add a share for G1 group to allow reading and updating
	t.Logf("Adding share for G1 for read and update")
	// prep share
	var createGroupShareRequest protocol.ObjectShare
	createGroupShareRequest.AllowRead = true
	createGroupShareRequest.AllowUpdate = true
	createGroupShareRequest.Share = makeGroupShare("DCTC", "DCTC", "ODrive_G1")
	// jsonify it
	jsonBody, _ = json.Marshal(createGroupShareRequest)
	// prep http request
	httpCreateGroupShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateGroupShare.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateGroupShareResponse, err := clients[tester1].Client.Do(httpCreateGroupShare)
	failNowOnErr(t, err, "unable to do request")
	defer util.FinishBody(httpCreateShareResponse.Body)
	// check status of response
	statusMustBe(t, 200, httpCreateGroupShareResponse, "Bad status when creating share")

	// parse back to object
	var updatedObject2 protocol.Object
	err = util.FullDecode(httpCreateGroupShareResponse.Body, &updatedObject2)
	failNowOnErr(t, err, "error decoding json to Object")

	// ### Verify tester 1-10 can read it, but not others
	shouldHaveReadForObjectID(t, updatedObject2.ID, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0)
	shouldNotHaveReadForObjectID(t, updatedObject2.ID, 10)
	t.Logf("Verify 1-0 can read, but not others")

	// ### Verify that Tester 9 can now update it
	t.Logf("Verify tester9 can update")
	tester9 := 9
	updatedObject2.Name += " changed by Tester09"
	uriUpdate := mountPoint + "/objects/" + updatedObject2.ID + "/properties"
	// jsonify it
	jsonBody, _ = json.Marshal(updatedObject2)
	// prep http request
	httpUpdateObject, _ := http.NewRequest("POST", uriUpdate, bytes.NewBuffer(jsonBody))
	httpUpdateObject.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpUpdateObjectResponse, err := clients[tester9].Client.Do(httpUpdateObject)
	failNowOnErr(t, err, "unable to do request")
	defer util.FinishBody(httpUpdateObjectResponse.Body)
	// check status of response
	statusMustBe(t, 200, httpUpdateObjectResponse, "bad status when updating object")
}

// TestAddReadShareForGroupRemovesEveryone - User T1 creates object O9 with
// ACM having no share. Verify T1..T10, and another known DN have access to
// read the object. User T1 Adds Share with read permission for group
// ODrive G2 to O9. The existing share to everyone should
// be revoked. T1..T5 should have read access. T1 retains create, delete,
// update, and share access. T6..T10 should no longer see the object as its not
// shared to everyone.
func TestAddReadShareForGroupRemovesEveryone(t *testing.T) {

	t.Logf("* Create object O9 as tester1")
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O9"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"]}`
	createObjectRequest.ContentSize = 0
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := mountPoint + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateResponse, err := clients[tester1].Client.Do(httpCreate)
	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, 200, httpCreateResponse, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	failNowOnErr(t, err, "error decoding json to Object")

	t.Logf("* Verify all clients can read it")
	uriGetProperties := mountPoint + "/objects/" + createdObject.ID + "/properties"
	httpGet, _ := http.NewRequest("GET", uriGetProperties, nil)
	shouldHaveReadForObjectID(t, createdObject.ID, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	shouldNotHaveReadForObjectID(t, createdObject.ID, 10)
	shouldHaveEveryonePermission(t, createdObject.ID, 1)

	t.Logf("* User T1 Adds Share with read permission for group ODrive G2 to O9.")
	// prep share
	var createGroupShareRequest protocol.ObjectShare
	createGroupShareRequest.AllowRead = true
	createGroupShareRequest.AllowUpdate = true
	createGroupShareRequest.Share = makeGroupShare("DCTC", "DCTC", "ODrive_G2")
	// jsonify it
	jsonBody, _ = json.Marshal(createGroupShareRequest)
	// prep http request
	uriShare := mountPoint + "/shared/" + createdObject.ID
	httpCreateGroupShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateGroupShare.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateGroupShareResponse, err := clients[tester1].Client.Do(httpCreateGroupShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpCreateGroupShareResponse.Body)
	// check status of response
	if httpCreateGroupShareResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating share: %s", httpCreateGroupShareResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject2 protocol.Object
	err = util.FullDecode(httpCreateGroupShareResponse.Body, &updatedObject2)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify 1-5 can read, but not others since Everyone removed")
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
		switch clientIdx {
		case 1, 2, 3, 4, 5:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // twl-server-generic and any others that may get added later
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		if clientIdx == len(clients)-1 {
			t.Logf("* Resulting permissions")
			hasEveryone := false
			for _, permission := range updatedObject2.Permissions {
				t.Logf("%s", permission)
				if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
					hasEveryone = true
				}
				if strings.Contains(permission.Grantee, "cn=test tester01,") {
					if !permission.AllowCreate ||
						!permission.AllowUpdate ||
						!permission.AllowDelete ||
						!permission.AllowShare {
						t.Logf("Expected tester1 to have Create, Update, Delete, and Share")
						t.Fail()
					}
				}
			}
			if hasEveryone {
				t.Logf("Expected %s to have been removed", models.EveryoneGroup)
				t.Fail()
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

}

// TestAddReadShareToUserWithoutEveryone - User T1 creates object O10 with
// ACM having no share. Verify T1..T10, and another known DN have access to
// read the object. User T1 Adds Share with read permission for group
// ODrive G2 to O10. The existing share to everyone should
// be revoked. T1..T5 should have read access. T1 retains create, delete,
// update, and share access. T6..T10 should no longer see the object as its not
// shared to everyone.  User T1 Adds Share with read permission for user T10 to
// O10. T1 retains full CRUDS, T1..T5 retains read access from the prior share
// established and T10 should now get read access.
func TestAddReadShareToUserWithoutEveryone(t *testing.T) {
	t.Logf("* Create object O10 as tester1")
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O10"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"]}`
	createObjectRequest.ContentSize = 0
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := mountPoint + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateResponse, err := clients[tester1].Client.Do(httpCreate)
	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, 200, httpCreateResponse, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify all clients can read it")
	uriGetProperties := mountPoint + "/objects/" + createdObject.ID + "/properties"
	httpGet, _ := http.NewRequest("GET", uriGetProperties, nil)
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
		if clientIdx < 10 {
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d. Status was %s", clientIdx, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
			if clientIdx == len(clients)-1 {
				var retrievedObject protocol.Object
				err = util.FullDecode(httpGetResponse.Body, &retrievedObject)
				if err != nil {
					t.Logf("Error decoding json to Object: %v", err)
					t.FailNow()
				}
				t.Logf("* Resulting permissions")
				hasEveryone := false
				for _, permission := range retrievedObject.Permissions {
					t.Logf("%s", permission)
					if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
						hasEveryone = true
					}
				}
				if !hasEveryone {
					t.Logf("Missing %s", models.EveryoneGroup)
					t.FailNow()
				}
			} else {
				ioutil.ReadAll(httpGetResponse.Body)
			}
		} else {
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d. Status was %s", clientIdx, httpGetResponse.Status)
				t.Fail()
			}
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

	t.Logf("* User T1 Adds Share with read permission for group ODrive G2 to O10.")
	// prep share
	var createGroupShareRequest protocol.ObjectShare
	createGroupShareRequest.AllowRead = true
	createGroupShareRequest.AllowUpdate = true
	createGroupShareRequest.Share = makeGroupShare("DCTC", "DCTC", "ODrive_G2")
	// jsonify it
	jsonBody, _ = json.Marshal(createGroupShareRequest)
	// prep http request
	uriShare := mountPoint + "/shared/" + createdObject.ID
	httpCreateGroupShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateGroupShare.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateGroupShareResponse, err := clients[tester1].Client.Do(httpCreateGroupShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpCreateGroupShareResponse.Body)
	// check status of response
	if httpCreateGroupShareResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating share: %s", httpCreateGroupShareResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject2 protocol.Object
	err = util.FullDecode(httpCreateGroupShareResponse.Body, &updatedObject2)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify 1-5 can read, but not others since Everyone removed")
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
		switch clientIdx {
		case 1, 2, 3, 4, 5:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // twl-server-generic and any others that may get added later
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		if clientIdx == len(clients)-1 {
			t.Logf("* Resulting permissions")
			hasEveryone := false
			for _, permission := range updatedObject2.Permissions {
				t.Logf("%s", permission)
				if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
					hasEveryone = true
				}
				if strings.Contains(permission.Grantee, "cn=test tester01,") {
					if !permission.AllowCreate ||
						!permission.AllowUpdate ||
						!permission.AllowDelete ||
						!permission.AllowShare {
						t.Logf("Expected tester1 to have Create, Update, Delete, and Share")
						t.Fail()
					}
				}
			}
			if hasEveryone {
				t.Logf("Expected %s to have been removed", models.EveryoneGroup)
				t.Fail()
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

	t.Logf("* User T1 Adds Share with read permission for user tester10 to O10.")
	// prep share
	var createUserShareRequest protocol.ObjectShare
	createUserShareRequest.AllowRead = true
	createUserShareRequest.Share = makeUserShare(fakeDN0)
	// jsonify it
	jsonBody, _ = json.Marshal(createUserShareRequest)
	// prep http request
	httpCreateUserShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateUserShare.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateUserShareResponse, err := clients[tester1].Client.Do(httpCreateUserShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpCreateUserShareResponse.Body)
	// check status of response
	if httpCreateUserShareResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating share: %s", httpCreateUserShareResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject3 protocol.Object
	err = util.FullDecode(httpCreateUserShareResponse.Body, &updatedObject3)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify 1-5, 10 can read, but not others since Everyone removed")
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
		switch clientIdx {
		case 0, 1, 2, 3, 4, 5:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // twl-server-generic and any others that may get added later
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		if clientIdx == len(clients)-1 {
			t.Logf("* Resulting permissions")
			hasEveryone := false
			for _, permission := range updatedObject3.Permissions {
				t.Logf("%s", permission)
				if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
					hasEveryone = true
				}
				if strings.Contains(permission.Grantee, "cn=test tester01,") {
					if !permission.AllowCreate ||
						!permission.AllowUpdate ||
						!permission.AllowDelete ||
						!permission.AllowShare {
						t.Logf("Expected tester1 to have Create, Update, Delete, and Share")
						t.Fail()
					}
				}
				if strings.Contains(permission.Grantee, "tester10") {
					if !permission.AllowRead {
						t.Logf("Expected tester10 to have Read")
						t.Fail()
					}
				}
			}
			if hasEveryone {
				t.Logf("Expected %s to have been removed", models.EveryoneGroup)
				t.Fail()
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

}

// TestUpdateAcmWithoutSharingToUser - User T1 creates object O10 with
// ACM having no share. Verify T1..T10, and another known DN have access to
// read the object. User T1 Adds Share with read permission for group
// ODrive G2 to O10. The existing share to everyone should
// be revoked. T1..T5 should have read access. T1 retains create, delete,
// update, and share access. T6..T10 should no longer see the object as its not
// shared to everyone.  User T1 Adds Share with read permission for user T10 to
// O10. T1 retains full CRUDS, T1..T5 retains read access from the prior share
// established and T10 should now get read access. -> User T1 Updates O1
// setting an ACM that has a share for G2. T1 retains full CRUDS, T1..T5
// retains read access from the share as it remains in place, and T10 should
// lose access as the read permission should be marked deleted since ACM
// overrides.
func TestUpdateAcmWithoutSharingToUser(t *testing.T) {
	t.Logf("* Create object O11 as tester1")
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O11"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"]}`
	createObjectRequest.ContentSize = 0
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := mountPoint + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateResponse, err := clients[tester1].Client.Do(httpCreate)
	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, 200, httpCreateResponse, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify all clients can read it")
	uriGetProperties := mountPoint + "/objects/" + createdObject.ID + "/properties"
	httpGet, _ := http.NewRequest("GET", uriGetProperties, nil)
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
		if clientIdx < 10 {
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d. Status was %s", clientIdx, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		} else {
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d. Status was %s", clientIdx, httpGetResponse.Status)
				t.Fail()
			}
		}
		if clientIdx == 0 {
			var retrievedObject protocol.Object
			err = util.FullDecode(httpGetResponse.Body, &retrievedObject)
			if err != nil {
				t.Logf("Error decoding json to Object: %v", err)
				t.FailNow()
			}
			t.Logf("* Resulting permissions")
			hasEveryone := false
			for _, permission := range retrievedObject.Permissions {
				t.Logf("%s", permission)
				if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
					hasEveryone = true
				}
			}
			if !hasEveryone {
				t.Logf("Missing %s", models.EveryoneGroup)
				t.FailNow()
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

	t.Logf("* User T1 Adds Share with read permission for group ODrive G2 to O11.")
	// prep share
	var createGroupShareRequest protocol.ObjectShare
	createGroupShareRequest.AllowRead = true
	createGroupShareRequest.AllowUpdate = true
	createGroupShareRequest.Share = makeGroupShare("DCTC", "DCTC", "ODrive_G2")
	// jsonify it
	jsonBody, _ = json.Marshal(createGroupShareRequest)
	// prep http request
	uriShare := mountPoint + "/shared/" + createdObject.ID
	httpCreateGroupShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateGroupShare.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateGroupShareResponse, err := clients[tester1].Client.Do(httpCreateGroupShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpCreateGroupShareResponse.Body)
	// check status of response
	if httpCreateGroupShareResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating share: %s", httpCreateGroupShareResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject2 protocol.Object
	err = util.FullDecode(httpCreateGroupShareResponse.Body, &updatedObject2)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify 1-5 can read, but not others since Everyone removed")
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
		switch clientIdx {
		case 1, 2, 3, 4, 5:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // twl-server-generic and any others that may get added later
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		if clientIdx == len(clients)-1 {
			t.Logf("* Resulting permissions")
			hasEveryone := false
			for _, permission := range updatedObject2.Permissions {
				t.Logf("%s", permission)
				if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
					hasEveryone = true
				}
				if strings.Contains(permission.Grantee, "cn=test tester01,") {
					if !permission.AllowCreate ||
						!permission.AllowUpdate ||
						!permission.AllowDelete ||
						!permission.AllowShare {
						t.Logf("Expected tester1 to have Create, Update, Delete, and Share")
						t.Fail()
					}
				}
			}
			if hasEveryone {
				t.Logf("Expected %s to have been removed", models.EveryoneGroup)
				t.Fail()
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

	t.Logf("* User T1 Adds Share with read permission for user tester10 to O11.")
	// prep share
	var createUserShareRequest protocol.ObjectShare
	createUserShareRequest.AllowRead = true
	createUserShareRequest.Share = makeUserShare(fakeDN0)
	// jsonify it
	jsonBody, _ = json.Marshal(createUserShareRequest)
	// prep http request
	httpCreateUserShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateUserShare.Header.Set("Content-Type", "application/json")
	// exec and get response
	// TODO figure out if this is really tester1
	httpCreateUserShareResponse, err := clients[tester1].Client.Do(httpCreateUserShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpCreateUserShareResponse.Body)
	// check status of response
	if httpCreateUserShareResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating share: %s", httpCreateUserShareResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject3 protocol.Object
	err = util.FullDecode(httpCreateUserShareResponse.Body, &updatedObject3)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify 1-5, 10 can read, but not others since Everyone removed")
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
		switch clientIdx {
		case 0, 1, 2, 3, 4, 5:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // twl-server-generic and any others that may get added later
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		if clientIdx == len(clients)-1 {
			t.Logf("* Resulting permissions")
			hasEveryone := false
			for _, permission := range updatedObject3.Permissions {
				t.Logf("%s", permission)
				if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
					hasEveryone = true
				}
				if strings.Contains(permission.Grantee, "cn=test tester01,") {
					if !permission.AllowCreate ||
						!permission.AllowUpdate ||
						!permission.AllowDelete ||
						!permission.AllowShare {
						t.Logf("Expected tester1 to have Create, Update, Delete, and Share")
						t.Fail()
					}
				}
				if strings.Contains(permission.Grantee, "tester10") {
					if !permission.AllowRead {
						t.Logf("Expected tester10 to have Read")
						t.Fail()
					}
				}
			}
			if hasEveryone {
				t.Logf("Expected %s to have been removed", models.EveryoneGroup)
				t.Fail()
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

	t.Logf("Update O11 setting an ACM sharing to ODrive G2, which will revoke read from T10")
	acmWithODriveG1 := `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive_G2"]}}}}`
	updatedObject3.RawAcm = acmWithODriveG1
	updatedObject3.Permission = protocol.Permission{}
	uriUpdate := mountPoint + "/objects/" + createdObject.ID + "/properties"
	// jsonify it
	jsonBody, _ = json.Marshal(updatedObject3)
	// prep http request
	httpUpdateObject, _ := http.NewRequest("POST", uriUpdate, bytes.NewBuffer(jsonBody))
	httpUpdateObject.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpUpdateObjectResponse, err := clients[tester1].Client.Do(httpUpdateObject)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpUpdateObjectResponse.Body)
	// check status of response
	if httpUpdateObjectResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when updating object: %s", httpUpdateObjectResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject4 protocol.Object
	err = util.FullDecode(httpUpdateObjectResponse.Body, &updatedObject4)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify 1-5 can read, but nobody else")
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
		switch clientIdx {
		case 1, 2, 3, 4, 5:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // twl-server-generic and any others that may get added later
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		if clientIdx == len(clients)-1 {
			t.Logf("* Resulting permissions")
			hasEveryone := false
			for _, permission := range updatedObject4.Permissions {
				t.Logf("%s", permission)
				if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
					hasEveryone = true
				}
				if strings.Contains(permission.Grantee, "cn=test tester01,") {
					if !permission.AllowCreate ||
						!permission.AllowUpdate ||
						!permission.AllowDelete ||
						!permission.AllowShare {
						t.Logf("Expected tester1 to have Create, Update, Delete, and Share")
						t.Fail()
					}
				}
				if strings.Contains(permission.Grantee, "tester10") {
					t.Logf("Expected tester10 permission to have been removed")
					t.Fail()
				}
			}
			if hasEveryone {
				t.Logf("Expected %s to have been removed", models.EveryoneGroup)
				t.Fail()
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

}

// TestUpdateAcmWithoutAnyShare - User T1 creates object O10 with
// ACM having no share. Verify T1..T10, and another known DN have access to
// read the object. User T1 Adds Share with read permission for group
// ODrive G2 to O10. The existing share to everyone should
// be revoked. T1..T5 should have read access. T1 retains create, delete,
// update, and share access. T6..T10 should no longer see the object as its not
// shared to everyone.  User T1 Adds Share with read permission for user T10 to
// O10. T1 retains full CRUDS, T1..T5 retains read access from the prior share
// established and T10 should now get read access. User T1 Updates O1
// setting an ACM that has a share for O1. T1 retains full CRUDS, T1..T5
// retains read access from the share as it remains in place, and T10 should
// lose access as the read permission should be marked deleted since ACM
// overrides. -> User T1 Updates O1 setting an ACM that has an empty share. T1
// retains full CRUDS. Share to Odrive G2 in ACM is removed as is permission.
// Permission to EveryoneGroup established. T1..T10 have read access. Any other
// recognized DN also has read access
func TestUpdateAcmWithoutAnyShare(t *testing.T) {
	t.Logf("* Create object O12 as tester1")
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O12"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"]}`
	createObjectRequest.ContentSize = 0
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := mountPoint + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateResponse, err := clients[tester1].Client.Do(httpCreate)
	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, 200, httpCreateResponse, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify all clients can read it")
	uriGetProperties := mountPoint + "/objects/" + createdObject.ID + "/properties"
	httpGet, _ := http.NewRequest("GET", uriGetProperties, nil)
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
		if clientIdx < 10 {
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d. Status was %s", clientIdx, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		} else {
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d. Status was %s", clientIdx, httpGetResponse.Status)
				t.Fail()
			}
		}
		if clientIdx == 0 {
			var retrievedObject protocol.Object
			err = util.FullDecode(httpGetResponse.Body, &retrievedObject)
			if err != nil {
				t.Logf("Error decoding json to Object: %v", err)
				t.FailNow()
			}
			t.Logf("* Resulting permissions")
			hasEveryone := false
			for _, permission := range retrievedObject.Permissions {
				t.Logf("%s", permission)
				if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
					hasEveryone = true
				}
			}
			if !hasEveryone {
				t.Logf("Missing %s", models.EveryoneGroup)
				t.FailNow()
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

	t.Logf("* User T1 Adds Share with read permission for group ODrive G2 to O12.")
	// prep share
	var createGroupShareRequest protocol.ObjectShare
	createGroupShareRequest.AllowRead = true
	//createGroupShareRequest.AllowUpdate = true
	createGroupShareRequest.Share = makeGroupShare("DCTC", "DCTC", "ODrive_G2")
	// jsonify it
	jsonBody, _ = json.Marshal(createGroupShareRequest)
	// prep http request
	uriShare := mountPoint + "/shared/" + createdObject.ID
	httpCreateGroupShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateGroupShare.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateGroupShareResponse, err := clients[tester1].Client.Do(httpCreateGroupShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpCreateGroupShareResponse.Body)
	// check status of response
	if httpCreateGroupShareResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating share: %s", httpCreateGroupShareResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject2 protocol.Object
	err = util.FullDecode(httpCreateGroupShareResponse.Body, &updatedObject2)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify 1-5 can read, but not others since Everyone removed")
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
		switch clientIdx {
		case 1, 2, 3, 4, 5:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // twl-server-generic and any others that may get added later
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		if clientIdx == len(clients)-1 {
			t.Logf("* Resulting permissions")
			hasEveryone := false
			for _, permission := range updatedObject2.Permissions {
				t.Logf("%s", permission)
				if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
					hasEveryone = true
				}
				if strings.Contains(permission.Grantee, "cn=test tester01,") {
					if !permission.AllowCreate ||
						!permission.AllowUpdate ||
						!permission.AllowDelete ||
						!permission.AllowShare {
						t.Logf("Expected tester1 to have Create, Update, Delete, and Share")
						t.Fail()
					}
				}
			}
			if hasEveryone {
				t.Logf("Expected %s to have been removed", models.EveryoneGroup)
				t.Fail()
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

	t.Logf("* User T1 Adds Share with read permission for user tester10 to O12.")
	// prep share
	var createUserShareRequest protocol.ObjectShare
	createUserShareRequest.AllowRead = true
	createUserShareRequest.Share = makeUserShare(fakeDN0)
	// jsonify it
	jsonBody, _ = json.Marshal(createUserShareRequest)
	// prep http request
	httpCreateUserShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateUserShare.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateUserShareResponse, err := clients[tester1].Client.Do(httpCreateUserShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpCreateUserShareResponse.Body)
	// check status of response
	if httpCreateUserShareResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating share: %s", httpCreateUserShareResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject3 protocol.Object
	err = util.FullDecode(httpCreateUserShareResponse.Body, &updatedObject3)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify 1-5, 10 can read, but not others since Everyone removed")
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
		switch clientIdx {
		case 0, 1, 2, 3, 4, 5:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // twl-server-generic and any others that may get added later
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		if clientIdx == len(clients)-1 {
			t.Logf("* Resulting permissions")
			hasEveryone := false
			for _, permission := range updatedObject3.Permissions {
				t.Logf("%s", permission)
				if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
					hasEveryone = true
				}
				if strings.Contains(permission.Grantee, "cn=test tester01,") {
					if !permission.AllowCreate ||
						!permission.AllowUpdate ||
						!permission.AllowDelete ||
						!permission.AllowShare {
						t.Logf("Expected tester1 to have Create, Update, Delete, and Share")
						t.Fail()
					}
				}
				if strings.Contains(permission.Grantee, "tester10") {
					if !permission.AllowRead {
						t.Logf("Expected tester10 to have Read")
						t.Fail()
					}
				}
			}
			if hasEveryone {
				t.Logf("Expected %s to have been removed", models.EveryoneGroup)
				t.Fail()
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

	t.Logf("Update O12 setting an ACM sharing to ODrive G2, which will revoke read from T10")
	acmWithODriveG1 := `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive_G2"]}}}}`
	updatedObject3.RawAcm = acmWithODriveG1
	updatedObject3.Permission = protocol.Permission{}
	uriUpdate := mountPoint + "/objects/" + createdObject.ID + "/properties"
	// jsonify it
	jsonBody, _ = json.Marshal(updatedObject3)
	// prep http request
	httpUpdateObject, _ := http.NewRequest("POST", uriUpdate, bytes.NewBuffer(jsonBody))
	httpUpdateObject.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpUpdateObjectResponse, err := clients[tester1].Client.Do(httpUpdateObject)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpUpdateObjectResponse.Body)
	// check status of response
	if httpUpdateObjectResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when updating object: %s", httpUpdateObjectResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject4 protocol.Object
	err = util.FullDecode(httpUpdateObjectResponse.Body, &updatedObject4)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify 1-5 can read, but nobody else")
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
		switch clientIdx {
		case 1, 2, 3, 4, 5:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // twl-server-generic and any others that may get added later
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		if clientIdx == len(clients)-1 {
			t.Logf("* Resulting permissions")
			hasEveryone := false
			for _, permission := range updatedObject4.Permissions {
				t.Logf("%s", permission)
				if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
					hasEveryone = true
				}
				if strings.Contains(permission.Grantee, "cn=test tester01,") {
					if !permission.AllowCreate ||
						!permission.AllowUpdate ||
						!permission.AllowDelete ||
						!permission.AllowShare {
						t.Logf("Expected tester1 to have Create, Update, Delete, and Share")
						t.Fail()
					}
				}
				if strings.Contains(permission.Grantee, "tester10") {
					t.Logf("Expected tester10 permission to have been removed")
					t.Fail()
				}
			}
			if hasEveryone {
				t.Logf("Expected %s to have been removed", models.EveryoneGroup)
				t.Fail()
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

	t.Logf("Update O12 setting an ACM without a share, which will result in everyone getting access again")
	acmWithNoShare := `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{}}`
	updatedObject4.RawAcm = acmWithNoShare
	updatedObject4.Permissions = []protocol.Permission_1_0{}
	updatedObject4.Permission = protocol.Permission{}
	// jsonify it
	jsonBody, _ = json.Marshal(updatedObject4)
	// prep http request
	httpUpdateObject, _ = http.NewRequest("POST", uriUpdate, bytes.NewBuffer(jsonBody))
	httpUpdateObject.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpUpdateObjectResponse, err = clients[tester1].Client.Do(httpUpdateObject)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpUpdateObjectResponse.Body)
	// check status of response
	if httpUpdateObjectResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when updating object: %s", httpUpdateObjectResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject5 protocol.Object
	err = util.FullDecode(httpUpdateObjectResponse.Body, &updatedObject5)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify everyone can read")
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
		if clientIdx < 10 {
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		} else {
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			}
		}
		if clientIdx == 0 {
			t.Logf("* Resulting permissions")
			hasEveryone := false
			for _, permission := range updatedObject5.Permissions {
				t.Logf("%s", permission)
				if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
					hasEveryone = true
				}
				if strings.Contains(permission.Grantee, "cn=test tester01,") {
					if !permission.AllowCreate ||
						!permission.AllowUpdate ||
						!permission.AllowDelete ||
						!permission.AllowShare {
						t.Logf("Expected tester1 to have Create, Update, Delete, and Share")
						t.Fail()
					}
				}
				if strings.Contains(permission.Grantee, "tester10") {
					t.Logf("Expected tester10 permission to have been removed")
					t.Fail()
				}
				if strings.Contains(permission.Grantee, "odrive_g2") {
					t.Logf("Expected odrive_g2 permission to have been removed")
					t.Fail()
				}
			}
			if !hasEveryone {
				t.Logf("Expected %s to have read", models.EveryoneGroup)
				t.Fail()
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

}

func shouldHaveEveryonePermission(t *testing.T, objID string, clientIdxs ...int) {
	uri := mountPoint + "/objects/" + objID + "/properties"
	getReq, _ := http.NewRequest("GET", uri, nil)
	for _, i := range clientIdxs {
		// reaches for package global clients
		c := clients[i].Client
		resp, err := c.Do(getReq)
		failNowOnErr(t, err, "Unable to do request")
		defer util.FinishBody(resp.Body)
		statusExpected(t, 200, resp, fmt.Sprintf("client id %d should have read for ID %s", i, objID))
		var obj protocol.Object
		err = util.FullDecode(resp.Body, &obj)
		failNowOnErr(t, err, "unable to decode object from json")
		hasEveryone := false
		for _, permission := range obj.Permissions {
			t.Logf("%s", permission)
			if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
				hasEveryone = true
			}
		}

		if !hasEveryone {
			t.Logf("expected -Everyone permission")
			t.FailNow()
		}
	}

}
