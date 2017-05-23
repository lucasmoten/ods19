package server_test

import (
	"strings"
	"testing"
	"time"

	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestShareRecursive(t *testing.T) {

	/* resource strings for sharing
	fakeDN0 = `cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`
	fakeDN1 = `cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`
	fakeDN2 = `cn=test tester02,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`
	newOwner := "user/" + fakeDN1
	*/

	cases := []struct {
		// creator and requester are indexes into global clients array
		creator   int
		sharer    int
		requester int

		// object properties
		classif string
		perms   protocol.Permission
		// shareTo changes?
		shareTo string
		// TODO(cm) abstract ACM up here instead of hardcoding below.
	}{
		{
			0, // creator
			0, // sharer
			1, // requester
			"U",
			protocol.Permission{
				Create: protocol.PermissionCapability{
					AllowedResources: []string{
						"user/" + fakeDN1 + "/test tester01",
					},
				},
				Read: protocol.PermissionCapability{
					AllowedResources: []string{
						"user/" + fakeDN1 + "/test tester01",
					},
				},
				Update: protocol.PermissionCapability{
					AllowedResources: []string{
						"user/" + fakeDN1 + "/test tester01",
					},
				},
				Delete: protocol.PermissionCapability{
					AllowedResources: []string{
						"user/" + fakeDN1 + "/test tester01",
					},
				},
				Share: protocol.PermissionCapability{
					AllowedResources: []string{
						"user/" + fakeDN1 + "/test tester01",
					},
				},
			},
			"",
		},
	}

	// quick func to make random names
	randomName := func(name string) string {
		s, _ := util.NewGUID()
		return name + s
	}

	for _, c := range cases {
		root, child1, child2, child3 := randomName("root"), randomName("child1"), randomName("child2"), randomName("child3")

		t.Logf("Create object hierarchy:\n root: %s\n child1: %s\n child2: %s\n child3: %s\n",
			root, child1, child2, child3)

		cor := protocol.CreateObjectRequest{
			NamePathDelimiter: ":::",
			Name:              strings.Join([]string{root, child1, child2, child3}, ":::"),
			// RawAcm:            testhelpers.ValidACMUnclassifiedFOUOSharedToTester01And02,
			RawAcm: testhelpers.ValidACMUnclassifiedFOUOSharedToTester10,
		}
		// child3 is returned
		child3Obj, err := clients[c.creator].C.CreateObject(cor, nil)
		if err != nil {
			t.Errorf("create failed for creator %v: %v", c.requester, err)
		}
		// get the parentid of the parentid
		child2Obj, err := clients[c.creator].C.GetObject(child3Obj.ParentID)
		if err != nil {
			t.Errorf("creator could not get child2: %v", err)
		}
		child1Obj, err := clients[c.creator].C.GetObject(child2Obj.ParentID)
		if err != nil {
			t.Errorf("expected creator to have access to child1: %v", err)
		}

		uor := protocol.UpdateObjectRequest{
			ID:            child1Obj.ID,
			ChangeToken:   child1Obj.ChangeToken,
			RecusiveShare: true,
			Permission:    c.perms,
			// We HAVE to use an ACM? See issue 775
			RawAcm: testhelpers.ValidACMUnclassifiedFOUOSharedToTester01And02,
			// RawAcm: testhelpers.ValidACMUnclassifiedFOUOSharedToTester10,
		}
		clients[c.sharer].C.Verbose = testing.Verbose()
		_, err = clients[c.sharer].C.UpdateObject(uor)
		if err != nil {
			t.Errorf("error updating object: %v", err)
			t.FailNow()
		}

		// can requester get child 1?
		_, err = clients[c.requester].C.GetObject(child1Obj.ID)
		if err != nil {
			t.Errorf("requester could not get child1: %v", err)
			t.FailNow()
		}

		// can requester get object child 3?
		tries := 0
		for {
			// We must retry because this is an async operation on the server.
			_, err := clients[c.requester].C.GetObject(child3Obj.ID)
			if err != nil {
				if tries < 50 {
					tries++
					t.Logf("Sleeping 50 ms. Tries %v", tries)
					time.Sleep(50 * time.Millisecond)
					continue
				}
				t.Errorf("GetObject should succeed for tester1 on child3: %v", err)
				t.FailNow()
			}
			break
		}

	}

}
