package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/server"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
	"decipher.com/object-drive-server/utils"
)

func TestGetObjectStreamForRevision_CurrentVersion(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	clientID := 0

	// ### Create object with stream
	data := "object stream for TestGetObjectStreamForRevision_CurrentVersion"
	tmp1, err := ioutil.TempFile(".", "__tempfile__")
	failNowOnErr(t, err, "could not open temp file for write")
	defer func() {
		name := tmp1.Name()
		tmp1.Close()
		err = os.Remove(name)
	}()
	tmp1.WriteString(data)

	createObjectReq, err := testhelpers.NewCreateObjectPOSTRequest(host, "", tmp1)
	failNowOnErr(t, err, "unable to create HTTP request")
	createObjectRes, err := clients[clientID].Client.Do(createObjectReq)
	failNowOnErr(t, err, "unable to do request")
	defer util.FinishBody(createObjectRes.Body)
	statusMustBe(t, 200, createObjectRes, "error creating object")

	var objResponse protocol.Object
	err = util.FullDecode(createObjectRes.Body, &objResponse)
	failNowOnErr(t, err, "could not decode objResponse")
	defer createObjectRes.Body.Close()

	objID := objResponse.ID

	getObjectStreamReq, err := testhelpers.NewGetObjectStreamRequest(objID, "", host)
	failNowOnErr(t, err, "unable to create HTTP request")

	getObjectStreamRes, err := clients[clientID].Client.Do(getObjectStreamReq)
	failNowOnErr(t, err, "GetObjectStream request failed")
	statusMustBe(t, 200, getObjectStreamRes, "error retriving object stream")
	assertBodyNotNil(t, getObjectStreamRes)
	defer util.FinishBody(getObjectStreamRes.Body)

	tmp2, err := ioutil.TempFile(".", "__tempfile__")
	failNowOnErr(t, err, "could not open temp file for write")
	defer func() {
		name := tmp2.Name()
		tmp2.Close()
		err = os.Remove(name)
	}()

	io.Copy(tmp2, getObjectStreamRes.Body)
	if !testhelpers.AreFilesTheSame(tmp1, tmp2) {
		t.Errorf("Retrieved file contents from getObjectStream don't match original")
		t.FailNow()
	}

	t.Logf("  getting object revision at /history/0")
	getObjectStreamRevisionReq, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, "0", "", host)
	failNowOnErr(t, err, "unable to create HTTP request")
	getObjectStreamRevisionRes, err := clients[clientID].Client.Do(getObjectStreamRevisionReq)
	failNowOnErr(t, err, "getObjectStreamRevision request failed")
	defer util.FinishBody(getObjectStreamRevisionRes.Body)

	statusMustBe(t, 200, getObjectStreamRevisionRes, "error retrieving object stream revision 0")
	assertBodyNotNil(t, getObjectStreamRevisionRes)

	tmp3, err := ioutil.TempFile(".", "__tempfile__")
	failNowOnErr(t, err, "could not open temp file for write")
	defer func() {
		name := tmp3.Name()
		tmp3.Close()
		err = os.Remove(name)
	}()
	io.Copy(tmp3, getObjectStreamRevisionRes.Body)

	if !testhelpers.AreFilesTheSame(tmp1, tmp3) {
		t.Errorf("Retrieved file contents from getObjectStreamRevision don't match original")
		t.FailNow()
	}
}

func TestGetObjectStreamForRevision_OriginalVersion(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	clientID := 0

	// ### Create object with stream
	data1 := "object stream for TestGetObjectStreamForRevision_OriginalVersion"
	tmp1, err := ioutil.TempFile(".", "__tempfile__")
	failNowOnErr(t, err, "could not open temp file for write")
	tmp1.WriteString(data1)
	defer func() {
		name := tmp1.Name()
		tmp1.Close()
		err = os.Remove(name)
	}()

	createObjectReq, err := testhelpers.NewCreateObjectPOSTRequest(host, "", tmp1)
	failNowOnErr(t, err, "unable to create HTTP request")

	createObjectRes, err := clients[clientID].Client.Do(createObjectReq)
	failNowOnErr(t, err, "unable to do request")
	statusMustBe(t, 200, createObjectRes, "error creating object")
	defer util.FinishBody(createObjectRes.Body)

	var objResponse1 protocol.Object
	err = util.FullDecode(createObjectRes.Body, &objResponse1)
	failNowOnErr(t, err, "could not decode createObjectRes")
	defer createObjectRes.Body.Close()

	objID := objResponse1.ID
	changeToken := objResponse1.ChangeToken

	t.Logf("   Update Object Stream with changed stream")
	data2 := data1 + " --- CHANGED"
	tmp2, err := ioutil.TempFile(".", "__tempfile__")
	failNowOnErr(t, err, "could not open temp file for write")
	tmp2.WriteString(data2)
	defer func() {
		name := tmp2.Name()
		tmp2.Close()
		err = os.Remove(name)
	}()

	updateObjectReq, err := testhelpers.UpdateObjectStreamPOSTRequest(objID, changeToken, host, "", tmp2)
	failNowOnErr(t, err, "unable to create HTTP request")

	updateObjectRes, err := clients[clientID].Client.Do(updateObjectReq)
	failNowOnErr(t, err, "unable to do updateObjectReq")
	defer util.FinishBody(updateObjectRes.Body)

	statusMustBe(t, 200, updateObjectRes, "error creating object")

	var objResponse2 protocol.Object
	err = util.FullDecode(updateObjectRes.Body, &objResponse2)
	failNowOnErr(t, err, "could not decode objResponse2")
	defer updateObjectRes.Body.Close()

	// Capture ChangeCount
	changeCount := objResponse2.ChangeCount

	// ### Call GetObjectStream
	getObjectStreamReq, err := testhelpers.NewGetObjectStreamRequest(objID, "", host)
	failNowOnErr(t, err, "unable to create HTTP	request")
	getObjectStreamRes, err := clients[clientID].Client.Do(getObjectStreamReq)
	failNowOnErr(t, err, "getObjectStreamReq failed")
	defer util.FinishBody(getObjectStreamRes.Body)
	statusMustBe(t, 200, getObjectStreamRes, "error retrieving object stream")
	assertBodyNotNil(t, getObjectStreamRes)

	tmp3, err := ioutil.TempFile(".", "__tempfile__")
	failNowOnErr(t, err, "could not open temp file for write")

	defer func() {
		name := tmp3.Name()
		tmp3.Close()
		err = os.Remove(name)
	}()
	io.Copy(tmp3, getObjectStreamRes.Body)
	// ### Compare contents with the update stream that was saved
	if !testhelpers.AreFilesTheSame(tmp2, tmp3) {
		t.Errorf("Retrieved file contents from getObjectStream don't match expected updated stream")
		t.FailNow()
	}

	// ### Call GetObjectStreamRevision /history/0
	getObjectStreamRevisionReq, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, "0", "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRevisionRes, err := clients[clientID].Client.Do(getObjectStreamRevisionReq)
	failNowOnErr(t, err, "getObjectStreamRevisionReq failed")
	statusMustBe(t, 200, getObjectStreamRevisionRes, "error retriving object stream revision")
	assertBodyNotNil(t, getObjectStreamRes)
	defer util.FinishBody(getObjectStreamRevisionRes.Body)

	tmp4, err := ioutil.TempFile(".", "__tempfile__")
	failNowOnErr(t, err, "could not open temp file for write")

	defer func() {
		name := tmp4.Name()
		tmp4.Close()
		err = os.Remove(name)
	}()
	io.Copy(tmp4, getObjectStreamRevisionRes.Body)
	// ### Compare contents with stream that was saved
	if !testhelpers.AreFilesTheSame(tmp1, tmp4) {
		t.Errorf("Retrieved file contents from getObjectStreamRevision don't match original")
		t.FailNow()
	}

	// ### Call GetObjectStreamRevision /history/x
	getObjectStreamRevisionReq2, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, strconv.Itoa(changeCount), "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRevisionRes2, err := clients[clientID].Client.Do(getObjectStreamRevisionReq2)
	if err != nil {
		t.Errorf("GetObjectStreamRevision request 2 failed: %v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(getObjectStreamRevisionRes2.Body)
	if getObjectStreamRevisionRes2.StatusCode != http.StatusOK {
		t.Errorf("Error retrieving object stream revision, got code %d", getObjectStreamRevisionRes2.StatusCode)
		t.FailNow()
	}
	if getObjectStreamRevisionRes2.Body == nil {
		t.Errorf("Response from GetObjectStreamRevision 2 had no body")
		t.FailNow()
	}
	tmp5, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
		t.FailNow()
	}
	defer func() {
		name := tmp5.Name()
		tmp5.Close()
		err = os.Remove(name)
	}()
	io.Copy(tmp5, getObjectStreamRevisionRes2.Body)
	// ### Compare contents with stream that was saved
	if !testhelpers.AreFilesTheSame(tmp2, tmp5) {
		t.Errorf("Retrieved file contents from getObjectStreamRevision 2 don't match update")
		t.FailNow()
	}

}

func TestGetObjectStreamForRevision_DeletedVersion(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	clientID := 0

	// ### Create object with stream
	data1 := "object stream for TestGetObjectStreamForRevision_DeletedVersion"
	tmp1, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
		t.FailNow()
	}
	tmp1.WriteString(data1)
	defer func() {
		name := tmp1.Name()
		tmp1.Close()
		err = os.Remove(name)
	}()
	createObjectReq, err := testhelpers.NewCreateObjectPOSTRequest(host, "", tmp1)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	createObjectRes, err := clients[clientID].Client.Do(createObjectReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(createObjectRes.Body)
	if createObjectRes.StatusCode != http.StatusOK {
		t.Errorf("Error creating object, got code %d", createObjectRes.StatusCode)
		t.FailNow()
	}
	var objResponse1 protocol.Object
	err = util.FullDecode(createObjectRes.Body, &objResponse1)
	if err != nil {
		t.Errorf("Could not decode CreateObject response.")
		t.FailNow()
	}
	createObjectRes.Body.Close()

	// Capture ID and ChangeToken for usage in get calls
	objID := objResponse1.ID
	changeToken := objResponse1.ChangeToken

	// ### Update Object Stream with changed stream (stream 2)
	data2 := data1 + " --- CHANGED"
	tmp2, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
		t.FailNow()
	}
	tmp2.WriteString(data2)
	defer func() {
		name := tmp2.Name()
		tmp2.Close()
		err = os.Remove(name)
	}()
	updateObjectReq, err := testhelpers.UpdateObjectStreamPOSTRequest(objID, changeToken, host, "", tmp2)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	updateObjectRes, err := clients[clientID].Client.Do(updateObjectReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(updateObjectRes.Body)
	if updateObjectRes.StatusCode != http.StatusOK {
		t.Errorf("Error creating object, got code %d", updateObjectRes.StatusCode)
		t.FailNow()
	}
	var objResponse2 protocol.Object
	err = util.FullDecode(updateObjectRes.Body, &objResponse2)
	if err != nil {
		t.Errorf("Could not decode CreateObject response.")
		t.FailNow()
	}
	updateObjectRes.Body.Close()

	// ### Delete Object
	deleteObjectReq, err := testhelpers.NewDeleteObjectRequest(objResponse2, "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	deleteObjectRes, err := clients[clientID].Client.Do(deleteObjectReq)
	if err != nil {
		t.Errorf("DeleteObject request failed: %v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(deleteObjectRes.Body)
	if deleteObjectRes.StatusCode != http.StatusOK {
		t.Errorf("Error calling delete object, got code %d", deleteObjectRes.StatusCode)
		t.FailNow()
	}

	// ### Call GetObjectStream
	getObjectStreamReq, err := testhelpers.NewGetObjectStreamRequest(objID, "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRes, err := clients[clientID].Client.Do(getObjectStreamReq)
	if err != nil {
		t.Errorf("GetObjectStream request failed: %v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(getObjectStreamRes.Body)
	// ### Expect failure because current is deleted
	if getObjectStreamRes.StatusCode == http.StatusOK {
		t.Errorf("Error! We retrieved stream for deleted object %s", getObjectStreamRes.StatusCode)
		t.FailNow()
	}

	// ### Call GetObjectStreamRevision /history/0
	getObjectStreamRevisionReq, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, "0", "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRevisionRes, err := clients[clientID].Client.Do(getObjectStreamRevisionReq)
	if err != nil {
		t.Errorf("GetObjectStreamRevision request failed: %v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(getObjectStreamRevisionRes.Body)
	// ### Expect failure because current is deleted, even though original is not
	if getObjectStreamRevisionRes.StatusCode == http.StatusOK {
		t.Errorf("Error! We retrieved stream of older version when current is deleted %d", getObjectStreamRevisionRes.StatusCode)
		t.FailNow()
	}

}

func TestGetObjectStreamForRevision_WithoutPermission(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	tester10 := 0 // 10
	tester1 := 1  // "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US"

	t.Logf("* Create object with stream")
	data1 := "object stream for TestGetObjectStreamForRevision_WithoutPermission"
	tmp1, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
		t.FailNow()
	}
	tmp1.WriteString(data1)
	defer func() {
		name := tmp1.Name()
		tmp1.Close()
		err = os.Remove(name)
	}()
	createObjectReq, err := testhelpers.NewCreateObjectPOSTRequest(host, "", tmp1)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	createObjectRes, err := clients[tester10].Client.Do(createObjectReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(createObjectRes.Body)
	if createObjectRes.StatusCode != http.StatusOK {
		t.Errorf("Error creating object, got code %d", createObjectRes.StatusCode)
		t.FailNow()
	}
	var objResponse1 protocol.Object
	err = util.FullDecode(createObjectRes.Body, &objResponse1)
	if err != nil {
		t.Errorf("Could not decode CreateObject response.")
		t.FailNow()
	}
	createObjectRes.Body.Close()
	// Capture ID and ChangeToken for usage in get calls
	objID := objResponse1.ID
	changeToken := objResponse1.ChangeToken

	t.Logf("* Add read permission granted to tester1 and tester10")
	shareuri := host + cfg.NginxRootURL + "/shared/" + objID
	shareSetting := protocol.ObjectShare{}
	shareSetting.Share = server.CombineInterface(makeUserShare(fakeDN0), makeUserShare(fakeDN1))
	shareSetting.AllowRead = true
	jsonBody, err := json.Marshal(shareSetting)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	createShareReq, err := http.NewRequest("POST", shareuri, bytes.NewBuffer(jsonBody))
	// grantee := fakeDN1
	// createShareReq, err := testhelpers.NewCreateReadPermissionRequest(objResponse1, grantee, "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	createShareRes, err := clients[tester10].Client.Do(createShareReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(createShareRes.Body)
	if createShareRes.StatusCode != http.StatusOK {
		t.Errorf("Error creating share, got code %d", createShareRes.StatusCode)
		t.FailNow()
	}
	var objShare protocol.Object
	err = util.FullDecode(createShareRes.Body, &objShare)
	if err != nil {
		t.Errorf("Could not decode CreateShare response")
		t.FailNow()
	} else {
		t.Logf("* Resulting permissions")
		hasEveryone := false
		for _, permission := range objShare.Permissions {
			t.Logf("%s", permission)
			if permission.GroupName == models.EveryoneGroup {
				hasEveryone = true
			}
		}
		if hasEveryone {
			t.Logf("Expected %s to have been removed", models.EveryoneGroup)
			t.FailNow()
		}
	}
	createShareRes.Body.Close()

	// Capture new change token from the object being changed from adding a share
	changeToken = objShare.ChangeToken

	t.Logf("* Update Object Stream with changed stream (stream 2)")
	data2 := data1 + " --- CHANGED"
	tmp2, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
		t.FailNow()
	}
	tmp2.WriteString(data2)
	defer func() {
		name := tmp2.Name()
		tmp2.Close()
		err = os.Remove(name)
	}()
	updateObjectReq, err := testhelpers.UpdateObjectStreamPOSTRequest(objID, changeToken, host, "", tmp2)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	updateObjectRes, err := clients[tester10].Client.Do(updateObjectReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(updateObjectRes.Body)

	if updateObjectRes.StatusCode != http.StatusOK {
		t.Errorf("Error updating object stream, got code %d", updateObjectRes.StatusCode)
		t.FailNow()
	}
	var objResponse2 protocol.Object
	err = util.FullDecode(updateObjectRes.Body, &objResponse2)
	failNowOnErr(t, err, "could not decode object response")
	t.Logf("* Resulting permissions")

	hasEveryone := false
	for _, permission := range objResponse2.Permissions {
		t.Logf("%s", permission)
		if permission.GroupName == models.EveryoneGroup {
			hasEveryone = true
		}
	}
	if !hasEveryone {
		t.Logf("Expected %s", models.EveryoneGroup)
		t.FailNow()
	}

	updateObjectRes.Body.Close()

	t.Logf("* Call GetObjectStreamRevision /history/x as second user")
	getObjectStreamRevisionReq1, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, strconv.Itoa(objResponse2.ChangeCount), "", host)
	failNowOnErr(t, err, "unable to create HTTP request")
	getObjectStreamRevisionRes1, err := clients[tester1].Client.Do(getObjectStreamRevisionReq1)
	failNowOnErr(t, err, "GetObjectStreamRevision request 1 failed")
	defer util.FinishBody(getObjectStreamRevisionRes1.Body)

	statusMustBe(t, 200, getObjectStreamRevisionRes1, "error retrieving object stream revision")
	assertBodyNotNil(t, getObjectStreamRevisionRes1)
	tmp3, err := ioutil.TempFile(".", "__tempfile__")
	failNowOnErr(t, err, "could not open temp file for write")
	defer func() {
		name := tmp3.Name()
		tmp3.Close()
		err = os.Remove(name)
	}()
	io.Copy(tmp3, getObjectStreamRevisionRes1.Body)
	// ### Compare contents with stream that was saved
	if !testhelpers.AreFilesTheSame(tmp2, tmp3) {
		t.Errorf("Retrieved file contents from getObjectStreamRevision 1 don't match update")
		t.FailNow()
	}

	t.Logf("* Call GetObjectStreamRevision /history/0 as tester1")
	getObjectStreamRevisionReq2, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, "0", "", host)
	failNowOnErr(t, err, "unable to create http request")
	getObjectStreamRevisionRes2, err := clients[tester1].Client.Do(getObjectStreamRevisionReq2)
	failNowOnErr(t, err, "GetObjectStreamRevision request 2 failed")
	defer util.FinishBody(getObjectStreamRevisionRes2.Body)
	statusMustBe(t, 200, getObjectStreamRevisionRes2,
		"unable to retrieve object stream on request 2 desipte share to everyone")

	assertBodyNotNil(t, getObjectStreamRevisionRes2)
	tmp4, err := ioutil.TempFile(".", "__tempfile__")
	failNowOnErr(t, err, "could not open temp file for write")
	defer func() {
		name := tmp4.Name()
		tmp4.Close()
		err = os.Remove(name)
	}()
	io.Copy(tmp4, getObjectStreamRevisionRes2.Body)
	// ### Compare contents with stream that was saved
	if !testhelpers.AreFilesTheSame(tmp1, tmp4) {
		t.Errorf("Retrieved file contents from getObjectStreamRevision 2 don't match original")
		t.FailNow()
	}

	t.Logf("* Call GetObjectStreamRevision /history/x as tester1")
	getObjectStreamRevisionRes3, err := clients[tester1].Client.Do(getObjectStreamRevisionReq1)
	failNowOnErr(t, err, "GetObjectStreamRevision request 3 failed")
	defer util.FinishBody(getObjectStreamRevisionRes3.Body)

	// ### Expect success since the call to update without share restrictions grants to everyone
	statusMustBe(t, 200, getObjectStreamRevisionRes3,
		"unable to retrieve object stream on request 3 desipte share to everyone")

	t.Logf("* Call GetObjectStreamRevision /history/0 as tester1")
	getObjectStreamRevisionRes4, err := clients[tester1].Client.Do(getObjectStreamRevisionReq2)
	failNowOnErr(t, err, "GetObjectStreamRevision request 4 failed")
	defer util.FinishBody(getObjectStreamRevisionRes4.Body)
	// ### Expect success, same as above
	statusMustBe(t, 200, getObjectStreamRevisionRes4,
		"unable to retrieve object stream on request 4 desipte share to everyone")

}

func TestGetObjectStreamForRevision_WithoutPermissionToCurrent(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	tester10 := 0 // 10
	tester1 := 1  // "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US"

	t.Logf("* Create object with stream")
	data1 := "object stream for TestGetObjectStreamForRevision_WithoutPermissionToCurrent"
	tmp1, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
		t.FailNow()
	}
	tmp1.WriteString(data1)
	defer func() {
		name := tmp1.Name()
		tmp1.Close()
		err = os.Remove(name)
	}()
	createObjectReq, err := testhelpers.NewCreateObjectPOSTRequest(host, "", tmp1)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	createObjectRes, err := clients[tester10].Client.Do(createObjectReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(createObjectRes.Body)
	if createObjectRes.StatusCode != http.StatusOK {
		t.Errorf("Error creating object, got code %d", createObjectRes.StatusCode)
		t.FailNow()
	}
	var objResponse1 protocol.Object
	err = util.FullDecode(createObjectRes.Body, &objResponse1)
	if err != nil {
		t.Errorf("Could not decode CreateObject response.")
		t.FailNow()
	}
	createObjectRes.Body.Close()
	objID := objResponse1.ID

	t.Logf("* Verify tester1 can read since shared to everyone and has clearance")
	getObjectStreamRevisionReq1, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, "0", "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRevisionRes1, err := clients[tester1].Client.Do(getObjectStreamRevisionReq1)
	failNowOnErr(t, err, "getObjectStreamRevisionReq1 failed")
	statusMustBe(t, 200, getObjectStreamRevisionRes1, "error retrieving object stream revision")
	assertBodyNotNil(t, getObjectStreamRevisionRes1)
	defer util.FinishBody(getObjectStreamRevisionRes1.Body)

	t.Logf("* Update object, setting ACM to exceed tester1 clearance")
	updateObj := protocol.UpdateObjectRequest{}
	updateObj.ChangeToken = objResponse1.ChangeToken
	updateObj.ContainsUSPersonsData = objResponse1.ContainsUSPersonsData
	updateObj.Description = objResponse1.Description
	updateObj.ExemptFromFOIA = objResponse1.ExemptFromFOIA
	updateObj.ID = objResponse1.ID
	updateObj.Name = objResponse1.Name + " updated"
	updateObj.RawAcm, _ = utils.UnmarshalStringToInterface(testhelpers.ValidACMTopSecretSITK)
	updateObj.TypeID = objResponse1.TypeID
	updateObj.TypeName = objResponse1.TypeName
	updateUri := host + cfg.NginxRootURL + "/objects/" + objID + "/properties"
	updateReq := makeHTTPRequestFromInterface(t, "POST", updateUri, updateObj)
	updateRes, err := clients[tester10].Client.Do(updateReq)
	failNowOnErr(t, err, "update failed")
	statusMustBe(t, 200, updateRes, "error updating object")
	defer util.FinishBody(updateRes.Body)

	t.Logf("* Verify tester1 cannot read current version")
	getObjectStreamRevisionReq2, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, "1", "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRevisionRes2, err := clients[tester1].Client.Do(getObjectStreamRevisionReq2)
	statusMustBe(t, 403, getObjectStreamRevisionRes2, "expected forbidden when retrieving revision as tester1")
	messageMustContain(t, getObjectStreamRevisionRes2, "User does not have sufficient Clearance")
	defer util.FinishBody(getObjectStreamRevisionRes2.Body)

	t.Logf("* Verify tester1 can no longer read original version")
	getObjectStreamRevisionReq3, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, "0", "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRevisionRes3, err := clients[tester1].Client.Do(getObjectStreamRevisionReq3)
	statusMustBe(t, 403, getObjectStreamRevisionRes3, "expected forbidden when retrieving revision as tester1")
	messageMustContain(t, getObjectStreamRevisionRes3, "User does not have sufficient Clearance")
	defer util.FinishBody(getObjectStreamRevisionRes3.Body)

	t.Logf("* Verify tester10 can read current version")
	getObjectStreamRevisionReq4, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, "1", "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRevisionRes4, err := clients[tester10].Client.Do(getObjectStreamRevisionReq4)
	statusMustBe(t, 200, getObjectStreamRevisionRes4, "error retrieving object stream revision")
	assertBodyNotNil(t, getObjectStreamRevisionRes4)
	defer util.FinishBody(getObjectStreamRevisionRes4.Body)

	t.Logf("* Verify tester10 can read original version")
	getObjectStreamRevisionReq5, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, "0", "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRevisionRes5, err := clients[tester10].Client.Do(getObjectStreamRevisionReq5)
	statusMustBe(t, 200, getObjectStreamRevisionRes5, "error retrieving object stream revision")
	assertBodyNotNil(t, getObjectStreamRevisionRes5)
	defer util.FinishBody(getObjectStreamRevisionRes5.Body)
}

func assertBodyNotNil(t *testing.T, resp *http.Response) {
	if resp.Body == nil {
		t.Errorf("body was nil")
		t.FailNow()
	}
}
