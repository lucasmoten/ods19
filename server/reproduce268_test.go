package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/protocol"
)

func TestReproduce268(t *testing.T) {

	/* AAC didn't like "classif": ["u"] so switched it to "classif": "\[\"u\"\]" */
	once := `{"dissem_countries": ["USA"], "banner": "UNCLASSIFIED", "classif": "U", "users":["%s"], "version": "2.1.0"}`
	twice := `{"dissem_countries": ["USA"], "banner": "UNCLASSIFIED", "classif": "U", "users":["%s", "%s"], "version": "2.1.0"}`

	once = fmt.Sprintf(once, fakeDN0)
	twice = fmt.Sprintf(twice, fakeDN0, fakeDN0)

	// Create object with once as raw ACM

	tester10 := 0
	c := clients[tester10].Client
	objCreate := protocol.CreateObjectRequest{
		RawAcm: once,
		Name:   newGUID(t),
	}
	f, fn, err := GenerateTempFile("somedata")
	if err != nil {
		t.Errorf("could not make file")
	}
	defer fn()
	data, err := json.Marshal(objCreate)
	failNowOnErr(t, err, "could not marshal create object as json")
	req, _ := NewCreateObjectPOSTRequestRaw(
		"objects", "", f, "somefilename", data,
	)
	resp, err := c.Do(req)
	failNowOnErr(t, err, "could not do create object request")
	t.Log("* first response")
	t.Log(resp)

	// with resp, use same response as update. Clear permissions array and permission field, override RawACM with twice
	var objResp protocol.Object
	returned, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(returned, &objResp)
	failNowOnErr(t, err, "could not unmarshal json response")
	objResp.RawAcm = twice
	objResp.Permission = protocol.Permission{}
	objResp.Permissions = make([]protocol.Permission1_0, 0)

	// Do the update
	data2, err := json.Marshal(objResp)
	failNowOnErr(t, err, "could not marshal second request from first")
	updateURL := fmt.Sprintf("/objects/%s/properties", objResp.ID)
	t.Logf("update URL: %s", updateURL)
	req2, _ := http.NewRequest("POST", mountPoint+updateURL, bytes.NewBuffer(data2))
	req2.Header.Set("Content-Type", "application/json")
	t.Logf("logging request: %v", req2)
	resp2, err := c.Do(req2)
	failNowOnErr(t, err, "could not do second request")
	t.Log("* second response")
	t.Log(resp2)

}
