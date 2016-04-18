package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	cfg "decipher.com/object-drive-server/config"

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
