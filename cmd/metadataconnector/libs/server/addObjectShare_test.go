package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/util/testhelpers"

	"decipher.com/object-drive-server/protocol"
)

func TestAddObjectShare(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid1 := 0
	clientid2 := 1

	if verboseOutput {
		t.Logf("(Verbose Mode) Using client id %d", clientid1)
		fmt.Println()
	}

	// Create 2 folders under root
	folder1, err := makeFolderViaJSON("Test Folder 1 "+strconv.FormatInt(time.Now().Unix(), 10), clientid1)
	if err != nil {
		t.Logf("Error making folder 1: %v", err)
		t.FailNow()
	}
	folder2, err := makeFolderViaJSON("Test Folder 2 "+strconv.FormatInt(time.Now().Unix(), 10), clientid1)
	if err != nil {
		t.Logf("Error making folder 2: %v", err)
		t.FailNow()
	}

	// Attempt to move folder 2 under folder 1
	moveuri := host + cfg.RootURL + "/objects/" + folder2.ID + "/move/" + folder1.ID
	objChangeToken := protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = folder2.ChangeToken
	jsonBody, err := json.Marshal(objChangeToken)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("POST", moveuri, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	res, err := httpclients[clientid1].Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// process Response
	if res.StatusCode != http.StatusOK {
		t.Logf("bad status: %s", res.Status)
		t.FailNow()
	}

	// Attempt to retrieve folder1 as clientid2
	geturi := host + cfg.RootURL + "/objects/" + folder1.ID + "/properties"
	getReq1, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes1, err := httpclients[clientid2].Do(getReq1)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if getRes1.StatusCode == http.StatusOK {
		t.Logf("clientid2 was able to get object when not shared")
		t.FailNow()
	}

	// Attempt to retrieve folder2 as clientid2
	geturi = host + cfg.RootURL + "/objects/" + folder2.ID + "/properties"
	getReq2, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes2, err := httpclients[clientid2].Do(getReq2)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if getRes2.StatusCode == http.StatusOK {
		t.Logf("clientid2 was able to get object when not shared")
		t.FailNow()
	}

	// Add share as clientid1 for clientid2 to folder1 without propagation
	shareuri := host + cfg.RootURL + "/shared/" + folder1.ID
	shareSetting := protocol.ObjectGrant{}
	shareSetting.Grantee = "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US"
	shareSetting.Read = true
	jsonBody, err = json.Marshal(shareSetting)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	getReq3, err := http.NewRequest("POST", shareuri, bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes3, err := httpclients[clientid1].Do(getReq3)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if getRes3.StatusCode != http.StatusOK {
		t.Logf("share creation failed")
		t.FailNow()
	}

	// Attempt to retrieve folder1 as clientid2
	geturi = host + cfg.RootURL + "/objects/" + folder1.ID + "/properties"
	getReq4, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes4, err := httpclients[clientid2].Do(getReq4)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if getRes4.StatusCode != http.StatusOK {
		t.Logf("clientid2 was not able to get shared object, got status %d", getRes4.StatusCode)
		t.FailNow()
	}

	// Attempt to retrieve folder2 as clientid2
	geturi = host + cfg.RootURL + "/objects/" + folder2.ID + "/properties"
	getReq5, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes5, err := httpclients[clientid2].Do(getReq5)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if getRes5.StatusCode == http.StatusOK {
		t.Logf("clientid2 was able to get object when not shared")
		t.FailNow()
	}

	// Add share as clientid1 for clientid2 to folder1 with propagation
	shareuri = host + cfg.RootURL + "/shared/" + folder1.ID
	shareSetting = protocol.ObjectGrant{}
	shareSetting.Grantee = "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US"
	shareSetting.Read = true
	shareSetting.PropagateToChildren = true
	jsonBody, err = json.Marshal(shareSetting)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	getReq6, err := http.NewRequest("POST", shareuri, bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes6, err := httpclients[clientid1].Do(getReq6)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if getRes6.StatusCode != http.StatusOK {
		t.Logf("share creation failed")
		t.FailNow()
	}

	// Attempt to retrieve folder2 as clientid2
	geturi = host + cfg.RootURL + "/objects/" + folder2.ID + "/properties"
	getReq7, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes7, err := httpclients[clientid2].Do(getReq7)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if getRes7.StatusCode != http.StatusOK {
		t.Logf("clientid2 was not able to get object when shared")
		t.FailNow()
	}
}

func TestAddObjectShareAndVerifyACM(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid1 := 0
	clientid2 := 1
	clientTestperson10 := httpclients[clientid1]
	clientTestperson01 := httpclients[clientid2]

	if verboseOutput {
		t.Logf("(Verbose Mode) Using client id %d", clientid1)
		fmt.Println()
	}

	// Create object as testperson10 with ACM that is TS
	createdFolder, err := makeFolderWithACMViaJSON("TestAddFolderWithTSSITK "+strconv.FormatInt(time.Now().Unix(), 10), testhelpers.ValidACMTopSecretSITK, clientid1)
	if err != nil {
		t.Logf("Error making folder 1: %v", err)
		t.FailNow()
	}

	// Verify testperson10 can get object
	getReq1, err := testhelpers.NewGetObjectRequest(createdFolder.ID, "", host)
	if err != nil {
		t.Logf("Unable to generate get re-request:%v", err)
	}
	getRes1, err := clientTestperson10.Do(getReq1)
	if err != nil {
		t.Logf("Unable to do get request as tp10:%v\n", err)
		t.FailNow()
	}
	if getRes1.StatusCode != http.StatusOK {
		t.Logf("Unexpected status getting object created by testperson10: %d", getRes1.StatusCode)
		t.FailNow()
	}

	// Verify testperson01 can not get object
	getRes2, err := clientTestperson01.Do(getReq1)
	if err != nil {
		t.Logf("Unable to do get request as tp01:%v\n", err)
		t.FailNow()
	}
	if getRes2.StatusCode == http.StatusOK {
		t.Logf("Unexpected status requesting object created by testperson10 as testperson01: %d", getRes2.StatusCode)
		t.FailNow()
	}

	// Create share granting read access to testperson01
	shareuri := host + cfg.RootURL + "/shared/" + createdFolder.ID
	shareSetting := protocol.ObjectGrant{}
	shareSetting.Grantee = "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US"
	shareSetting.Read = true
	jsonBody, err := json.Marshal(shareSetting)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	shareRequest, err := http.NewRequest("POST", shareuri, bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	shareResponse, err := clientTestperson10.Do(shareRequest)
	if err != nil {
		t.Logf("Unable to create share:%v", err)
		t.FailNow()
	}
	if shareResponse.StatusCode != http.StatusOK {
		t.Logf("share creation failed")
		t.FailNow()
	}

	// Verify that the object is listed as shared from testperson10 /shared
	shareListURI := host + cfg.RootURL + "/shared?filterField=name&condition=equals&expression=" + url.QueryEscape(createdFolder.Name)
	shareListRequest, err := http.NewRequest("GET", shareListURI, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	shareListResponse, err := clientTestperson10.Do(shareListRequest)
	if err != nil {
		t.Logf("Unable to retrieve share list: %v", err)
		t.FailNow()
	}
	if shareListResponse.StatusCode != http.StatusOK {
		t.Logf("List retrieval failed. Unexpected status code: %d", shareListResponse.StatusCode)
		t.FailNow()
	}
	decoder := json.NewDecoder(shareListResponse.Body)
	var listOfObjects protocol.ObjectResultset
	err = decoder.Decode(&listOfObjects)
	if err != nil {
		t.Logf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}
	if listOfObjects.TotalRows == 0 {
		t.Logf("No matching shares")
		t.FailNow()
	}

	// Verify testperson01 can not get object (they lack ACM)
	getRes3, err := clientTestperson01.Do(getReq1)
	if err != nil {
		t.Logf("Unable to do get request as tp01:%v\n", err)
		t.FailNow()
	}
	if getRes3.StatusCode == http.StatusOK {
		t.Logf("Unexpected status requesting object created by testperson10 as testperson01 after it was shared: %d", getRes3.StatusCode)
		t.FailNow()
	}

	// Update object as testperson10 to change ACM to be U
	folderUpdate := createdFolder
	folderUpdate.RawAcm = testhelpers.ValidACMUnclassified
	updateuri := host + cfg.RootURL + "/objects/" + folderUpdate.ID + "/properties"
	jsonBody, err = json.Marshal(folderUpdate)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	updateRequest, err := http.NewRequest("POST", updateuri, bytes.NewBuffer(jsonBody))
	updateRequest.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	updateResponse, err := clientTestperson10.Do(updateRequest)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// process Response
	if updateResponse.StatusCode != http.StatusOK {
		t.Logf("bad status: %s", updateResponse.Status)
		t.FailNow()
	}

	// Verify testperson01 can list objects in the shared to me /shared
	sharedToMeListURI := host + cfg.RootURL + "/shares?filterField=name&condition=equals&expression=" + url.QueryEscape(createdFolder.Name)
	sharedToMeListRequest, err := http.NewRequest("GET", sharedToMeListURI, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	sharedToMeResponse, err := clientTestperson01.Do(sharedToMeListRequest)
	if err != nil {
		t.Logf("Unable to retrieve share list: %v", err)
		t.FailNow()
	}
	if sharedToMeResponse.StatusCode != http.StatusOK {
		t.Logf("List retrieval failed. Unexpected status code: %d", sharedToMeResponse.StatusCode)
		t.FailNow()
	}
	decoder = json.NewDecoder(sharedToMeResponse.Body)
	var listOfObjects2 protocol.ObjectResultset
	err = decoder.Decode(&listOfObjects2)
	if err != nil {
		t.Logf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}
	if listOfObjects2.TotalRows == 0 {
		t.Logf("No matching shares")
		t.FailNow()
	}

	// Verify testperson01 can get the shared object
	getRes4, err := clientTestperson01.Do(getReq1)
	if err != nil {
		t.Logf("Unable to do get request as tp01:%v\n", err)
		t.FailNow()
	}
	if getRes4.StatusCode != http.StatusOK {
		t.Logf("Unexpected status requesting object created by testperson10 as testperson01 after it was shared and acm changed: %d", getRes4.StatusCode)
		t.FailNow()
	}
}
