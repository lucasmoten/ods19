package server_test

import (
	"encoding/json"
	"log"
	"net/http"
	"testing"

	"decipher.com/oduploader/protocol"
	"decipher.com/oduploader/util"
	"decipher.com/oduploader/util/testhelpers"
)

func TestCreatObjectMalicious(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	clientID := 5

	data := "Initial test data 1"
	//An exe name with some backspace chars to make it display as txt
	tmpName := "initialTestData1.exe\b\b\btxt"
	tmp, tmpCloser, err := testhelpers.GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer tmpCloser()

	//TODO: the json parsing should be returning errors with unknown fields so that
	// we limit the input grammar we accept.  (ie: completely filter out protocol.Object values
	// without having to carefully track what gets copied into ODObject for every situation).
	//
	// this should fail because of an attempt to set the creator
	jsonString := `
    {
      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" : "lol",
      "id":"deadbeef",
      "typeName": "File",
      "name": "",
      "description": "",
      "acm": "{\"version\":\"2.1.0\",\"classif\":\"S\"}",
      "createdBy": "CN=POTUS,C=US"
    }
    `
	jsonBody := []byte(jsonString)

	req, err := testhelpers.NewCreateObjectPOSTRequestRaw(
		"object",
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

	//If it comes back ok, it at least needs to have
	//stopped us from doing something bad
	if res == nil || res.StatusCode != 200 {
		t.Fail()
	}

	decoder := json.NewDecoder(res.Body)
	var objResponse protocol.Object
	err = decoder.Decode(&objResponse)
	res.Body.Close()

	log.Printf("become POTUS")
	if objResponse.CreatedBy == "CN=POTUS,C=US" {
		t.Fail()
	}
	log.Printf("set bad id")
	if objResponse.ID == "deadbeef" {
		t.Fail()
	}
}

func xTestCreatObjectSimple(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	doTestCreateObjectSimple(t)
}

var ValidAcmCreateObjectSimple = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":["FOUO"],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U//FOUO","banner":"UNCLASSIFIED//FOUO","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

func doTestCreateObjectSimple(t *testing.T) (*http.Response, protocol.Object) {
	clientID := 5
	client := httpclients[clientID]
	testName, err := util.NewGUID()
	if err != nil {
		t.Fail()
	}

	acm := ValidAcmCreateObjectSimple
	data := "Initial test data 1"
	tmpName := "initialTestData1.txt"
	tmp, tmpCloser, err := testhelpers.GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer tmpCloser()

	// TODO change this to object metadata?
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
		"object",
		host, "",
		tmp,
		tmpName,
		jsonBody,
	)
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
	return res, jres
}
