package server_test

import (
	"strings"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestListObjectsSharedToOthers(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	tester1 := 1

	t.Logf("* Create folder1 as Tester01 shared to everyone")
	folder1 := makeFolderViaJSON("TestListObjectsSharedToOthers - Everyone", tester1, t)

	t.Logf("* Create folder2 as Tester01 that is private to Tester01")
	folder2, err := makeFolderWithACMViaJSON("TestListObjectsSharedToOthers - Tester01", testhelpers.ValidACMUnclassifiedFOUOSharedToTester01, tester1)

	t.Logf("* Create folder3 as Tester01 that is shared to Tester01 and Tester02")
	folder3, err := makeFolderWithACMViaJSON("TestListObjectsSharedToOthers - Tester01, Tester02", testhelpers.ValidACMUnclassifiedFOUOSharedToTester01And02, tester1)

	t.Logf("* Get list of objects shared by tester01")
	uriShared := host + cfg.NginxRootURL + "/shared"
	listReq1 := makeHTTPRequestFromInterface(t, "GET", uriShared, nil)
	listRes1, err := clients[tester1].Client.Do(listReq1)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, listRes1, "Bad status when listing objects shared by tester01")
	var resultset protocol.ObjectResultset
	err = util.FullDecode(listRes1.Body, &resultset)

	t.Logf("* Iterate objects in resultset, looking for folder1 and folder3, but not folder2")
	found1 := false
	found2 := false
	found3 := false
	for _, obj := range resultset.Objects {
		if strings.Compare(obj.ID, folder1.ID) == 0 {
			found1 = true
		}
		if strings.Compare(obj.ID, folder2.ID) == 0 {
			found2 = true
		}
		if strings.Compare(obj.ID, folder3.ID) == 0 {
			found3 = true
		}
	}
	if !found1 {
		t.Logf("Object shared to everyone (%s) was not found in call to %s", folder1.ID, uriShared)
		t.Fail()
	}
	if found2 {
		t.Logf("Object that is private to Tester01 (%s) was reported as shared in call to %s", folder2.ID, uriShared)
		t.Fail()
	}
	if !found3 {
		t.Logf("Object shared to tester02 (%s) was not found in call to %s", folder3.ID, uriShared)
		t.Fail()
	}

}
