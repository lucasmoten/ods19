package server_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

func TestGetBulkProperties(t *testing.T) {
	clientid := 0

	// Create 5 folders under root
	folder0 := makeFolderViaJSON("Test Folder for Update ", clientid, t)
	folder1 := makeFolderViaJSON("Test Folder for Update ", clientid, t)
	folder2 := makeFolderViaJSON("Test Folder for Update ", clientid, t)
	folder3 := makeFolderViaJSON("Test Folder for Update ", clientid, t)
	folder4 := makeFolderViaJSON("Test Folder for Update ", clientid, t)
	folder5 := makeFolderViaJSON("Test Folder for Update ", clientid, t)

	objectIds := []string{
		folder0.ID,
		folder1.ID,
		folder2.ID,
		folder3.ID,
		folder4.ID,
		folder5.ID,
	}

	objects := protocol.ObjectIds{
		ObjectIds: objectIds,
	}

	jsonBytes, err := json.MarshalIndent(objects, "", "  ")
	if err != nil {
		t.FailNow()
	}

	uri := mountPoint + "/objects/properties"
	t.Logf("trying: POST %s", uri)
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonBytes))
	if err != nil {
		t.Logf("unable to make request for zip: %v", err)
		t.FailNow()
	}
	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName:       "Bulk Retrieve Object Properties",
			RequestDescription:  "Get a set of (existing!) objects in bulk",
			ResponseDescription: "Response to getting objects in bulk",
		},
	)
	res, err := clients[clientid].Client.Do(req)
	trafficLogs[APISampleFile].Response(t, res)
	defer util.FinishBody(res.Body)

	if res.StatusCode != http.StatusOK {
		t.Logf("wrong status code: %d", res.StatusCode)
		t.FailNow()
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Logf("failed to read data: %v", err)
		t.FailNow()
	}
	var result protocol.ObjectResultset

	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Logf("data: %s", string(data))
		t.Logf("Unable to unmarshal data: %v", err)
		t.FailNow()
	}

	objCount := len(result.Objects)
	if objCount != 6 || (objCount != result.TotalRows) {
		t.Logf("data: %s", string(data))
		t.Logf("wrong number of objects returned: %v", objCount)
		t.FailNow()
	}

}
