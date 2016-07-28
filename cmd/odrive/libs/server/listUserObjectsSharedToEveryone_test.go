package server_test

import (
	"strings"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestListObjectsSharedToEveryone(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	tester1 := 1

	t.Logf("* Create folder as Tester01 shared to everyone")
	folder1 := makeFolderViaJSON("TestListObjectsSharedToEveryone - Everyone", tester1, t)

	t.Logf("* Create folder as Tester01 not shared to everyone")
	folder2, err := makeFolderWithACMViaJSON("TestListObjectsSharedToEveryone - Not Everyone", testhelpers.ValidACMUnclassifiedFOUOSharedToTester01And02, tester1)

	t.Logf("* Get list of objects shared to everyone")
	uriEveryone := host + cfg.NginxRootURL + "/sharedpublic"
	listReq1 := makeHTTPRequestFromInterface(t, "GET", uriEveryone, nil)
	listRes1, err := clients[tester1].Client.Do(listReq1)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, listRes1, "Bad status when listing objects shared to everyone")
	var resultset protocol.ObjectResultset
	err = util.FullDecode(listRes1.Body, &resultset)

	t.Logf("* Iterate objects in resultset, looking for folder1 and folder2")
	found1 := false
	found2 := false
	for _, obj := range resultset.Objects {
		if strings.Compare(obj.ID, folder1.ID) == 0 {
			found1 = true
		}
		if strings.Compare(obj.ID, folder2.ID) == 0 {
			found2 = true
		}
	}
	if !found1 {
		t.Logf("Object shared to everyone was not found in call to %s", uriEveryone)
		t.Fail()
	}
	if found2 {
		t.Logf("Object not shared to everyone appeared in call to %s", uriEveryone)
		t.Fail()
	}

}
