package server_test

import (
	"strings"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/util"

	"decipher.com/object-drive-server/protocol"
)

func TestListObjectRevisions(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	tester10 := 0

	t.Logf("Create a folder")
	originalName := "Testing Revisions - Created"
	folder1 := makeFolderViaJSON(originalName, tester10, t)
	originalName = folder1.Name

	t.Logf("Modify it, changing the name")
	changedName := "Testing Revisions - Renamed"
	updateuri := host + cfg.NginxRootURL + "/objects/" + folder1.ID + "/properties"
	folder1.Name = changedName
	updateFolderReq := makeHTTPRequestFromInterface(t, "POST", updateuri, folder1)
	updateFolderRes, err := clients[tester10].Client.Do(updateFolderReq)
	failNowOnErr(t, err, "Unable to do request")
	statusExpected(t, 200, updateFolderRes, "Bad status when renaming folder")
	var updatedFolder protocol.Object
	err = util.FullDecode(updateFolderRes.Body, &updatedFolder)
	if strings.Compare(updatedFolder.Name, changedName) != 0 {
		t.Logf(" Name is %s expected %s", updatedFolder.Name, changedName)
		t.FailNow()
	}
	changedName = updatedFolder.Name

	t.Logf("Get revisions for the folder")
	revisionsuri := host + cfg.NginxRootURL + "/revisions/" + folder1.ID
	revisionsReq := makeHTTPRequestFromInterface(t, "GET", revisionsuri, nil)
	revisionsRes, err := clients[tester10].Client.Do(revisionsReq)
	failNowOnErr(t, err, "Unable to do request")
	statusExpected(t, 200, revisionsRes, "Bad status when getting revisions")
	var listOfRevisions protocol.ObjectResultset
	err = util.FullDecode(revisionsRes.Body, &listOfRevisions)
	for _, revision := range listOfRevisions.Objects {
		switch revision.ChangeCount {
		case 0:
			if revision.Name != originalName {
				t.Logf("Name for original revision (%s) does not match expected value (%s)", revision.Name, originalName)
				t.Fail()
			}
		case 1:
			if revision.Name != changedName {
				t.Logf("Name for first revision (%s) does not match expected value (%s)", revision.Name, changedName)
				t.Fail()
			}
		default:
			t.Logf("More revisions exist then expected. There are %d records", listOfRevisions.TotalRows)
			t.Fail()
		}
	}
}
