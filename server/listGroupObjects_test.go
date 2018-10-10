package server_test

import (
	"log"
	"net/http"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"

	"bitbucket.di2e.net/dime/object-drive-server/protocol"
)

func TestListGroupObjects(t *testing.T) {
	clientid := 0
	groupName := models.AACFlatten(fakeDN0)

	// URL
	uri := mountPoint + "/groupobjects/" + groupName
	uri1 := uri + "?PageNumber=1&PageSize=2"

	// Request
	req, err := http.NewRequest("GET", uri1, nil)
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName:       "List Root Objects for a Group",
			RequestDescription:  "Send a response for a paged listing",
			ResponseDescription: "We get back a page from the listing",
			ResponseBodyHide:    true, //this might be kind of big due to test re-runs
		},
	)

	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	var listOfObjects protocol.ObjectResultset
	err = util.FullDecode(res.Body, &listOfObjects)
	if err != nil {
		log.Printf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}
}

func TestListGroupObjectsCounts(t *testing.T) {
	clientid := 0
	groupName := "dctc_odrive_g1"
	groupResourceName := "group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1"
	// URL
	uri := mountPoint + "/groupobjects/" + groupName
	uri1 := uri + "?PageNumber=1&PageSize=2"

	t.Logf("* List objects owned by group %s", groupName)
	req, err := http.NewRequest("GET", uri1, nil)
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := clients[clientid].Client.Do(req)
	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(res.Body)
	// Response validation
	statusMustBe(t, 200, res, "Bad status when listing objects in group")
	var listOfObjects protocol.ObjectResultset
	err = util.FullDecode(res.Body, &listOfObjects)
	if err != nil {
		log.Printf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}
	totalBeforeAddingObject := listOfObjects.TotalRows
	t.Logf("  total objects: %d", totalBeforeAddingObject)

	t.Logf("* Adding a new object")
	folder1 := makeFolderViaJSON("new folder", clientid, t)

	t.Logf("* Changing ownership to the group")
	changeowneruri := mountPoint + "/objects/" + folder1.ID + "/owner/" + groupResourceName
	objChangeToken := protocol.ChangeTokenStruct{ChangeToken: folder1.ChangeToken}
	changeOwnerRequest := makeHTTPRequestFromInterface(t, "POST", changeowneruri, objChangeToken)
	changeOwnerResponse, err := clients[clientid].Client.Do(changeOwnerRequest)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, changeOwnerResponse, "Bad status when changing owner")
	var updatedObject protocol.Object
	err = util.FullDecode(changeOwnerResponse.Body, &updatedObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* List objects owned by group %s", groupName)
	grouplistReq2 := makeHTTPRequestFromInterface(t, "GET", uri1, nil)
	grouplistRes2, err := clients[clientid].Client.Do(grouplistReq2)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, grouplistRes2, "Bad status when listing objects in group")
	var newListOfObjects protocol.ObjectResultset
	err = util.FullDecode(grouplistRes2.Body, &newListOfObjects)
	if err != nil {
		log.Printf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}
	totalAfterAddingObject := newListOfObjects.TotalRows
	t.Logf("  total objects: %d", totalAfterAddingObject)

	if totalAfterAddingObject != totalBeforeAddingObject+1 {
		t.Logf("Expected number of objects owned by the group to increase. Before %d after %d", totalBeforeAddingObject, totalAfterAddingObject)
		t.FailNow()
	}
}
