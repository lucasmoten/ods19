package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/deciphernow/object-drive-server/protocol"
	"github.com/deciphernow/object-drive-server/server"
	"github.com/deciphernow/object-drive-server/util"
)

func TestListMyGroupsWithObjects(t *testing.T) {

	// as tester10 ...
	tester10 := 0

	t.Logf("Check existing groups, and for each, add an object at root owned by that group, then call again verifying an increase")
	groupspaces := getUserGroups(t, tester10)
	if groupspaces.TotalRows > 0 {
		t.Logf("/groups returned existing groups")
		for _, groupspace := range groupspaces.GroupSpaces {
			t.Logf("Adding an object for %s which currently has %d objects", groupspace.ResourceString, groupspace.Quantity)
			makeObjectOwnedByGroup(t, tester10, groupspace.ResourceString)
			groupspaces2 := getUserGroups(t, tester10)
			granteefound := false
			for _, groupspace2 := range groupspaces2.GroupSpaces {
				if groupspace2.Grantee == groupspace.Grantee {
					granteefound = true
					if groupspace2.Quantity <= groupspace.Quantity {
						t.Logf("Added new root object owned by %s but quantity did not increase for group", groupspace.ResourceString)
						t.Failed()
					} else {
						t.Logf("Now it has %d objects", groupspace2.Quantity)
					}
				}
			}
			if !granteefound {
				t.Logf("Added new root object owned by %s but now that grantee not returned in /groups", groupspace.ResourceString)
				t.Failed()
			}
		}
	}

	t.Logf("Explicitly create objects in root for groups we expect to be a member of")
	makeObjectOwnedByGroup(t, tester10, "group/dctc/DCTC/ODrive")
	makeObjectOwnedByGroup(t, tester10, "group/dctc/DCTC/ODrive_G1")

	t.Logf("verify that we get expected groups for those we're a member of")
	groupspaces3 := getUserGroups(t, tester10)
	found_dctc_odrive := false
	found_dctc_odrive_g1 := false
	for _, groupspace3 := range groupspaces3.GroupSpaces {
		if groupspace3.Grantee == "dctc_odrive" {
			t.Logf("Found %s with %d in root", groupspace3.Grantee, groupspace3.Quantity)
			found_dctc_odrive = true
		}
		if groupspace3.Grantee == "dctc_odrive_g1" {
			t.Logf("Found %s with %d in root", groupspace3.Grantee, groupspace3.Quantity)
			found_dctc_odrive_g1 = true
		}
	}
	if !found_dctc_odrive {
		t.Logf("dctc_odrive was not returned in groups")
		t.Failed()
	}
	if !found_dctc_odrive_g1 {
		t.Logf("dctc_odrive_g1 was not returned in groups")
		t.Failed()
	}
}

func makeObjectOwnedByGroup(t *testing.T, clientid int, ownedby string) {
	objuri := mountPoint + "/objects"
	obj := protocol.Object{}
	obj.Name = "TestListMyGroupsWithObjects " + strconv.FormatInt(time.Now().Unix(), 10)
	obj.TypeName = "Folder"
	obj.OwnedBy = ownedby
	obj.RawAcm = server.ValidACMUnclassified
	jsonBody, err := json.Marshal(obj)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("POST", objuri, bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := clients[clientid].Client.Do(req)
	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(res.Body)
	statusMustBe(t, 200, res, fmt.Sprintf("client id %d could not create object %s", clientid, obj.Name))
}
func getUserGroups(t *testing.T, clientid int) *protocol.GroupSpaceResultset {
	groupsuri := mountPoint + "/groups"
	req, err := http.NewRequest("GET", groupsuri, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
		return nil
	}
	res, err := clients[clientid].Client.Do(req)
	failNowOnErr(t, err, "Unable to do request")
	defer util.FinishBody(res.Body)
	statusMustBe(t, 200, res, fmt.Sprintf("client id %d could not get groups", clientid))
	var groupspaces protocol.GroupSpaceResultset
	err = util.FullDecode(res.Body, &groupspaces)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		return nil
	}
	return &groupspaces
}
