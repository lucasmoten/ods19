package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/util"

	"decipher.com/object-drive-server/protocol"
)

func TestRemoveObjectShare(t *testing.T) {
	// Skipping this test for now as it needs rewritten after remove object share is re-implemented.
	// It may be that we dont actually do a remove object share, but rather set object share to flush and recreate
	// related files that may go are protocol objects for RemoveObjectShareRequest and RemovedObjectShareResponse

	t.Skip()

	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid1, clientid2 := 0, 1

	if verboseOutput {
		t.Logf("(Verbose Mode) Using client id %d\n", clientid1)
	}

	// Create 2 folders under root
	folder1 := makeFolderViaJSON("Test Folder 1 ", clientid1, t)
	folder2 := makeFolderViaJSON("Test Folder 2 ", clientid1, t)

	// Attempt to move folder 2 under folder 1
	moveuri := host + cfg.NginxRootURL + "/objects/" + folder2.ID + "/move/" + folder1.ID
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
	res, err := clients[clientid1].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// process Response
	if res.StatusCode != http.StatusOK {
		t.Logf("bad status: %s", res.Status)
		t.FailNow()
	}
	res.Body.Close()

	// Add share as clientid1 for clientid2 to folder1 with propagation
	shareuri := host + cfg.NginxRootURL + "/shared/" + folder1.ID
	shareSetting := protocol.ObjectShare{}
	shareSetting.Share = makeUserShare(fakeDN1)
	//shareSetting.Grantee = fakeDN1
	shareSetting.AllowRead = true
	shareSetting.PropagateToChildren = true
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
	getRes3, err := clients[clientid1].Client.Do(getReq3)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if getRes3.StatusCode != http.StatusOK {
		t.Logf("share creation failed")
		t.FailNow()
	}
	var createdShare protocol.Permission
	err = util.FullDecode(getRes3.Body, &createdShare)
	if err != nil {
		t.Logf("Error decoding json to Permission: %v", err)
		t.FailNow()
	}
	getRes3.Body.Close()

	// Attempt to retrieve folder1 as clientid2
	geturi := host + cfg.NginxRootURL + "/objects/" + folder1.ID + "/properties"
	getReq4, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes4, err := clients[clientid2].Client.Do(getReq4)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if getRes4.StatusCode != http.StatusOK {
		t.Logf("clientid2 was not able to get shared object")
		t.FailNow()
	}
	getRes4.Body.Close()

	// Attempt to retrieve folder2 as clientid2
	geturi = host + cfg.NginxRootURL + "/objects/" + folder2.ID + "/properties"
	getReq5, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes5, err := clients[clientid2].Client.Do(getReq5)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if getRes5.StatusCode != http.StatusOK {
		t.Logf("clientid2 was able to get object when not shared")
		t.FailNow()
	}
	getRes5.Body.Close()

	// Remove share as clientid1 with propagation
	removeshareuri := host + cfg.NginxRootURL + "/shared/" + folder1.ID + "/" + createdShare.ID
	removesharebody := protocol.RemoveObjectShareRequest{}
	removesharebody.ObjectID = folder1.ID
	removesharebody.ShareID = createdShare.ID
	removesharebody.ChangeToken = createdShare.ChangeToken
	t.Logf("Share id: %s, change token: %s", createdShare.ID, createdShare.ChangeToken)
	removesharebody.PropagateToChildren = true
	jsonBody, err = json.Marshal(removesharebody)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	getReq6, err := http.NewRequest("DELETE", removeshareuri, bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getReq6.Header.Set("Content-Type", "application/json")
	getRes6, err := clients[clientid1].Client.Do(getReq6)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if getRes6.StatusCode != http.StatusOK {
		t.Logf("share removal failed")
		t.FailNow()
	}
	var removedShare protocol.RemovedObjectShareResponse
	err = util.FullDecode(getRes6.Body, &removedShare)
	if err != nil {
		t.Logf("Error decoding json to Removed share response: %v", err)
		t.FailNow()
	}
	t.Logf("share deleted %s", removedShare.DeletedDate)
	getRes6.Body.Close()

	// Attempt to retrieve folder1 as clientid2
	geturi = host + cfg.NginxRootURL + "/objects/" + folder1.ID + "/properties"
	getReq7, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes7, err := clients[clientid2].Client.Do(getReq7)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if getRes7.StatusCode != http.StatusOK {
		t.Logf("clientid2 was not able to get object shared to 'everyone' after personal share was removed")
		t.FailNow()
	}
	getRes7.Body.Close()

	// Attempt to retrieve folder2 as clientid2
	geturi = host + cfg.NginxRootURL + "/objects/" + folder2.ID + "/properties"
	getReq8, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes8, err := clients[clientid2].Client.Do(getReq8)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if getRes8.StatusCode != http.StatusOK {
		t.Logf("clientid2 was not able to get object shared to 'everyone' for object 2 after personal share removed")
		t.FailNow()
	}
	getRes8.Body.Close()
}
