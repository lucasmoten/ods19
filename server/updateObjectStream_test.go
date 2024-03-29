package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

func TestUpdateObjectStreamWithMismatchedIDs(t *testing.T) {
	t.Logf("Create new test object with stream")
	clientID := 6
	data, _ := util.NewGUID()
	f, closer, err := GenerateTempFile(data)
	defer closer()

	req, err := NewCreateObjectPOSTRequest("", f)
	failNowOnErr(t, err, "could not create a createObject POST request")

	res, err := clients[clientID].Client.Do(req)

	failNowOnErr(t, err, "unable to do createObject request")
	statusMustBe(t, 200, res, "expected status 200 when creating object")
	var obj protocol.Object
	err = util.FullDecode(res.Body, &obj)
	failNowOnErr(t, err, "could not decode object response")
	defer res.Body.Close()

	correctID := obj.ID

	t.Logf("Try to update object by changing to a bad URL")
	wrongID, _ := util.NewGUID()
	badUpdateURI := mountPoint + fmt.Sprintf("/objects/%s/stream", wrongID)
	buf, boundary := NewMultipartRequestBody(t, obj, f)
	updateReq, _ := http.NewRequest("POST", badUpdateURI, buf)
	updateReq.Header.Set("Content-Type", boundary)
	t.Logf("  boundary is %s", boundary)

	updateResp, err := clients[clientID].Client.Do(updateReq)
	failNowOnErr(t, err, "could not do stream update request")
	statusMustBe(t, 404, updateResp, "expected 404 due to invalid id in URI")

	t.Logf("Try to update object by changing to a bad JSON id")
	obj.ID = wrongID
	goodUpdateURI := mountPoint + fmt.Sprintf("/objects/%s/stream", correctID)
	buf, boundary = NewMultipartRequestBody(t, obj, f)
	updateReq, _ = http.NewRequest("POST", goodUpdateURI, buf)
	updateReq.Header.Set("Content-Type", boundary)
	t.Logf("  boundary is %s", boundary)

	updateResp, err = clients[clientID].Client.Do(updateReq)
	failNowOnErr(t, err, "could not do stream update request")
	statusMustBe(t, 400, updateResp, "expected 400 due to invalid id in update request payload")

	//The whitespace in this string matters for the attack that previously crashed the server
	t.Logf("Send a form-data form on a multipart/form-data without boundaries")
	buf2 := `

Content-Disposition: form-data; name="ObjectMetadata"

{"id":"%s","changeToken":"%s","name":"My New Name"}
`
	t.Logf("Try to update the object with a malformed request")
	req, _ = http.NewRequest(
		"POST",
		goodUpdateURI,
		bytes.NewBuffer([]byte(fmt.Sprintf(buf2, correctID, obj.ChangeToken))),
	)
	req.Header.Add("Content-Type", "multipart/form-data")
	res, err = clients[clientID].Client.Do(req)
	statusMustBe(t, 400, res, "expected to catch a bad multipart encode")

	//The whitespace in this string matters for the attack that previously crashed the server
	//Send a form-data form on a multipart/form-data without boundaries
	t.Logf("Send a form-data form on a multipart/form-data without no file part")
	buf3 := `

--XXXX
Content-Disposition: form-data; name="ObjectMetadata"

{"id":"%s","changeToken":"%s","name":"My New Name"}

--XXXX--
`
	t.Logf("Try to update the object with a malformed request")
	req, _ = http.NewRequest("POST",
		goodUpdateURI,
		bytes.NewBuffer([]byte(fmt.Sprintf(buf3, correctID, obj.ChangeToken))),
	)
	req.Header.Add("Content-Type", "multipart/form-data; boundary=XXXX")
	res, err = clients[clientID].Client.Do(req)
	statusMustBe(t, 400, res, "expected to catch a bad multipart encode")

}

func TestUpdateObjectMalicious(t *testing.T) {
	clientID := 5
	data := "0123456789"
	_, obj := doTestCreateObjectSimple(t, data, clientID, nil, nil, ValidAcmCreateObjectSimple)
	doCheckFileNowExists(t, clientID, obj)

	if len(obj.ChangeToken) == 0 {
		t.FailNow()
	}

	jsonString := `
    {
      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" : "lol",
      "id":"deadbeef",
      "typeName": "File",
      "name": "",
      "description": "",
      "acm": "{\"version\":\"2.1.0\",\"classif\":\"U\"}",
      "createdBy": "CN=POTUS,C=US",
      "changeToken" : "%s"
    }
    `

	doMaliciousUpdate(t, obj.ID, fmt.Sprintf(jsonString, obj.ChangeToken))
}

func TestUpdateObjectWithProperties(t *testing.T) {
	clientID := 5
	data := "0123456789"
	_, created := doTestCreateObjectSimple(t, data, clientID, nil, nil, ValidAcmCreateObjectSimple)
	doCheckFileNowExists(t, clientID, created)

	if len(created.ChangeToken) == 0 {
		t.FailNow()
	}

	// NOTE: do we need to do string escaping here?
	acm := strings.Replace(ValidACMUnclassifiedFOUO, "\"", "\\\"", -1)
	t.Logf("Use changetoken for update. id:%s oldChangeToken:%s changeCount:%d", created.ID, created.ChangeToken, created.ChangeCount)
	doPropertyUpdate(t, clientID, created.ID, fmt.Sprintf(updateTemplate, created.ID, acm, created.ChangeToken),
		trafficLogs[APISampleFile],
		&TrafficLogDescription{
			OperationName: "Modify Object Property",
			RequestDescription: `
				Update a property.  It is required to pass in changeToken as proof that we have seen the latest version.
				`,
			ResponseDescription: `
				We get back an object with updated properties,
				with changeToken and changeCount being important changed values.`,
		},
	)
	doReCheckProperties(t, created.ID, fmt.Sprintf(updateTemplate, created.ID, acm, created.ChangeToken))
}

func TestUpdateStream(t *testing.T) {
	clientID := 5
	data, _ := util.NewGUID()
	newName, _ := util.NewGUID()
	_, created := doTestCreateObjectSimple(t, data, clientID,
		trafficLogs[APISampleFile],
		&TrafficLogDescription{
			OperationName: "Update the stream of an existing object",
			RequestDescription: `<p>
				A POST request into the server requires a json part followed by a multi-part body.
				The json part content-type must be application/json, and the body part is specified
				by the caller, unless the caller is ok with the server making a guess based on
				the file extension specified in the name part of the json part.
				The identifier in the URI should have come back from a prior create request.
				It is critical to use a multipart/form-data mime type with a boundary, and to
				send the json first and call it ObjectMetadata.  The next part must be the bytes for
				the file.  This part must come second because it could be very large.
				</p>`,
			ResponseDescription: `We get back an object with updated json changeToken and version`,
		},
		ValidAcmCreateObjectSimple,
	)
	doCheckFileNowExists(t, clientID, created)

	created.Name = newName
	updated := doUpdateStreamForObjectID(t, clientID, created.ID, created)
	if updated.Name != newName {
		t.Errorf("Expected name to be updated")
		t.FailNow()
	}

}

func TestUpdateStreamWithoutProvidingACM(t *testing.T) {

	clientID := 5
	data := "0123456789"

	_, created := doTestCreateObjectSimple(t, data, clientID, nil, nil, ValidAcmCreateObjectSimple)
	doCheckFileNowExists(t, clientID, created)

	doPropertyUpdate(t, clientID, created.ID, fmt.Sprintf(updateTemplate, created.ID, "", created.ChangeToken), nil, nil)
}

func TestUpdateStreamWithoutProvidingACMButHasStream(t *testing.T) {

	clientID := 0
	var created protocol.Object
	var err error

	createAsFile := false
	acm := `{"banner":"UNCLASSIFIED","classif":"U","dissem_countries":["USA"],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":["cntesttester10oupeopleoudaeouchimeraou_s_governmentcus"],"portion":"U","share":{"projects":null,"users":["CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US","cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`

	if createAsFile {
		data := "0123456789"
		_, created = doTestCreateObjectSimple(t, data, clientID, nil, nil, acm)
		doCheckFileNowExists(t, clientID, created)
	} else {
		var topFolder *protocol.Object
		topFolder, err = makeFolderWithACMViaJSON("UpdateStreamNoACM-Top Folder", acm, clientID)
		if err != nil {
			t.Logf("Error making topfolder in test: %v", err)
			t.FailNow()
		}
		var childFolder *protocol.Object
		childFolder, err = makeFolderWithACMWithParentViaJSON("UpdateStreamNoACM-Object In Top Folder", topFolder.ID, acm, clientID)
		if err != nil {
			t.Logf("Error making childfolder in test: %v", err)
			t.FailNow()
		}
		created = *childFolder
	}

	// note that parentid and permissions fields will be ignored
	updateTemplate := `{"id":"%s","changeToken":"%s","name":"%s","parentId":"%s","permissions":[]}`
	updateMeta := fmt.Sprintf(updateTemplate, created.ID, created.ChangeToken, created.Name, created.ParentID)
	var jsonBody []byte
	jsonBody = []byte(updateMeta)
	updateURISuffix := "objects/" + created.ID + "/stream"
	data2 := `{"x":"123"}`
	tmpName := "_capco_favorites.json"
	f, closer, err := GenerateTempFile(data2)
	defer closer()
	req, err := NewCreateObjectPOSTRequestRaw(updateURISuffix, "", f, tmpName, jsonBody)

	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}

	client := clients[clientID].Client
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	if res == nil {
		t.Fail()
	}

	if res.StatusCode != 200 {
		t.Logf("Status was %d", res.StatusCode)
		t.Logf("Message was %s", res.Status)
		t.FailNow()
	}

	var objResponse protocol.Object
	err = util.FullDecode(res.Body, &objResponse)
	res.Body.Close()

}

var updateTemplate = `
{
	  "id": "%s",
      "description": "describeit",
	  "acm": "%s",
	  "changeToken" : "%s",
	  "properties" : [
          {"name":"dogname", "value":"arf", "classificationPM":"U"}
      ]
}
`

func doPropsCheck(t *testing.T, jsonResponseBytes []byte) {
	jsonResponse := string(jsonResponseBytes)
	decoder := json.NewDecoder(strings.NewReader(jsonResponse))
	var objResponse protocol.Object
	err := decoder.Decode(&objResponse)
	if err != nil {
		t.Errorf("unable to decode response:%s", jsonResponse)
	}

	t.Logf("id:%s newChangeToken:%s changeCount:%d", objResponse.ID, objResponse.ChangeToken, objResponse.ChangeCount)

	if objResponse.Description != "describeit" {
		t.Errorf("objResponse was expected to be 'describeit'")
	}

	acmRaw := objResponse.RawAcm
	acmMap, ok := acmRaw.(map[string]interface{})
	if !ok {
		t.Errorf("Unable to convert ACM in response to map")
	}
	if acmMap["banner"] == nil {
		t.Errorf("ACM returned does not have a banner")
	}
	acmBanner := acmMap["banner"].(string)
	if acmBanner != "UNCLASSIFIED//FOUO" {
		t.Errorf("acm did not have expected banner value")
	}

	if len(objResponse.Properties) == 0 {
		t.Logf("We did not get properties coming back in: %s", jsonResponse)
	}
	if objResponse.Properties[0].Name != "dogname" && objResponse.Properties[0].Value != "arf" && objResponse.Properties[0].ClassificationPM != "U" {
		t.Logf("We did not get a match on properties")
	}
}

func doMaliciousUpdate(t *testing.T, oid, jsonString string) {
	if testing.Short() {
		t.Skip()
	}

	clientID := 5

	data := "Initial test data 2"
	// An exe name with some backspace chars to make it display as txt
	tmpName := "initialTestData2.exe\b\b\btxt"
	tmp, tmpCloser, err := GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer tmpCloser()

	jsonBody := []byte(jsonString)

	req, err := NewCreateObjectPOSTRequestRaw(
		fmt.Sprintf("objects/%s/stream", oid), "", tmp, tmpName, jsonBody)
	failNowOnErr(t, err, "unable to create HTTP request")

	res, err := clients[clientID].Client.Do(req)
	failNowOnErr(t, err, "unable to do request")
	statusMustBe(t, 400, res, "expected create object from doMaliciousUpdate to fail")
	defer util.FinishBody(res.Body)

	var objResponse protocol.Object
	err = util.FullDecode(res.Body, &objResponse)
	res.Body.Close()

	if objResponse.CreatedBy == "CN=POTUS,C=US" {
		log.Printf("checking to see if we are now POTUS")
		t.Fail()
	}
	if objResponse.ID == "deadbeef" {
		log.Printf("checking to see if we modified the id")
		t.Fail()
	}
}

func doPropertyUpdate(
	t *testing.T,
	clientID int,
	oid string,
	updateJSON string,
	trafficLog *TrafficLog,
	description *TrafficLogDescription,
) {

	data := "Initial test data 3 asdf"
	tmpName := "initialTestData3.txt"
	tmp, tmpCloser, err := GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer tmpCloser()

	jsonBody := []byte(updateJSON)
	urlPath := fmt.Sprintf("objects/%s/stream", oid)
	req, err := NewCreateObjectPOSTRequestRaw(urlPath, "", tmp, tmpName, jsonBody)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}
	if trafficLog != nil {
		trafficLog.Request(t, req, description)
	}

	client := clients[clientID].Client
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	if trafficLog != nil {
		trafficLog.Response(t, res)
	}
	defer util.FinishBody(res.Body)
	jsonResponseBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("unable to read data:%v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("bad status code:%d", res.StatusCode)
	}

	doPropsCheck(t, jsonResponseBytes)
}

func doUpdateStreamForObjectID(t *testing.T, clientID int, oid string, newObj protocol.Object) protocol.Object {

	req := NewUpdateObjectStreamPOSTRequest(t, newObj)

	res, err := clients[clientID].Client.Do(req)
	failNowOnErr(t, err, "could not do update stream request")
	defer util.FinishBody(res.Body)

	statusMustBe(t, 200, res, "bad status calling updateObjectStream")

	jsonResponseBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("unable to read data:%v", err)
	}

	var updated protocol.Object

	err = json.Unmarshal(jsonResponseBytes, &updated)
	failNowOnErr(t, err, "could not unmarshal response from updateObjectStream")

	return updated
}

func doReCheckProperties(t *testing.T, oid, jsonString string) {
	// TODO try to remove hardcoding here so we can have stream tests that
	// use more than one client ID.
	clientID := 5

	req, err := NewGetObjectRequest(oid, "")
	if err != nil {
		t.Logf("Unable to generate get re-request:%v", err)
	}
	client := clients[clientID].Client
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("Unable to do re-request:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	jsonResponseBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("unable to read data:%v", err)
	}
	res.Body.Close()

	doPropsCheck(t, jsonResponseBytes)
}
