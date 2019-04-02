package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

var mountPoint = util.GetClientMountPoint()

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
	shareString := `{` + makeUserShareString(userDN) + `}`
	var shareInterface interface{}
	json.Unmarshal([]byte(shareString), &shareInterface)
	return shareInterface
}
func makeUserShareString(userDN string) string {
	shareString := fmt.Sprintf(`"users":["%s"]`, userDN)
	return shareString
}

func makeGroupShare(project string, displayName string, groupName string) interface{} {
	shareString := `{` + makeGroupShareString(project, displayName, groupName) + `}`
	var shareInterface interface{}
	json.Unmarshal([]byte(shareString), &shareInterface)
	return shareInterface
}
func makeGroupShareString(project string, displayName string, groupName string) string {
	shareString := fmt.Sprintf(`"projects":{"%s":{"disp_nm":"%s","groups":["%s"]}}`, project, displayName, groupName)
	return shareString
}

func doAddObjectShare(t *testing.T, obj *protocol.Object, share *protocol.ObjectShare, clientid int) *protocol.Object {
	shareuri := mountPoint + "/shared/" + obj.ID
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
	return makeFolderWithACMWithParentViaJSON(folderName, "", rawAcm, clientid)
}

func makeFolderWithACMWithParentViaJSON(folderName string, parentID string, rawAcm string, clientid int) (*protocol.Object, error) {
	folderuri := mountPoint + "/objects"
	folder := protocol.Object{}
	folder.Name = folderName
	folder.TypeName = "Folder"
	folder.ParentID = parentID
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
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		return nil, err
	}
	defer util.FinishBody(res.Body)
	// process Response
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		htmlData, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("Status was %s. Body: %s", res.Status, string(htmlData))
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
	return makeFolderWithParentViaJSON(folderName, "", clientid, t)
}

func makeFolderWithParentViaJSON(folderName string, parentID string, clientid int, t *testing.T) *protocol.Object {

	nameWithTimestamp := folderName + strconv.FormatInt(time.Now().UTC().Unix(), 10)
	obj, err := makeFolderWithACMWithParentViaJSON(nameWithTimestamp, parentID, ValidACMUnclassified, clientid)

	if err != nil {
		t.Errorf("Error creating folder %s: %v\n", folderName, err)
		t.FailNow()
	}
	return obj
}

func listChildren(parentID string, clientid int, t *testing.T) *protocol.ObjectResultset {
	uri := mountPoint + "/objects/" + parentID
	req, _ := http.NewRequest("GET", uri, nil)
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// process Response
	if res.StatusCode != http.StatusOK {
		t.Errorf("bad status: %s", res.Status)
		htmlData, _ := ioutil.ReadAll(res.Body)
		t.Errorf("Status was %s. Body: %s", res.Status, string(htmlData))
	}
	var resultset protocol.ObjectResultset
	err = util.FullDecode(res.Body, &resultset)
	if err != nil {
		t.Errorf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}
	return &resultset
}

func getObject(id string, clientid int, t *testing.T) *protocol.Object {
	uri := mountPoint + "/objects/" + id + "/properties"
	req, _ := http.NewRequest("GET", uri, nil)
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// process Response
	if res.StatusCode != http.StatusOK {
		t.Errorf("bad status: %s", res.Status)
		htmlData, _ := ioutil.ReadAll(res.Body)
		t.Errorf("Status was %s. Body: %s", res.Status, string(htmlData))
	}
	var result protocol.Object
	err = util.FullDecode(res.Body, &result)
	if err != nil {
		t.Errorf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	return &result
}

// DoWithDecodedResult is the common case of getting back a json response that is ok
// Need to have one that isn't closed yet so we can dump out to traffic log
// The _test package structure is preventing just moving this into the testhelpers
func DoWithDecodedResult(
	t *testing.T,
	client *http.Client,
	req *http.Request,
	trafficLog *TrafficLog,
	description *TrafficLogDescription,
) (*http.Response, interface{}, error) {
	if trafficLog != nil {
		trafficLog.Request(t, req, description)
	}
	var objResponse protocol.Object
	res, err := client.Do(req)
	if err != nil {
		return nil, objResponse, err
	}
	if trafficLog != nil {
		trafficLog.Response(t, res)
	}
	d := json.NewDecoder(res.Body)
	err = d.Decode(&objResponse)
	if err != nil {
		log.Printf("error decoding response: %v", err)
	}
	util.FinishBody(res.Body)
	res.Body.Close()
	return res, objResponse, err
}

func doTestCreateObjectSimple(
	t *testing.T,
	data string,
	clientID int,
	trafficLog *TrafficLog,
	description *TrafficLogDescription,
	acmString string,
) (*http.Response, protocol.Object) {
	return doTestCreateObjectSimpleWithType(t, data, clientID, trafficLog, description, acmString, "File")
}

func doTestCreateObjectSimpleWithType(
	t *testing.T,
	data string,
	clientID int,
	trafficLog *TrafficLog,
	description *TrafficLogDescription,
	acmString string,
	typeName string,
) (*http.Response, protocol.Object) {
	client := clients[clientID].Client
	testName, err := util.NewGUID()
	if err != nil {
		t.Fail()
	}

	var acm interface{}
	json.Unmarshal([]byte(acmString), &acm)
	tmpName := "initialTestData1.txt"
	tmp, tmpCloser, err := GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer tmpCloser()

	createRequest := protocol.CreateObjectRequest{
		Name:     testName,
		TypeName: typeName,
		RawAcm:   acm,
	}

	var jsonBody []byte
	jsonBody, err = json.MarshalIndent(createRequest, "", "  ")
	if err != nil {
		t.Logf("failed request: %v", err)
		t.Fail()
	}

	req, err := NewCreateObjectPOSTRequestRaw(
		"objects", "", tmp, tmpName, jsonBody)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}

	res, untyped, err := DoWithDecodedResult(t, client, req, trafficLog, description)
	if err != nil {
		t.Fail()
	}

	statusExpected(t, 200, res, "doTestCreateObjectSimple internal failure")
	obj := untyped.(protocol.Object)

	return res, obj
}

func doTestUpdateObjectSimple(
	t *testing.T,
	data string,
	clientID int,
	oldObject protocol.Object,
	trafficLog *TrafficLog,
	description *TrafficLogDescription,
	acmString string,
) (*http.Response, protocol.Object) {
	client := clients[clientID].Client
	testName, err := util.NewGUID()
	if err != nil {
		t.Fail()
	}

	var acm interface{}
	json.Unmarshal([]byte(acmString), &acm)
	tmpName := "initialTestData1.txt"
	tmp, tmpCloser, err := GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer tmpCloser()

	createRequest := protocol.UpdateObjectRequest{
		ID:          oldObject.ID,
		ChangeToken: oldObject.ChangeToken,
		Name:        testName,
		TypeName:    "File",
		RawAcm:      acm,
	}

	var jsonBody []byte
	jsonBody, err = json.MarshalIndent(createRequest, "", "  ")
	if err != nil {
		t.Fail()
	}

	req, err := NewCreateObjectPOSTRequestRaw(
		"objects/"+oldObject.ID+"/stream", "", tmp, tmpName, jsonBody)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}

	res, untyped, err := DoWithDecodedResult(t, client, req, trafficLog, description)
	if err != nil {
		t.Fail()
	}

	statusExpected(t, 200, res, "doTestUpdateObjectSimple internal failure")
	obj := untyped.(protocol.Object)

	return res, obj
}

func doCheckFileNowExists(t *testing.T, clientID int, jres protocol.Object) {

	uri2 := mountPoint + "/objects"
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
		if testing.Verbose() {
			t.Logf("lookfor: %s id:%s", jres.ID, v.ID)
		}
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
	uri := mountPoint + "/objects/" + objID + "/properties"
	getReq, _ := http.NewRequest("GET", uri, nil)
	for _, i := range clientIdxs {
		// reaches for package global clients
		c := clients[i].Client
		resp, err := c.Do(getReq)
		failNowOnErr(t, err, "Unable to do request")
		defer util.FinishBody(resp.Body)
		statusExpected(t, 200, resp, fmt.Sprintf("client id %d should have read for ID %s", i, objID))
		data, _ := ioutil.ReadAll(resp.Body)
		var obj protocol.Object
		err = json.Unmarshal(data, &obj)
		failNowOnErr(t, err, "could not Unmarshal json response")
		if !obj.CallerPermission.AllowRead {
			t.Errorf("expected READ on CallerPermission to be true")
		}

	}
}

func shouldNotHaveReadForObjectID(t *testing.T, objID string, clientIdxs ...int) {
	uri := mountPoint + "/objects/" + objID + "/properties"
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

func expectingReadForObjectIDVersion(t *testing.T, code int, version int, objID string, clientIdxs ...int) {
	uri := mountPoint + "/revisions/" + objID + "/" + strconv.Itoa(version) + "/stream"
	getReq, _ := http.NewRequest("GET", uri, nil)
	for _, i := range clientIdxs {
		// reaches for package global clients
		c := clients[i].Client
		resp, err := c.Do(getReq)
		failNowOnErr(t, err, "Unable to do request")
		defer util.FinishBody(resp.Body)
		statusExpected(t, code, resp, fmt.Sprintf("client id %d should get http %d ID %s version %d using %s", i, code, objID, version, uri))
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
	if resp == nil {
		t.Logf("Expected status %v but got no response", expected)
		t.Fail()
	}
	if resp.StatusCode != expected {
		if msg != "" {
			t.Logf(msg)
		}
		t.Logf("Expected status %v but got %v", expected, resp.StatusCode)
		t.Logf("%s", resp.Status)
		t.Fail()
	}
}

func messageMustContain(t *testing.T, res *http.Response, lookFor string) {
	msg, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("fail to read body: %s", err.Error())
	}
	if !strings.Contains(string(msg), lookFor) {
		t.Errorf("was looking for '%s', but found '%s'", lookFor, string(msg))
	}
}
func allTrue(vals ...bool) bool {
	for _, v := range vals {
		if !v {
			return false
		}
	}
	return true
}

func jsonEscape(i string) string {
	o := i
	o = strings.Replace(o, "\"", "\\\"", -1)
	return o
}
