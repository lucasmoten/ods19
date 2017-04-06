package server_test

import (
	"net/http"
	"strings"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
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
		testhelpers.ValidACMTopSecretSITK)
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
		testhelpers.ValidACMUnclassifiedFOUO)
	t.Logf("Check the access from a user with lower clearance")
	unclearedID := 1
	shouldHaveReadForObjectID(t, updated.ID, unclearedID)
	t.Logf("Lower cleared version can see unclassified version")
	expectingReadForObjectIDVersion(t, http.StatusOK, 1, updated.ID, unclearedID)
	t.Logf("Lower cleared user should not be able to see highly classified version")
	expectingReadForObjectIDVersion(t, http.StatusForbidden, 0, updated.ID, unclearedID)

	t.Logf("Lower cleared user lists versions")
	uri := host + cfg.NginxRootURL + "/revisions/" + updated.ID
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
