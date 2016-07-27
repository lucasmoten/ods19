package server_test

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"

	"io/ioutil"

	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

func doMaliciousUpdate(t *testing.T, oid, jsonString string) {
	if testing.Short() {
		t.Skip()
	}

	clientID := 5

	data := "Initial test data 2"
	//An exe name with some backspace chars to make it display as txt
	tmpName := "initialTestData2.exe\b\b\btxt"
	tmp, tmpCloser, err := testhelpers.GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer tmpCloser()

	jsonBody := []byte(jsonString)

	req, err := testhelpers.NewCreateObjectPOSTRequestRaw(
		fmt.Sprintf("objects/%s/stream", oid),
		host, "",
		tmp,
		tmpName,
		jsonBody,
	)
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
	//We expect to get a bad error code here
	if res != nil && res.StatusCode == 200 {
		t.Fail()
	}

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

func TestUpdateObjectMalicious(t *testing.T) {
	clientID := 5
	//Create an object ....
	data := "0123456789"
	_, jres := doTestCreateObjectSimple(t, data, clientID)

	if len(jres.ChangeToken) == 0 {
		t.Fail()
	}

	oid := jres.ID

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

	//Use its changeToken for an update ....
	doMaliciousUpdate(t, oid, fmt.Sprintf(jsonString, jres.ChangeToken))
}

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

	// if objResponse.RawAcm != testhelpers.ValidACMUnclassifiedFOUO {
	// 	t.Errorf("acm was not what we passed in")
	// }

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

func doReCheckProperties(t *testing.T, oid, jsonString string) {
	clientID := 5

	// XXX - properties not populated in returned value
	// so: re-retrieve the request fresh
	req, err := testhelpers.NewGetObjectRequest(oid, "", host)
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

func doPropertyUpdate(t *testing.T, oid, updateJSON string) {

	// TODO we should be able to pass this in
	clientID := 5

	data := "Initial test data 3 asdf"
	//An exe name with some backspace chars to make it display as txt
	tmpName := "initialTestData3.txt"
	tmp, tmpCloser, err := testhelpers.GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer tmpCloser()

	jsonBody := []byte(updateJSON)

	req, err := testhelpers.NewCreateObjectPOSTRequestRaw(
		fmt.Sprintf("objects/%s/stream", oid),
		host, "",
		tmp,
		tmpName,
		jsonBody,
	)
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
	jsonResponseBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("unable to read data:%v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("bad status code:%d", res.StatusCode)
	}

	doPropsCheck(t, jsonResponseBytes)
}

func TestUpdateObjectWithProperties(t *testing.T) {
	clientID := 5
	data := "0123456789"
	_, created := doTestCreateObjectSimple(t, data, clientID)

	if len(created.ChangeToken) == 0 {
		t.Fail()
	}

	acm := strings.Replace(testhelpers.ValidACMUnclassifiedFOUO, "\"", "\\\"", -1)
	//Use its changeToken for an update ....
	t.Logf("id:%s oldChangeToken:%s changeCount:%d", created.ID, created.ChangeToken, created.ChangeCount)
	doPropertyUpdate(t, created.ID, fmt.Sprintf(updateTemplate, acm, created.ChangeToken))
	//Do an independent re-retrieve
	doReCheckProperties(t, created.ID, fmt.Sprintf(updateTemplate, acm, created.ChangeToken))
}

func TestUpdateStreamWithoutProvidingACM(t *testing.T) {

	clientID := 5
	data := "0123456789"
	_, created := doTestCreateObjectSimple(t, data, clientID)
	// update object with empty string ACM
	// This function will fail internally.
	doPropertyUpdate(t, created.ID, fmt.Sprintf(updateTemplate, "", created.ChangeToken))
}

var updateTemplate = `
{
      "description": "describeit"
      ,"acm": "%s"
      ,"changeToken" : "%s"
      ,"properties" : [
          {"name":"dogname", "value":"arf", "classificationPM":"U"}
      ]
}
`
