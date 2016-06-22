package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

func jsonEscape(i string) string {
	o := i
	o = strings.Replace(o, "\"", "\\\"", -1)
	return o
}

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

	jsonString := fmt.Sprintf(`
    {
      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" : "lol",
      "id":"deadbeef",
      "typeName": "File",
      "name": "",
      "description": "",
      "acm": "%s",
      "createdBy": "CN=POTUS,C=US"
    }
    `, jsonEscape(testhelpers.ValidACMUnclassified))
	t.Log(jsonString)
	jsonBody := []byte(jsonString)

	req, err := testhelpers.NewCreateObjectPOSTRequestRaw(
		"objects",
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

	var objResponse protocol.Object
	err = util.FullDecode(res.Body, &objResponse)
	res.Body.Close()

	t.Logf("become POTUS")
	if objResponse.CreatedBy == "CN=POTUS,C=US" {
		t.Fail()
	}
	t.Logf("set bad id")
	if objResponse.ID == "deadbeef" {
		t.Fail()
	}
}

func TestCreatObjectSimple(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	data := "Initial test data 1"
	doTestCreateObjectSimple(t, data, 5)
}

var ValidAcmCreateObjectSimple = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":["FOUO"],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U//FOUO","banner":"UNCLASSIFIED//FOUO","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

func doTestCreateObjectSimple(t *testing.T, data string, clientID int) (*http.Response, protocol.Object) {
	client := httpclients[clientID]
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
		"objects",
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

func TestCreateWithACMInObjectFormat(t *testing.T) {

	// Test originates from a sample request from Rob Olson from email
	// on 2016-06-22T03:47Z that was failing when an ACM was provided
	// in object format instead of serialized string.

	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d", clientid)
		fmt.Println()
	}

	// Attempt to post a new object
	uri := host + cfg.NginxRootURL + "/objects"
	body := `--651f24479ab34530af50aa607ce7512c
Content-Disposition: form-data; name="ObjectMetadata"
Content-Type: application/json

{
    "contentType": "image/jpeg", 
    "name": "upload.txt", 
    "isFOIAExempt": false, 
    "acm": {
        "classif": "U", 
        "atom_energy": [], 
        "disp_only": "", 
        "disponly_to": [""], 
        "dissem_ctrls": [], 
        "fgi_protect": [], 
        "f_missions": [], 
        "dissem_countries": ["USA"], 
        "sar_id": [], 
        "sci_ctrls": [], 
        "version": "2.1.0", 
        "rel_to": [], 
        "f_atom_energy": [], 
        "f_macs": [], 
        "non_ic": [], 
        "f_oc_org": [], 
        "banner": "UNCLASSIFIED", 
        "f_sci_ctrls": [], 
        "fgi_open": [], 
        "f_accms": [], 
        "f_share": [], 
        "portion": "U", 
        "f_regions": [], 
        "f_clearance": ["u"], 
        "owner_prod": []
    }, 
    "typeName": "File", 
    "isUSPersonsData": false, 
    "description": "description"
}
--651f24479ab34530af50aa607ce7512c
Content-Disposition: form-data; name="filestream"; filename="upload.txt"
Content-Type: application/octet-stream

posting a file
--651f24479ab34530af50aa607ce7512c--   
`
	byteBody := []byte(body)
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(byteBody))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=651f24479ab34530af50aa607ce7512c")

	// do the request
	res, err := httpclients[clientid].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	// process Response
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
}
