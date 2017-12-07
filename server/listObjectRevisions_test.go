package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/deciphernow/object-drive-server/protocol"
	"github.com/deciphernow/object-drive-server/util"
)

func TestUpdateObjectWithClassificationDrop(t *testing.T) {
	t.Logf("Create a classified document with a highly cleared user")
	clientID := 5
	data := "Area 51 Gray Alien Xenu says: It's time!"
	_, created := doTestCreateObjectSimple(t, data, clientID,
		trafficLogs[APISampleFile],
		&TrafficLogDescription{
			OperationName:       "Create Classified File About Grey Aliens",
			RequestDescription:  "Generate a file with high classification",
			ResponseDescription: "Get an object to update",
		},
		ValidACMTopSecretSITK)
	t.Logf("Verifying newly created file exists")
	doCheckFileNowExists(t, clientID, created)
	t.Logf("Update with a lower classification")
	data = "**** ** **** ***** **** says: It's time!"
	_, updated := doTestUpdateObjectSimple(t, data, clientID,
		created,
		trafficLogs[APISampleFile],
		&TrafficLogDescription{
			OperationName:       "Declassify File About Grey Aliens",
			RequestDescription:  "Lower the classification request",
			ResponseDescription: "The redacted file",
		},
		ValidACMUnclassifiedFOUO)
	t.Logf("Check the access from a user with lower clearance")
	unclearedID := 1
	shouldHaveReadForObjectID(t, updated.ID, unclearedID)
	t.Logf("Lower cleared version can see unclassified version")
	expectingReadForObjectIDVersion(t, http.StatusOK, 1, updated.ID, unclearedID)
	t.Logf("Lower cleared user should not be able to see highly classified version")
	expectingReadForObjectIDVersion(t, http.StatusForbidden, 0, updated.ID, unclearedID)

	t.Logf("Lower cleared user lists versions")
	uri := mountPoint + "/revisions/" + updated.ID
	req := makeHTTPRequestFromInterface(t, "GET", uri, nil)
	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName:       "Show Revisions on Declassified File About Grey Aliens",
			RequestDescription:  "Ask for revisions",
			ResponseDescription: "Show redacted listing (should not have secret information)",
		},
	)
	res, err := clients[unclearedID].Client.Do(req)
	failNowOnErr(t, err, "Unable to do request")
	statusExpected(t, 200, res, "Bad status when getting revisions")
	trafficLogs[APISampleFile].Response(t, res)
	var listOfRevisions protocol.ObjectResultset
	err = util.FullDecode(res.Body, &listOfRevisions)
	if err != nil {
		t.Logf("Unable to decode version listing: %v", err)
		t.FailNow()
	}
	visibleCount := 0
	redactedCount := 0
	for _, v := range listOfRevisions.Objects {
		acmMap, ok := v.RawAcm.(map[string]interface{})
		if ok {
			visibleCount += 1
			banner, ok := acmMap["banner"].(string)
			if ok {
				if strings.HasPrefix(banner, "TOP SECRET") {
					t.Logf("We got something we don't have permission for")
					t.FailNow()
				}
			}
		} else {
			redactedCount += 1
		}
	}
	t.Logf("Visible: %d, Redacted: %d", visibleCount, redactedCount)
}

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
	updateuri := mountPoint + "/objects/" + folder1.ID + "/properties"
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
	revisionsuri := mountPoint + "/revisions/" + folder1.ID
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

func TestListObjectRevisionsWithProperties(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	tester10 := 0

	t.Logf("Create a folder")
	originalName := "Testing Revisions - Created"
	folder1 := makeFolderViaJSON(originalName, tester10, t)
	originalName = folder1.Name
	t.Logf("Folder ID = %s", folder1.ID)

	t.Logf("Modify it, adding a property")
	updateuri := mountPoint + "/objects/" + folder1.ID + "/properties"
	folder1.Properties = append(folder1.Properties, protocol.Property{Name: "property1", Value: "originalvalue1"})
	updateFolderReq := makeHTTPRequestFromInterface(t, "POST", updateuri, folder1)
	updateFolderRes, err := clients[tester10].Client.Do(updateFolderReq)
	failNowOnErr(t, err, "Unable to do request")
	statusExpected(t, 200, updateFolderRes, "Bad status when adding property to folder")
	var updatedFolder protocol.Object
	err = util.FullDecode(updateFolderRes.Body, &updatedFolder)
	if len(updatedFolder.Properties) != 1 {
		t.Logf(" Property was not added")
		t.FailNow()
	}
	if updatedFolder.Properties[0].Name != "property1" || updatedFolder.Properties[0].Value != "originalvalue1" {
		t.Logf(" Property[0] is not expected. %s=%s", updatedFolder.Properties[0].Name, updatedFolder.Properties[0].Value)
		t.FailNow()
	}

	t.Logf("Modify again, adding another property")
	updatedFolder.Properties = append(updatedFolder.Properties, protocol.Property{Name: "property2", Value: "originalvalue2"})
	updateFolderReq2 := makeHTTPRequestFromInterface(t, "POST", updateuri, updatedFolder)
	updateFolderRes2, err := clients[tester10].Client.Do(updateFolderReq2)
	failNowOnErr(t, err, "Unable to do request")
	statusExpected(t, 200, updateFolderRes, "Bad status when adding property to folder")
	var updatedFolder2 protocol.Object
	err = util.FullDecode(updateFolderRes2.Body, &updatedFolder2)
	if len(updatedFolder2.Properties) != 2 {
		t.Logf(" Property was not added")
		t.FailNow()
	}
	if updatedFolder2.Properties[0].Name != "property1" || updatedFolder2.Properties[0].Value != "originalvalue1" {
		t.Logf(" Property[0] is not expected (property1=originalvalue1). actual(%s=%s)", updatedFolder2.Properties[0].Name, updatedFolder2.Properties[0].Value)
		t.FailNow()
	}
	if updatedFolder2.Properties[1].Name != "property2" || updatedFolder2.Properties[1].Value != "originalvalue2" {
		t.Logf(" Property[1] is not expected (property2=originalvalue2). actual (%s=%s)", updatedFolder2.Properties[1].Name, updatedFolder2.Properties[1].Value)
		t.FailNow()
	}

	t.Logf("Modify again, changing value of first property")
	updatedFolder2.Properties[0].Value = "changedvalue"
	updateFolderReq3 := makeHTTPRequestFromInterface(t, "POST", updateuri, updatedFolder2)
	updateFolderRes3, err := clients[tester10].Client.Do(updateFolderReq3)
	failNowOnErr(t, err, "Unable to do request")
	statusExpected(t, 200, updateFolderRes, "Bad status when changing value of existing property")
	var updatedFolder3 protocol.Object
	err = util.FullDecode(updateFolderRes3.Body, &updatedFolder3)
	if len(updatedFolder3.Properties) != 2 {
		t.Logf(" Property count is incorrect")
		t.FailNow()
	}
	if updatedFolder3.Properties[0].Name != "property1" || updatedFolder3.Properties[0].Value != "changedvalue" {
		t.Logf(" Property[0] is not expected (property1=changedvalue). actual(%s=%s)", updatedFolder3.Properties[0].Name, updatedFolder3.Properties[0].Value)
		t.Fail()
	}
	if updatedFolder3.Properties[1].Name != "property2" || updatedFolder3.Properties[1].Value != "originalvalue2" {
		t.Logf(" Property[1] is not expected (property2=originalvalue2). actual (%s=%s)", updatedFolder3.Properties[1].Name, updatedFolder3.Properties[1].Value)
		t.Fail()
	}

	t.Logf("Get revisions for the folder")
	revisionsuri := mountPoint + "/revisions/" + folder1.ID
	revisionsReq := makeHTTPRequestFromInterface(t, "GET", revisionsuri, nil)
	revisionsRes, err := clients[tester10].Client.Do(revisionsReq)
	failNowOnErr(t, err, "Unable to do request")
	statusExpected(t, 200, revisionsRes, "Bad status when getting revisions")
	var listOfRevisions protocol.ObjectResultset
	err = util.FullDecode(revisionsRes.Body, &listOfRevisions)
	for _, revision := range listOfRevisions.Objects {
		switch revision.ChangeCount {
		case 0:
			if len(revision.Properties) != 0 {
				t.Logf("Original revision has %d properties", len(revision.Properties))
				t.Fail()
			}
		case 1:
			if len(revision.Properties) != 1 {
				t.Logf("1st revision has %d properties", len(revision.Properties))
				t.FailNow()
			}
			if revision.Properties[0].Name != "property1" || revision.Properties[0].Value != "originalvalue1" {
				t.Logf("1st revision property is %s=%s", revision.Properties[0].Name, revision.Properties[0].Value)
				t.Fail()
			}
		case 2:
			if len(revision.Properties) != 2 {
				t.Logf("2st revision has %d properties", len(revision.Properties))
				t.FailNow()
			}
			if revision.Properties[0].Name != "property1" || revision.Properties[0].Value != "originalvalue1" {
				t.Logf("2st revision 1st property is %s=%s", revision.Properties[0].Name, revision.Properties[1].Value)
				t.Fail()
			}
			if revision.Properties[1].Name != "property2" || revision.Properties[1].Value != "originalvalue2" {
				t.Logf("2st revision 2nd property is %s=%s", revision.Properties[1].Name, revision.Properties[1].Value)
				t.Fail()
			}
		case 3:
			if len(revision.Properties) != 2 {
				t.Logf("3rd revision has %d properties", len(revision.Properties))
				t.FailNow()
			}
			if revision.Properties[0].Name != "property1" || revision.Properties[0].Value != "changedvalue" {
				t.Logf("3rd revision 1st property is %s=%s", revision.Properties[0].Name, revision.Properties[1].Value)
				t.Fail()
			}
			if revision.Properties[1].Name != "property2" || revision.Properties[1].Value != "originalvalue2" {
				t.Logf("3rd revision 2nd property is %s=%s", revision.Properties[1].Name, revision.Properties[1].Value)
				t.Fail()
			}
		default:
			t.Logf("More revisions exist then expected. There are %d records", listOfRevisions.TotalRows)
			t.Fail()
		}
	}
}
