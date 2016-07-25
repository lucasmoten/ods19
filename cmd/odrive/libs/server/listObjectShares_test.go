package server_test

import (
	"net/http"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/util"

	"decipher.com/object-drive-server/protocol"
)

func XTestListObjectShares(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid1 := 0
	clientid2 := 1

	if verboseOutput {
		t.Logf("(Verbose Mode) Using client id %d", clientid1)
	}

	folder1 := makeFolderViaJSON("Test Folder 1 ", clientid1, t)

	// URL
	uri := host + cfg.NginxRootURL + "/object/" + folder1.ID + "/shares"
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}

	// Get shares as the creator
	res1, err := clients[clientid1].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if res1.StatusCode != http.StatusOK {
		t.Logf("Unexpected status %d for creator", res1.Status)
		t.FailNow()
	}

	// Get shares as a different user
	res2, err := clients[clientid2].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	if res2.StatusCode == http.StatusOK {
		t.Logf("Unexpected status %d for second user", res2.Status)
		t.FailNow()
	}

	// Parse first response to permissions
	var permissions []protocol.Permission
	err = util.FullDecode(res1.Body, &permissions)
	if err != nil {
		t.Logf("Error decoding json to Permission array: %v", err)
		t.FailNow()
	}
	if verboseOutput {
		t.Logf("Permission count: %d", len(permissions))
	}

}
