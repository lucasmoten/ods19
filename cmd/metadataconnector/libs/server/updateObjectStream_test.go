package server_test

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"decipher.com/oduploader/protocol"
	"decipher.com/oduploader/util/testhelpers"
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
		fmt.Sprintf("object/%s/stream", oid),
		host, "",
		tmp,
		tmpName,
		jsonBody,
	)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}

	client := httpclients[clientID]
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}

	//We expect to get a bad error code here
	if res != nil && res.StatusCode == 200 {
		t.Fail()
	}

	decoder := json.NewDecoder(res.Body)
	var objResponse protocol.Object
	err = decoder.Decode(&objResponse)
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

	//Create an object ....
	_, jres := doTestCreateObjectSimple(t)

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
