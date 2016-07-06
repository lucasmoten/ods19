package server_test

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"

	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestGetObjectStreamForRevision_CurrentVersion(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	clientID := 0

	// ### Create object with stream
	data := "object stream for TestGetObjectStreamForRevision_CurrentVersion"
	tmp1, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
		t.FailNow()
	}
	tmp1.WriteString(data)
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
	createObjectRes, err := httpclients[clientID].Do(createObjectReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	if createObjectRes.StatusCode != http.StatusOK {
		t.Errorf("Error creating object, got code %d", createObjectRes.StatusCode)
		t.FailNow()
	}
	var objResponse protocol.Object
	err = util.FullDecode(createObjectRes.Body, &objResponse)
	if err != nil {
		t.Errorf("Could not decode CreateObject response.")
		t.FailNow()
	}
	createObjectRes.Body.Close()

	// Capture ID for usage in get calls
	objID := objResponse.ID

	// ### Call GetObjectStream
	getObjectStreamReq, err := testhelpers.NewGetObjectStreamRequest(objID, "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRes, err := httpclients[clientID].Do(getObjectStreamReq)
	if err != nil {
		t.Errorf("GetObjectStream request failed: %v\n", err)
		t.FailNow()
	}
	if getObjectStreamRes.StatusCode != http.StatusOK {
		t.Errorf("Error retrieving object stream, got code %d", getObjectStreamRes.StatusCode)
		t.FailNow()
	}
	if getObjectStreamRes.Body == nil {
		t.Errorf("Response from GetObjectStream had no body")
		t.FailNow()
	}
	tmp2, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
		t.FailNow()
	}
	defer func() {
		name := tmp2.Name()
		tmp2.Close()
		err = os.Remove(name)
	}()
	io.Copy(tmp2, getObjectStreamRes.Body)
	// ### Compare contents with stream that was saved
	if !testhelpers.AreFilesTheSame(tmp1, tmp2) {
		t.Errorf("Retrieved file contents from getObjectStream don't match original")
		t.FailNow()
	}

	// ### Call GetObjectStreamRevision /history/0
	getObjectStreamRevisionReq, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, "0", "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRevisionRes, err := httpclients[clientID].Do(getObjectStreamRevisionReq)
	if err != nil {
		t.Errorf("GetObjectStreamRevision request failed: %v\n", err)
		t.FailNow()
	}
	if getObjectStreamRevisionRes.StatusCode != http.StatusOK {
		t.Errorf("Error retrieving object stream revision, got code %d", getObjectStreamRes.StatusCode)
		t.FailNow()
	}
	if getObjectStreamRevisionRes.Body == nil {
		t.Errorf("Response from GetObjectStreamRevision had no body")
		t.FailNow()
	}
	tmp3, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
		t.FailNow()
	}
	defer func() {
		name := tmp3.Name()
		tmp3.Close()
		err = os.Remove(name)
	}()
	io.Copy(tmp3, getObjectStreamRevisionRes.Body)
	// ### Compare contents with stream that was saved
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
	createObjectRes, err := httpclients[clientID].Do(createObjectReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
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
	updateObjectRes, err := httpclients[clientID].Do(updateObjectReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
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

	// Capture ChangeCount
	changeCount := objResponse2.ChangeCount

	// ### Call GetObjectStream
	getObjectStreamReq, err := testhelpers.NewGetObjectStreamRequest(objID, "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRes, err := httpclients[clientID].Do(getObjectStreamReq)
	if err != nil {
		t.Errorf("GetObjectStream request failed: %v\n", err)
		t.FailNow()
	}
	if getObjectStreamRes.StatusCode != http.StatusOK {
		t.Errorf("Error retrieving object stream, got code %d", getObjectStreamRes.StatusCode)
		t.FailNow()
	}
	if getObjectStreamRes.Body == nil {
		t.Errorf("Response from GetObjectStream had no body")
		t.FailNow()
	}
	tmp3, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
		t.FailNow()
	}
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
	getObjectStreamRevisionRes, err := httpclients[clientID].Do(getObjectStreamRevisionReq)
	if err != nil {
		t.Errorf("GetObjectStreamRevision request failed: %v\n", err)
		t.FailNow()
	}
	if getObjectStreamRevisionRes.StatusCode != http.StatusOK {
		t.Errorf("Error retrieving object stream revision, got code %d", getObjectStreamRevisionRes.StatusCode)
		t.FailNow()
	}
	if getObjectStreamRevisionRes.Body == nil {
		t.Errorf("Response from GetObjectStreamRevision had no body")
		t.FailNow()
	}
	tmp4, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
		t.FailNow()
	}
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
	getObjectStreamRevisionRes2, err := httpclients[clientID].Do(getObjectStreamRevisionReq2)
	if err != nil {
		t.Errorf("GetObjectStreamRevision request 2 failed: %v\n", err)
		t.FailNow()
	}
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
	createObjectRes, err := httpclients[clientID].Do(createObjectReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
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
	updateObjectRes, err := httpclients[clientID].Do(updateObjectReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
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
	deleteObjectRes, err := httpclients[clientID].Do(deleteObjectReq)
	if err != nil {
		t.Errorf("DeleteObject request failed: %v\n", err)
		t.FailNow()
	}
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
	getObjectStreamRes, err := httpclients[clientID].Do(getObjectStreamReq)
	if err != nil {
		t.Errorf("GetObjectStream request failed: %v\n", err)
		t.FailNow()
	}
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
	getObjectStreamRevisionRes, err := httpclients[clientID].Do(getObjectStreamRevisionReq)
	if err != nil {
		t.Errorf("GetObjectStreamRevision request failed: %v\n", err)
		t.FailNow()
	}
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

	clientID := 0  // 10
	clientID1 := 1 // "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US"

	// ### Create object with stream
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
	createObjectRes, err := httpclients[clientID].Do(createObjectReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
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

	// ### Add read permission granted to second user
	grantee := fakeDN1
	createShareReq, err := testhelpers.NewCreateReadPermissionRequest(objResponse1, grantee, "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	createShareRes, err := httpclients[clientID].Do(createShareReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	if createShareRes.StatusCode != http.StatusOK {
		t.Errorf("Error creating share, got code %d", createShareRes.StatusCode)
		t.FailNow()
	}
	var objShare protocol.Object
	err = util.FullDecode(createShareRes.Body, &objShare)
	if err != nil {
		t.Errorf("Could not decode CreateShare response")
		t.FailNow()
	}
	createShareRes.Body.Close()

	// Capture new change token from the object being changed from adding a share
	changeToken = objShare.ChangeToken

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
	updateObjectRes, err := httpclients[clientID].Do(updateObjectReq)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
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

	// ### Call GetObjectStreamRevision /history/x as second user
	getObjectStreamRevisionReq1, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, strconv.Itoa(objResponse2.ChangeCount), "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRevisionRes1, err := httpclients[clientID1].Do(getObjectStreamRevisionReq1)
	if err != nil {
		t.Errorf("GetObjectStreamRevision request 1 failed: %v\n", err)
		t.FailNow()
	}
	if getObjectStreamRevisionRes1.StatusCode != http.StatusOK {
		t.Errorf("Error retrieving object stream revision, got code %d", getObjectStreamRevisionRes1.StatusCode)
		t.FailNow()
	}
	if getObjectStreamRevisionRes1.Body == nil {
		t.Errorf("Response from GetObjectStreamRevision 1 had no body")
		t.FailNow()
	}
	tmp3, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
		t.FailNow()
	}
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

	// ### Call GetObjectStreamRevision /history/0 as second user
	getObjectStreamRevisionReq2, err := testhelpers.NewGetObjectStreamRevisionRequest(objID, "0", "", host)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
		t.FailNow()
	}
	getObjectStreamRevisionRes2, err := httpclients[clientID1].Do(getObjectStreamRevisionReq2)
	if err != nil {
		t.Errorf("GetObjectStreamRevision request 2 failed: %v\n", err)
		t.FailNow()
	}
	if getObjectStreamRevisionRes2.StatusCode != http.StatusOK {
		t.Errorf("Error retrieving object stream revision, got code %d", getObjectStreamRevisionRes2.StatusCode)
		t.FailNow()
	}
	if getObjectStreamRevisionRes2.Body == nil {
		t.Errorf("Response from GetObjectStreamRevision 2 had no body")
		t.FailNow()
	}
	tmp4, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
		t.FailNow()
	}
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

	// --------------------------------------
	// Cant remove permissions yet as this is not reimplemented yet

	// // ### Remove permissions for user 2
	// deleteShareReq, err := testhelpers.NewDeletePermissionRequest(objResponse1, objShare, "", host)
	// if err != nil {
	// 	t.Errorf("Unable to create HTTP request: %v\n", err)
	// 	t.FailNow()
	// }
	// deleteShareRes, err := httpclients[clientID].Do(deleteShareReq)
	// if err != nil {
	// 	t.Errorf("Unable to do request:%v\n", err)
	// 	t.FailNow()
	// }
	// if deleteShareRes.StatusCode != http.StatusGone {
	// 	t.Errorf("Error removing share, got code %d but expected %d (StatusGone) since it would have been deleted internally when updating the object due to ACM override", deleteShareRes.StatusCode, http.StatusGone)
	// 	t.FailNow()
	// }

	// ### Call GetObjectStreamRevision /history/x as second user
	getObjectStreamRevisionRes3, err := httpclients[clientID1].Do(getObjectStreamRevisionReq1)
	if err != nil {
		t.Errorf("GetObjectStreamRevision request 3 failed: %v\n", err)
		t.FailNow()
	}
	// ### Expect success since the call to update without share restrictions grants to everyone
	if getObjectStreamRevisionRes3.StatusCode != http.StatusOK {
		t.Errorf("Error! Was not able to retrieve object stream revision on request 3 despite share to everyone, status %d", getObjectStreamRevisionRes3.StatusCode)
		t.FailNow()
	}

	// ### Call GetObjectStreamRevision /history/0 as second user
	getObjectStreamRevisionRes4, err := httpclients[clientID1].Do(getObjectStreamRevisionReq2)
	if err != nil {
		t.Errorf("GetObjectStreamRevision request 4 failed: %v\n", err)
		t.FailNow()
	}
	// ### Expect success, same as above
	if getObjectStreamRevisionRes4.StatusCode != http.StatusOK {
		t.Errorf("Error! Was not able to retrieve object stream revision on request 4 despite share to everyon, status %d", getObjectStreamRevisionRes4.StatusCode)
		t.FailNow()
	}

}
