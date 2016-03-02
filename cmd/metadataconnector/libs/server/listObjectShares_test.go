package server_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"decipher.com/oduploader/protocol"
)

func TestListObjectShares(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid1 := 0
	clientid2 := 1

	if verboseOutput {
		t.Logf("(Verbose Mode) Using client id %d", clientid1)
	}

	folder1, err := makeFolderViaJSON("Test Folder 1 "+strconv.FormatInt(time.Now().Unix(), 10), clientid1)
	if err != nil {
		t.Logf("Error making folder 1: %v", err)
		t.FailNow()
	}

	// URL
	uri := host + "/service/metadataconnector/1.0/object/" + folder1.ID + "/shares"
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}

	// Get shares as the creator
	res1, err := httpclients[clientid1].Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if res1.StatusCode != http.StatusOK {
		t.Logf("Unexpected status %d for creator", res1.Status)
		t.FailNow()
	}

	// Get shares as a different user
	res2, err := httpclients[clientid2].Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if res2.StatusCode == http.StatusOK {
		t.Logf("Unexpected status %d for second user", res2.Status)
		t.FailNow()
	}

	// Parse first response to permissions
	decoder := json.NewDecoder(res1.Body)
	var permissions []protocol.Permission
	err = decoder.Decode(&permissions)
	if err != nil {
		t.Logf("Error decoding json to Permission array: %v", err)
		t.FailNow()
	}
	if verboseOutput {
		t.Logf("Permission count: %d", len(permissions))
	}

}
