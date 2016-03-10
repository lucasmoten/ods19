package server_test

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"decipher.com/oduploader/protocol"
	"decipher.com/oduploader/util/testhelpers"
)

func TestHTTPUndeleteObject(t *testing.T) {

	data := "Deletes are hard. Undeletes are harder!"

	if testing.Short() {
		t.Skip()
	}

	clientID := 6

	tmp, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}

	tmp.WriteString(data)
	defer func() {
		name := tmp.Name()
		tmp.Close()
		err = os.Remove(name)
	}()

	req, err := testhelpers.NewCreateObjectPOSTRequest(host, "", tmp)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}
	res, err := httpclients[clientID].Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	decoder := json.NewDecoder(res.Body)
	var objResponse protocol.Object

	err = decoder.Decode(&objResponse)
	if err != nil {
		t.Errorf("Could not decode CreateObject response.")
	}
	res.Body.Close()
	// Save out the objectID as a string
	objID := objResponse.ID

	expected := objResponse.Name

	deleteReq, err := testhelpers.NewDeleteObjectRequest(objResponse, "", host)
	res, err = httpclients[clientID].Do(deleteReq)
	if err != nil {
		t.Errorf("Delete request failed: %v\n", err)
	}
	res.Body.Close()

	// We must do another GET to get a valid changeToken
	getReq, err := testhelpers.NewGetObjectRequest(objID, "", host)
	if err != nil {
		t.Errorf("Error from NewGetObjectRequest: %v\n", err)
	}
	res, err = httpclients[clientID].Do(getReq)
	if err != nil {
		t.Errorf("GetObject request failed: %v\n", err)
	}
	decoder = json.NewDecoder(res.Body)
	var getObjectResponse protocol.Object
	err = decoder.Decode(&getObjectResponse)
	if err != nil {
		log.Printf("Error decoding json response from getObject to Object: %v", err)
		t.FailNow()
	}
	res.Body.Close()

	// This must be passed valid change token
	undeleteReq, err := testhelpers.NewUndeleteObjectPUTRequest(
		objID, getObjectResponse.ChangeToken, "", host)
	res, err = httpclients[clientID].Do(undeleteReq)
	if err != nil {
		t.Errorf("Delete request failed: %v\n", err)
	}

	// Assert object has been undeleted.
	decoder = json.NewDecoder(res.Body)
	var unDeletedObject protocol.Object
	err = decoder.Decode(&unDeletedObject)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	res.Body.Close()
	//t.Log("UndeletedObject: ", unDeletedObject)

	if unDeletedObject.Name != expected {
		t.Errorf("Expected returned object name to be %v. Got: %v\n",
			expected, unDeletedObject.Name)
	}

}
