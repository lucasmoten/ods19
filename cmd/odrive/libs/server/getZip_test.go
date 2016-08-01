package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/util"

	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util/testhelpers"
)

// Add a file into the zip file
func testZipMakeFile(t *testing.T, clientID int, parentID string, name string, data string) protocol.Object {
	client := clients[clientID].Client

	createRequest := protocol.CreateObjectRequest{
		Name:     name,
		TypeName: "File",
		RawAcm:   ValidAcmCreateObjectSimple,
		ParentID: parentID,
	}

	var jsonBody []byte
	var err error
	jsonBody, err = json.Marshal(createRequest)
	if err != nil {
		t.Fail()
	}

	tmpName := name
	tmp, tmpCloser, err := testhelpers.GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer tmpCloser()

	req, err := testhelpers.NewCreateObjectPOSTRequestRaw(
		"objects", host, "", tmp, tmpName, jsonBody)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}

	res, obj, err := testhelpers.DoWithDecodedResult(client, req)

	if err != nil {
		t.Fail()
	}

	if res != nil && res.StatusCode != http.StatusOK {
		t.Fail()
	}

	return obj.(protocol.Object)
}

func TestZip(t *testing.T) {
	tester10 := 0
	client := clients[tester10].Client

	//Make some stuff to zip up
	mapsFolder, err := makeFolderWithACMWithParentViaJSON("maps", "", ValidAcmCreateObjectSimple, tester10)
	if err != nil {
		t.Logf("failed to make folder: %v", err)
		t.FailNow()
	}
	for i := 0; i < 10; i++ {
		testZipMakeFile(t, tester10, mapsFolder.ID, fmt.Sprintf("mapdata%d.txt", i), "lat=5,log=6")
	}
	mapsFolder2, err := makeFolderWithACMWithParentViaJSON("dat", mapsFolder.ID, ValidAcmCreateObjectSimple, tester10)
	testZipMakeFile(t, tester10, mapsFolder2.ID, "data.txt", "count=325")

	//Actually perform a zip request and ensure that we get something back
	uri := host + cfg.NginxRootURL + "/documents/zip?folderId=" + mapsFolder.ID
	t.Logf("trying: GET %s", uri)
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		t.Logf("unable to make request for zip: %v", err)
		t.FailNow()
	}

	//Perform zip of file
	t.Logf("Starting zip: %v", time.Now())
	res, err := client.Do(req)
	t.Logf("Stopping zip: %v", time.Now())
	if err != nil {
		t.Logf("cannot get zip: %v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)

	if res.StatusCode != http.StatusOK {
		t.Logf("wrong status code: %d", res.StatusCode)
		t.FailNow()
	}

}
