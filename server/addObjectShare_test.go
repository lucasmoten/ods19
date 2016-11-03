package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/server"
	"decipher.com/object-drive-server/util/testhelpers"

	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

func TestAddObjectShare(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	tester10 := 0 // was clientid1
	tester1 := 1  // was clientid0

	if verboseOutput {
		t.Logf("(Verbose Mode) Using tester10")
		fmt.Println()
	}

	t.Logf("* Creating 2 folders under root")
	folder1 := makeFolderViaJSON("Test Folder 1 ", tester10, t)
	folder2 := makeFolderViaJSON("Test Folder 2 ", tester10, t)

	t.Logf("* Moving folder 2 under folder 1")
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
	res, err := clients[tester10].Client.Do(req)
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

	t.Logf("* Retrieve folder 1 as tester1")
	geturi := host + cfg.NginxRootURL + "/objects/" + folder1.ID + "/properties"
	getReq1, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes1, err := clients[tester1].Client.Do(getReq1)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(getRes1.Body)
	if getRes1.StatusCode != http.StatusOK {
		t.Logf("clientid2 was not able to get folder1 object despite being shared to 'Everyone'")
		t.FailNow()
	}
	var retrievedObject protocol.Object
	err = util.FullDecode(getRes1.Body, &retrievedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	for _, permission := range retrievedObject.Permissions {
		t.Logf("%s", permission)
	}

	t.Logf("* Retrieve folder 2 as tester1")
	geturi = host + cfg.NginxRootURL + "/objects/" + folder2.ID + "/properties"
	getReq2, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes2, err := clients[tester1].Client.Do(getReq2)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(getRes2.Body)
	if getRes2.StatusCode != http.StatusOK {
		t.Logf("tester1 was not able to get folder2 object despite being shared to 'Everyone'")
		t.FailNow()
	}
	err = util.FullDecode(getRes2.Body, &retrievedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	for _, permission := range retrievedObject.Permissions {
		t.Logf("%s", permission)
	}

	t.Logf("* Add read share as tester10 for tester1,tester10 to folder1")
	shareuri := host + cfg.NginxRootURL + "/shared/" + folder1.ID
	shareSetting := protocol.ObjectShare{}
	shareSetting.Share = server.CombineInterface(nil, makeUserShare(fakeDN0), makeUserShare(fakeDN1))
	shareSetting.AllowRead = true
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
	getRes3, err := clients[tester10].Client.Do(getReq3)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(getRes3.Body)
	if getRes3.StatusCode != http.StatusOK {
		t.Logf("share creation failed")
		t.FailNow()
	}
	err = util.FullDecode(getRes3.Body, &retrievedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	for _, permission := range retrievedObject.Permissions {
		t.Logf("%s", permission)
	}

	t.Logf("* Attempt to retrieve folder1 as tester1")
	geturi = host + cfg.NginxRootURL + "/objects/" + folder1.ID + "/properties"
	getReq4, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes4, err := clients[tester1].Client.Do(getReq4)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(getRes4.Body)
	if getRes4.StatusCode != http.StatusOK {
		t.Logf("clientid2 was not able to get shared object, got status %d", getRes4.StatusCode)
		t.FailNow()
	}
	err = util.FullDecode(getRes4.Body, &retrievedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	for _, permission := range retrievedObject.Permissions {
		t.Logf("%s", permission)
	}

	t.Logf("* Attempt to retrieve folder2 as tester1")
	geturi = host + cfg.NginxRootURL + "/objects/" + folder2.ID + "/properties"
	getReq5, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes5, err := clients[tester1].Client.Do(getReq5)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(getRes5.Body)
	if getRes5.StatusCode != http.StatusOK {
		t.Logf("clientid2 was not able to get shared object that should still be shared to everyone. Got status %d", getRes5.StatusCode)
		t.FailNow()
	}
	err = util.FullDecode(getRes5.Body, &retrievedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	hasEveryone := false
	for _, permission := range retrievedObject.Permissions {
		t.Logf("%s", permission)
		if permission.GroupName == models.EveryoneGroup {
			hasEveryone = true
		}
	}
	if !hasEveryone {
		t.Logf("Missing %s", models.EveryoneGroup)
		t.FailNow()
	}

	t.Logf("* Add share as tester10 for tester1 to folder1 (NOOP since read exists)")
	shareuri = host + cfg.NginxRootURL + "/shared/" + folder1.ID
	shareSetting = protocol.ObjectShare{}
	shareSetting.Share = server.CombineInterface(nil, makeUserShare(fakeDN0), makeUserShare(fakeDN1))
	shareSetting.AllowRead = true
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
	getRes6, err := clients[tester10].Client.Do(getReq6)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(getRes6.Body)
	if getRes6.StatusCode != http.StatusOK {
		t.Logf("share creation failed")
		t.FailNow()
	}
	err = util.FullDecode(getRes6.Body, &retrievedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	for _, permission := range retrievedObject.Permissions {
		t.Logf("%s", permission)
	}

	t.Logf("* Attempt to retrieve folder2 as tester1")
	geturi = host + cfg.NginxRootURL + "/objects/" + folder2.ID + "/properties"
	getReq7, err := http.NewRequest("GET", geturi, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	getRes7, err := clients[tester1].Client.Do(getReq7)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(getRes7.Body)
	if getRes7.StatusCode != http.StatusOK {
		t.Logf("clientid2 was not able to get object when shared")
		t.FailNow()
	}
	err = util.FullDecode(getRes7.Body, &retrievedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	hasEveryone = false
	for _, permission := range retrievedObject.Permissions {
		t.Logf("%s", permission)
		if permission.GroupName == models.EveryoneGroup {
			hasEveryone = true
		}
	}
	if !hasEveryone {
		t.Logf("Missing %s", models.EveryoneGroup)
		t.FailNow()
	}

}

func TestAddObjectShareAndVerifyACM(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientTestperson10 := clients[0].Client
	clientTestperson01 := clients[1].Client

	if verboseOutput {
		t.Logf("(Verbose Mode) Using testperson10")
		fmt.Println()
	}

	t.Logf("* Create object as testperson10 with ACM that is TS")
	createdFolder, err := makeFolderWithACMViaJSON("TestAddFolderWithTSSITK "+strconv.FormatInt(time.Now().Unix(), 10), testhelpers.ValidACMTopSecretSITK, 0)
	if err != nil {
		t.Logf("Error making folder 1: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify testperson10 can get object")
	getReq1, err := testhelpers.NewGetObjectRequest(createdFolder.ID, "", host)
	if err != nil {
		t.Logf("Unable to generate get re-request:%v", err)
	}
	getRes1, err := clientTestperson10.Do(getReq1)
	if err != nil {
		t.Logf("Unable to do get request as tp10:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(getRes1.Body)
	if getRes1.StatusCode != http.StatusOK {
		t.Logf("Unexpected status getting object created by testperson10: %d", getRes1.StatusCode)
		t.FailNow()
	}
	var retrievedObject protocol.Object
	err = util.FullDecode(getRes1.Body, &retrievedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	hasEveryone := false
	for _, permission := range retrievedObject.Permissions {
		t.Logf("%s", permission)
		if permission.GroupName == models.EveryoneGroup {
			hasEveryone = true
		}
	}
	if !hasEveryone {
		t.Logf("Expected %s", models.EveryoneGroup)
		t.FailNow()
	}

	t.Logf("* Verify testperson01 can not get object")
	getRes2, err := clientTestperson01.Do(getReq1)
	if err != nil {
		t.Logf("Unable to do get request as tp01:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(getRes2.Body)
	if getRes2.StatusCode != http.StatusForbidden { // == http.StatusOK {
		t.Logf("Unexpected status requesting object created by testperson10 as testperson01: %d", getRes2.StatusCode)
		t.FailNow()
	}

	t.Logf("* Create share granting read access to odrive") // will replace models.EveryoneGroup
	shareuri := host + cfg.NginxRootURL + "/shared/" + createdFolder.ID
	shareSetting := protocol.ObjectShare{}
	shareSetting.Share = makeGroupShare("DCTC", "DCTC", "ODrive")
	//shareSetting.Grantee = fakeDN1
	shareSetting.AllowRead = true
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
	defer util.FinishBody(shareResponse.Body)
	if shareResponse.StatusCode != http.StatusOK {
		t.Logf("share creation failed")
		t.FailNow()
	}
	// Since the share updates the object and returns updated state in response, we need to capture
	var updatedObject protocol.Object
	err = util.FullDecode(shareResponse.Body, &updatedObject)
	if err != nil {
		t.Logf("Error decoding json to Object:%v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	hasEveryone = false
	for _, permission := range updatedObject.Permissions {
		t.Logf("%s", permission)
		if permission.GroupName == models.EveryoneGroup {
			hasEveryone = true
		}
	}
	if hasEveryone {
		t.Logf("Expected %s to have been removed", models.EveryoneGroup)
		t.FailNow()
	}

	t.Logf("* Verify that the object is listed as shared from testperson10 /shared")
	shareListURI := host + cfg.NginxRootURL + "/shared?filterField=name&condition=equals&expression=" + url.QueryEscape(createdFolder.Name)
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
	defer util.FinishBody(shareListResponse.Body)
	if shareListResponse.StatusCode != http.StatusOK {
		t.Logf("List retrieval failed. Unexpected status code: %d", shareListResponse.StatusCode)
		t.FailNow()
	}
	var listOfObjects protocol.ObjectResultset
	err = util.FullDecode(shareListResponse.Body, &listOfObjects)
	if err != nil {
		t.Logf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}
	if listOfObjects.TotalRows == 0 {
		t.Logf("No matching shares")
		t.FailNow()
	}

	t.Logf("* Verify testperson01 can not get object (they lack ACM)")
	getRes3, err := clientTestperson01.Do(getReq1)
	if err != nil {
		t.Logf("Unable to do get request as tp01:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(getRes3.Body)
	if getRes3.StatusCode == http.StatusOK {
		t.Logf("Unexpected status requesting object created by testperson10 as testperson01 after it was shared: %d", getRes3.StatusCode)
		t.FailNow()
	}

	t.Logf("* Update object as testperson10 to change ACM to be U")
	//folderUpdate := createdFolder -- object changed from adding permission(s) so created object does not have current change token
	folderUpdate := updatedObject
	folderUpdate.RawAcm = testhelpers.ValidACMUnclassified
	updateuri := host + cfg.NginxRootURL + "/objects/" + folderUpdate.ID + "/properties"
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
	defer util.FinishBody(updateResponse.Body)
	// process Response
	if updateResponse.StatusCode != http.StatusOK {
		t.Logf("bad status: %s", updateResponse.Status)
		t.FailNow()
	}
	err = util.FullDecode(updateResponse.Body, &retrievedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	hasEveryone = false
	for _, permission := range retrievedObject.Permissions {
		t.Logf("%s", permission)
		if permission.GroupName == models.EveryoneGroup {
			hasEveryone = true
		}
	}
	if !hasEveryone {
		t.Logf("Expected %s", models.EveryoneGroup)
		t.FailNow()
	}

	t.Logf("* Verify testperson01 can list objects in the shared to me /shares")
	sharedToMeListURI := host + cfg.NginxRootURL + "/shares?filterField=name&condition=equals&expression=" + url.QueryEscape(createdFolder.Name)
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
	defer util.FinishBody(sharedToMeResponse.Body)
	if sharedToMeResponse.StatusCode != http.StatusOK {
		t.Logf("List retrieval failed. Unexpected status code: %d", sharedToMeResponse.StatusCode)
		t.FailNow()
	}
	var listOfObjects2 protocol.ObjectResultset
	err = util.FullDecode(sharedToMeResponse.Body, &listOfObjects2)
	if err != nil {
		t.Logf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}
	if listOfObjects2.TotalRows != 0 {
		t.Logf("Object was unexpectedly listed in tester1 shared to me when it is shared to everyone")
		t.FailNow()
	}

	t.Logf("* Verify testperson01 can get the shared object")
	getRes4, err := clientTestperson01.Do(getReq1)
	if err != nil {
		t.Logf("Unable to do get request as tp01:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(getRes4.Body)
	if getRes4.StatusCode != http.StatusOK {
		t.Logf("Unexpected status requesting object created by testperson10 as testperson01 after it was shared and acm changed: %d", getRes4.StatusCode)
		t.FailNow()
	}
	err = util.FullDecode(getRes4.Body, &retrievedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	hasEveryone = false
	for _, permission := range retrievedObject.Permissions {
		t.Logf("%s", permission)
		if permission.GroupName == models.EveryoneGroup {
			hasEveryone = true
		}
	}
	if !hasEveryone {
		t.Logf("Expected %s", models.EveryoneGroup)
		t.FailNow()
	}

}

func TestAddShareThatRevokesOwnerRead(t *testing.T) {
	t.Logf("* Create an object as tester1")
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM Revoking Owner"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"]}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := host + cfg.NginxRootURL + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{TLSClientConfig: clients[tester1].Config}
	client := &http.Client{Transport: transport}
	// exec and get response
	httpCreateResponse, err := client.Do(httpCreate)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpCreateResponse.Body)
	// check status of response
	if httpCreateResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating object: %s", httpCreateResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Resulting permissions")
	for _, permission := range createdObject.Permissions {
		t.Logf("%s", permission)
	}

	t.Logf("* Verify all clients can read it")
	uriGetProperties := host + cfg.NginxRootURL + "/objects/" + createdObject.ID + "/properties"
	httpGet, _ := http.NewRequest("GET", uriGetProperties, nil)
	for clientIdx, ci := range clients {
		transport := &http.Transport{TLSClientConfig: ci.Config}
		client := &http.Client{Transport: transport}
		httpGetResponse, err := client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
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
			for _, permission := range retrievedObject.Permissions {
				t.Logf("%s", permission)
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

	t.Logf("* Add share giving tester10 Update and Share access")
	// prep share
	var createShareRequest protocol.ObjectShare
	createShareRequest.AllowUpdate = true
	createShareRequest.AllowShare = true
	createShareRequest.Share = makeUserShare("cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us")
	// jsonify it
	jsonBody, _ = json.Marshal(createShareRequest)
	// prep http request
	uriShare := host + cfg.NginxRootURL + "/shared/" + createdObject.ID
	// prep http request
	httpCreateShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateShare.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateShareResponse, err := client.Do(httpCreateShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpCreateShareResponse.Body)
	// check status of response
	if httpCreateShareResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating share: %s", httpCreateShareResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject protocol.Object
	err = util.FullDecode(httpCreateShareResponse.Body, &updatedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify all clients can read it after tester10 given update/share permission")
	for clientIdx, ci := range clients {
		transport := &http.Transport{TLSClientConfig: ci.Config}
		client := &http.Client{Transport: transport}
		httpGetResponse, err := client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		defer util.FinishBody(httpGetResponse.Body)
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
			for _, permission := range retrievedObject.Permissions {
				t.Logf("%s", permission)
			}
		} else {
			ioutil.ReadAll(httpGetResponse.Body)
		}
		httpGetResponse.Body.Close()
	}
	if t.Failed() {
		t.FailNow()
	}

	t.Logf("* As tester10, attempt to add read share to odrive g1, which tester1 is not a member of ")
	tester10 := 0
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
	client10 := clients[tester10].Client
	httpCreateGroupShareResponse, err := client10.Do(httpCreateGroupShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpCreateGroupShareResponse.Body)
	// check status of response
	if httpCreateGroupShareResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating share: %s - Expected OK", httpCreateGroupShareResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject2 protocol.Object
	err = util.FullDecode(httpCreateGroupShareResponse.Body, &updatedObject2)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify only tester1 and members of odrive g1 can read it after odrive g1 given read/update permission")
	shouldHaveReadForObjectID(t, createdObject.ID, 0, 1, 6, 7, 8, 9)
	shouldNotHaveReadForObjectID(t, createdObject.ID, 2, 3, 4, 5, 10)
	if t.Failed() {
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	for _, permission := range updatedObject2.Permissions {
		t.Logf("%s", permission)
	}
}

func TestAddShareForCRDoesNotGiveCRUDS(t *testing.T) {
	tester10 := 0
	user_dn_ling_chen := "cn=ling chen,ou=people,ou=jitfct.twl,ou=six 3 systems,o=u.s. government,c=us"

	t.Logf("* Create an object as tester10")
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "Test Adding Share with Create & Read does not give CRUDS"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"]}`
	createObjectRequest.ContentSize = 0
	uriCreate := host + cfg.NginxRootURL + "/objects"
	httpCreate := makeHTTPRequestFromInterface(t, "POST", uriCreate, createObjectRequest)
	httpCreateResponse, err := clients[tester10].Client.Do(httpCreate)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, http.StatusOK, httpCreateResponse, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	for _, permission := range createdObject.Permissions {
		t.Logf("%s", permission)
	}

	t.Logf("* Make it private")
	var updateObjectRequest protocol.UpdateObjectRequest
	updateObjectRequest.Name = createdObject.Name + " (now private)"
	updateObjectRequest.ChangeToken = createdObject.ChangeToken
	updateObjectRequest.ID = createdObject.ID
	updateObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"users":["cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]}}`
	uriUpdateProperties := host + cfg.NginxRootURL + "/objects/" + createdObject.ID + "/properties"
	httpUpdate := makeHTTPRequestFromInterface(t, "POST", uriUpdateProperties, updateObjectRequest)
	httpUpdateResponse, err := clients[tester10].Client.Do(httpUpdate)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpUpdateResponse.Body)
	statusMustBe(t, http.StatusOK, httpUpdateResponse, "Bad status when updating object")
	var updatedObject protocol.Object
	err = util.FullDecode(httpUpdateResponse.Body, &updatedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	for _, permission := range updatedObject.Permissions {
		t.Logf("%s", permission)
	}

	t.Logf("* Share it to another user")
	var createShareRequest protocol.ObjectShare
	createShareRequest.AllowCreate = true
	createShareRequest.AllowRead = true
	createShareRequest.AllowUpdate = false
	createShareRequest.AllowDelete = false
	createShareRequest.AllowShare = false
	createShareRequest.Share = makeUserShare(user_dn_ling_chen)
	uriShare := host + cfg.NginxRootURL + "/shared/" + createdObject.ID
	httpShare := makeHTTPRequestFromInterface(t, "POST", uriShare, createShareRequest)
	httpShareResponse, err := clients[tester10].Client.Do(httpShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpShareResponse.Body)
	statusMustBe(t, http.StatusOK, httpShareResponse, "Bad status when sharing object")
	var updatedObject2 protocol.Object
	err = util.FullDecode(httpShareResponse.Body, &updatedObject2)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	for _, permission := range updatedObject2.Permissions {
		t.Logf("%s", permission)
		if strings.Compare(permission.UserDistinguishedName, user_dn_ling_chen) == 0 {
			if permission.AllowUpdate || permission.AllowDelete || permission.AllowShare {
				t.Logf("ling chen has more permissions then granted!")
				t.FailNow()
			}
		}
	}

}

func TestAddShareForRDoesNotGiveCRUDS(t *testing.T) {
	tester10 := 0
	user_dn_ling_chen := "cn=ling chen,ou=people,ou=jitfct.twl,ou=six 3 systems,o=u.s. government,c=us"

	t.Logf("* Create an object as tester10")
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "Test Adding Share with Read does not give CRUDS"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"]}`
	createObjectRequest.ContentSize = 0
	uriCreate := host + cfg.NginxRootURL + "/objects"
	httpCreate := makeHTTPRequestFromInterface(t, "POST", uriCreate, createObjectRequest)
	httpCreateResponse, err := clients[tester10].Client.Do(httpCreate)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpCreateResponse.Body)
	statusMustBe(t, http.StatusOK, httpCreateResponse, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	for _, permission := range createdObject.Permissions {
		t.Logf("%s", permission)
	}

	t.Logf("* Make it private")
	var updateObjectRequest protocol.UpdateObjectRequest
	updateObjectRequest.Name = createdObject.Name + " (now private)"
	updateObjectRequest.ChangeToken = createdObject.ChangeToken
	updateObjectRequest.ID = createdObject.ID
	updateObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"users":["cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]}}`
	uriUpdateProperties := host + cfg.NginxRootURL + "/objects/" + createdObject.ID + "/properties"
	httpUpdate := makeHTTPRequestFromInterface(t, "POST", uriUpdateProperties, updateObjectRequest)
	httpUpdateResponse, err := clients[tester10].Client.Do(httpUpdate)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpUpdateResponse.Body)
	statusMustBe(t, http.StatusOK, httpUpdateResponse, "Bad status when updating object")
	var updatedObject protocol.Object
	err = util.FullDecode(httpUpdateResponse.Body, &updatedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	for _, permission := range updatedObject.Permissions {
		t.Logf("%s", permission)
	}

	t.Logf("* Share it to another user")
	var createShareRequest protocol.ObjectShare
	createShareRequest.AllowCreate = false
	createShareRequest.AllowRead = true
	createShareRequest.AllowUpdate = false
	createShareRequest.AllowDelete = false
	createShareRequest.AllowShare = false
	createShareRequest.Share = makeUserShare(user_dn_ling_chen)
	uriShare := host + cfg.NginxRootURL + "/shared/" + createdObject.ID
	httpShare := makeHTTPRequestFromInterface(t, "POST", uriShare, createShareRequest)
	httpShareResponse, err := clients[tester10].Client.Do(httpShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(httpShareResponse.Body)
	statusMustBe(t, http.StatusOK, httpShareResponse, "Bad status when sharing object")
	var updatedObject2 protocol.Object
	err = util.FullDecode(httpShareResponse.Body, &updatedObject2)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	t.Logf("* Resulting permissions")
	for _, permission := range updatedObject2.Permissions {
		t.Logf("%s", permission)
		if strings.Compare(permission.UserDistinguishedName, user_dn_ling_chen) == 0 {
			if permission.AllowCreate || permission.AllowUpdate || permission.AllowDelete || permission.AllowShare {
				t.Logf("ling chen has more permissions then granted!")
				t.FailNow()
			}
		}
	}

}
