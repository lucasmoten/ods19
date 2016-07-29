package server_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"testing"
	"time"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

func makeHTTPRequestFromInterface(t *testing.T, method string, uri string, obj interface{}) *http.Request {
	var requestBuffer *bytes.Buffer
	requestBuffer = nil
	if obj != nil {
		jsonBody, err := json.Marshal(obj)
		if err != nil {
			t.Logf("Unable to marshal json for request: %v", err)
			t.FailNow()
		}
		requestBuffer = bytes.NewBuffer(jsonBody)
	} else {
		requestBuffer = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, uri, requestBuffer)
	if err != nil {
		t.Logf("Error setting up HTTP request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

func makeUserShare(userDN string) interface{} {
	shareString := fmt.Sprintf(`{"users":["%s"]}`, userDN)
	var shareInterface interface{}
	json.Unmarshal([]byte(shareString), &shareInterface)
	return shareInterface
}

func makeGroupShare(project string, displayName string, groupName string) interface{} {
	shareString := fmt.Sprintf(`{"projects":{"%s":{"disp_nm":"%s","groups":["%s"]}}}`, project, displayName, groupName)
	var shareInterface interface{}
	json.Unmarshal([]byte(shareString), &shareInterface)
	return shareInterface
}

func doAddObjectShare(t *testing.T, obj *protocol.Object, share *protocol.ObjectShare, clientid int) *protocol.Object {
	shareuri := host + cfg.NginxRootURL + "/shared/" + obj.ID
	req := makeHTTPRequestFromInterface(t, "POST", shareuri, share)
	res, err := clients[clientid].Client.Do(req)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, res, "Bad status when creating share")
	var updatedObject protocol.Object
	err = util.FullDecode(res.Body, &updatedObject)
	failNowOnErr(t, err, "Error decoding json to Object")
	return &updatedObject
}

func makeFolderWithACMViaJSON(folderName string, rawAcm string, clientid int) (*protocol.Object, error) {
	folderuri := host + cfg.NginxRootURL + "/objects"
	folder := protocol.Object{}
	folder.Name = folderName
	folder.TypeName = "Folder"
	folder.ParentID = ""
	folder.RawAcm = rawAcm
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		return nil, err
	}
	req, err := http.NewRequest("POST", folderuri, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		return nil, err
	}
	// do the request
	req.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{TLSClientConfig: clients[clientid].Config}
	client := &http.Client{Transport: transport}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		return nil, err
	}
	defer util.FinishBody(res.Body)
	// process Response
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		return nil, errors.New("Status was " + res.Status)
	}
	var createdFolder protocol.Object
	err = util.FullDecode(res.Body, &createdFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		return nil, err
	}
	return &createdFolder, nil
}

func makeFolderViaJSON(folderName string, clientid int, t *testing.T) *protocol.Object {

	nameWithTimestamp := folderName + strconv.FormatInt(time.Now().Unix(), 10)

	obj, err := makeFolderWithACMViaJSON(nameWithTimestamp, testhelpers.ValidACMUnclassified, clientid)
	if err != nil {
		t.Errorf("Error creating folder %s: %v\n", folderName, err)
		t.FailNow()
	}
	return obj
}

func doTestCreateObjectSimple(t *testing.T, data string, clientID int) (*http.Response, protocol.Object) {
	client := clients[clientID].Client
	testName, err := util.NewGUID()
	if err != nil {
		t.Fail()
	}

	acm := ValidAcmCreateObjectSimple
	tmpName := "initialTestData1.txt"
	tmp, tmpCloser, err := testhelpers.GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer tmpCloser()

	createRequest := protocol.CreateObjectRequest{
		Name:     testName,
		TypeName: "File",
		RawAcm:   acm,
	}

	var jsonBody []byte
	jsonBody, err = json.Marshal(createRequest)
	if err != nil {
		t.Fail()
	}

	req, err := testhelpers.NewCreateObjectPOSTRequestRaw(
		"objects", host, "", tmp, tmpName, jsonBody)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}

	res, jresif, err := testhelpers.DoWithDecodedResult(client, req)

	if err != nil {
		t.Fail()
	}

	if res != nil && res.StatusCode != http.StatusOK {
		t.Fail()
	}

	jres := jresif.(protocol.Object)
	doCheckFileNowExists(t, clientID, jres)

	return res, jres
}

func doCheckFileNowExists(t *testing.T, clientID int, jres protocol.Object) {

	//!!! STOP and verify that this object actually exists in listings!
	//Because we *just* created this, we expect it to be in first page!
	uri2 := host + cfg.NginxRootURL + "/objects"

	if jres.ParentID != "" {
		uri2 = uri2 + "/" + jres.ParentID
	}
	uri2 = uri2 + "?pageSize=10000"

	req2, err := http.NewRequest("GET", uri2, nil)
	if err != nil {
		t.Log("get create fail")
		t.FailNow()
	}
	res2, err := clients[clientID].Client.Do(req2)
	if err != nil {
		t.Log("client connect fail")
		t.FailNow()
	}
	defer util.FinishBody(res2.Body)
	var objectResultSet protocol.ObjectResultset
	bodyBytes, err := ioutil.ReadAll(res2.Body)
	if err != nil {
		t.Log("got no body back")
		t.FailNow()
	}
	json.Unmarshal(bodyBytes, &objectResultSet)
	foundIt := false
	for _, v := range objectResultSet.Objects {
		if v.ID == jres.ID {
			foundIt = true
		}
		t.Logf("lookfor: %s id:%s", jres.ID, v.ID)
	}
	if len(objectResultSet.Objects) == 0 {
		t.Logf("no objects in listing for %s", uri2)
	}
	if !foundIt {
		t.Logf("did not find object that we just created as user %d", clientID)
		t.FailNow()
	}
}

func shouldHaveReadForObjectID(t *testing.T, objID string, clientIdxs ...int) {
	uri := host + cfg.NginxRootURL + "/objects/" + objID + "/properties"
	getReq, _ := http.NewRequest("GET", uri, nil)
	for _, i := range clientIdxs {
		// reaches for package global clients
		c := clients[i].Client
		resp, err := c.Do(getReq)
		failNowOnErr(t, err, "Unable to do request")
		defer util.FinishBody(resp.Body)
		statusExpected(t, 200, resp, fmt.Sprintf("client id %d should have read for ID %s", i, objID))
		ioutil.ReadAll(resp.Body)
	}
}

func shouldNotHaveReadForObjectID(t *testing.T, objID string, clientIdxs ...int) {
	uri := host + cfg.NginxRootURL + "/objects/" + objID + "/properties"
	getReq, _ := http.NewRequest("GET", uri, nil)
	for _, i := range clientIdxs {
		// reaches for package global clients
		c := clients[i].Client
		resp, err := c.Do(getReq)
		failNowOnErr(t, err, "Unable to do request")
		defer util.FinishBody(resp.Body)
		statusExpected(t, 403, resp, fmt.Sprintf("client id %d should not have read for ID %s", i, objID))
		ioutil.ReadAll(resp.Body)
	}
}

// failNowOnErr fails immediately with a templated message.
func failNowOnErr(t *testing.T, err error, msg string) {
	if err != nil {
		t.Logf("%s: %v", msg, err)
		t.FailNow()
	}
}

func statusMustBe(t *testing.T, expected int, resp *http.Response, msg string) {
	statusExpected(t, expected, resp, msg)
	if t.Failed() {
		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		t.FailNow()
	}
}

func statusExpected(t *testing.T, expected int, resp *http.Response, msg string) {
	if resp.StatusCode != expected {
		if msg != "" {
			t.Logf(msg)
		}
		t.Logf("Expected status %v but got %v", expected, resp.StatusCode)
		t.Logf("%s", resp.Status)
		t.Fail()
	}
}
