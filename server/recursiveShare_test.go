package server_test

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/tidwall/gjson"

	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/utils"
)

func TestUpdateObjectShareRecursive(t *testing.T) {

	cases := []struct {
		// Scenario:
		// 1. creator creates an hierarchy of object with no permissions
		//    and initialACM: root/child1Obj/child2Obj/child3Obj
		// 2. sharer then updates child 1 with updatedACM and perms.
		// 3. requester tries to get child 1, expecting success
		// 4. requester tries to get child 3, expecting success

		// identities are indexes into the global clients array
		creator   int
		sharer    int
		requester int

		// Updated permissions for UpdateObject. None are passed to CreateObject.
		perms protocol.Permission

		// initialACM is given to CreateObject
		initialACM string

		// updatedACM is given to UpdateObject.
		updatedACM string
	}{
		{
			0, // creator
			0, // sharer
			2, // requester
			protocol.Permission{ // updated permissions. None are passed at first.
				Create: protocol.PermissionCapability{
					AllowedResources: []string{
						"user/" + fakeDN1 + "/test tester02",
					},
				},
				Read: protocol.PermissionCapability{
					AllowedResources: []string{
						"user/" + fakeDN1 + "/test tester02",
					},
				},
				Update: protocol.PermissionCapability{
					AllowedResources: []string{
						"user/" + fakeDN1 + "/test tester02",
					},
				},
				Delete: protocol.PermissionCapability{
					AllowedResources: []string{
						"user/" + fakeDN1 + "/test tester02",
					},
				},
				Share: protocol.PermissionCapability{
					AllowedResources: []string{
						"user/" + fakeDN1 + "/test tester02",
					},
				},
			},
			`{"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"portion":"U//FOUO","share":{"users":["cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`,
			`{"banner":"SECRET//NF","classif":"S","dissem_countries":["USA"],"dissem_ctrls":["NF"],"portion":"S//NF","share":{"users":["cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us","cn=test tester02,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us","cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`,
		},
	}

	for _, c := range cases {
		root, child1, child2, child3 := randomName("root"), randomName("child1"), randomName("child2"), randomName("child3")

		t.Logf("Create object hierarchy:\n root: %s\n child1: %s\n child2: %s\n child3: %s\n",
			root, child1, child2, child3)

		cor := protocol.CreateObjectRequest{
			NamePathDelimiter: ":::",
			Name:              strings.Join([]string{root, child1, child2, child3}, ":::"),
			RawAcm:            c.initialACM,
		}
		// child3 is returned
		child3Obj, err := clients[c.creator].C.CreateObject(cor, nil)
		if err != nil {
			t.Errorf("create failed for creator %v: %v", c.requester, err)
		}
		child3Acm, err := utils.MarshalInterfaceToString(child3Obj.RawAcm)
		if err != nil {
			t.Errorf("unable to convert acm")
		}
		child3classif := gjson.Get(child3Acm, "classif")
		if !child3classif.Exists() {
			t.Errorf("acm for child3 does not have classif")
		}
		child3fShare := gjson.Get(child3Acm, "f_share")
		if !child3fShare.Exists() {
			t.Errorf("acm for child3 does not have f_share")
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
			ID:             child1Obj.ID,
			ChangeToken:    child1Obj.ChangeToken,
			RecursiveShare: true,
			Permission:     c.perms,
			RawAcm:         c.updatedACM,
		}
		clients[c.sharer].C.Verbose = testing.Verbose()
		child1Updated, err := clients[c.sharer].C.UpdateObject(uor)
		if err != nil {
			t.Errorf("error updating object: %v", err)
			t.FailNow()
		}
		child1UpdatedAcm, err := utils.MarshalInterfaceToString(child1Updated.RawAcm)
		if err != nil {
			t.Errorf("unable to convert acm for updated child1")
			t.FailNow()
		}
		child1UpdatedfShare := gjson.Get(child1UpdatedAcm, "f_share")
		if !child1UpdatedfShare.Exists() {
			t.Error("acm for child1 updated does not have f_share")
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
			child3Updated, err := clients[c.requester].C.GetObject(child3Obj.ID)
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
			child3UpdatedAcm, err := utils.MarshalInterfaceToString(child3Updated.RawAcm)
			if err != nil {
				t.Errorf("unable to convert acm for updated child 3")
				t.FailNow()
			}
			child3Updatedclassif := gjson.Get(child3UpdatedAcm, "classif")
			if !child3Updatedclassif.Exists() {
				t.Error("acm for child3 updated does not have classif")
				t.FailNow()
			}
			child3UpdatedfShare := gjson.Get(child3UpdatedAcm, "f_share")
			if !child3UpdatedfShare.Exists() {
				t.Error("acm for child3 updated does not have f_share")
				t.FailNow()
			}
			if child3classif.Str != child3Updatedclassif.Str {
				t.Errorf("classification for child3 changed from %s to %s", child3classif, child3Updatedclassif)
				t.FailNow()
			}
			if child3UpdatedfShare == child3fShare {
				t.Error("expected f_share for child3 to change")
				t.FailNow()
			}
			// verify all f_share in child3 are found in child1
			child3UpdatedfShare.ForEach(func(key, value gjson.Result) bool {
				child3fShareVal := value.String()
				found := false
				child1UpdatedfShare.ForEach(func(key, value gjson.Result) bool {
					child1fShareVal := value.String()
					if child1fShareVal == child3fShareVal {
						found = true
						return false
					}
					return true
				})
				if !found {
					t.Errorf("expected f_share value %s from child3 to be found in child1 but it was not", child3fShareVal)
					t.FailNow()
				}
				return true
			})
			// verify all f_share in child1 are found in child3
			child1UpdatedfShare.ForEach(func(key, value gjson.Result) bool {
				child1fShareVal := value.String()
				found := false
				child3UpdatedfShare.ForEach(func(key, value gjson.Result) bool {
					child3fShareVal := value.String()
					if child3fShareVal == child1fShareVal {
						found = true
						return false
					}
					return true
				})
				if !found {
					t.Errorf("expected f_share value %s from child1 to be found in child3 but it was not", child1fShareVal)
					t.FailNow()
				}
				return true
			})
			break
		}
	}
}

func TestUpdateObjectAndStreamShareRecursive(t *testing.T) {

	cases := []struct {
		// Scenario:
		// 1. creator creates an hierarchy of object with no permissions
		//    and initialACM: root/child1Obj/child2Obj/child3Obj
		//    Note that child3Obj is created with a filestream.
		// 2. sharer then updates child 1 with updatedACM and perms.
		// 3. requester tries to get child 1's object stream, expecting success
		// 4. requester tries to get child 3's object stream, expecting success

		// identies are indexes into the global clients array
		creator   int
		sharer    int
		requester int

		// Updated permissions for UpdateObjectAndStream. None are passed to CreateObject.
		perms protocol.Permission

		// streamContents is copied into an io.Reader for UpdateObjectAndStream
		streamContents []byte

		// initialACM is given to UpdateObjectAndStream.
		initialACM string

		// updatedACM is given to UpdateObjectAndStream.
		updatedACM string
	}{
		{
			0, // creator
			0, // sharer
			1, // requester
			protocol.Permission{ // updated permissions. None are passed at first.
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
			[]byte("The Biologist"),
			ValidACMUnclassifiedFOUOSharedToTester10,      // initial acm
			ValidACMUnclassifiedFOUOSharedToTester01And02, // updated acm
		},
	}

	for _, c := range cases {
		root, child1, child2, child3 := randomName("root"), randomName("child1"), randomName("child2"), randomName("child3")

		t.Logf("Create object hierarchy:\n root: %s\n child1: %s\n child2: %s\n child3: %s\n",
			root, child1, child2, child3)

		cor := protocol.CreateObjectRequest{
			NamePathDelimiter: ":::",
			Name:              strings.Join([]string{root, child1, child2, child3}, ":::"),
			RawAcm:            c.initialACM,
		}
		// child3 is returned
		child3Obj, err := clients[c.creator].C.CreateObject(cor, bytes.NewBuffer(c.streamContents))
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

		uor := protocol.UpdateObjectAndStreamRequest{
			ID: child1Obj.ID,
			// NOTE(cm): If we do not supply a name here, we get
			// "status":400,"message":"file must be supplied as multipart mime part"
			// uploadDownload.go line 186
			// Is this a bug where we are not preserving the original name?
			Name:           "should_i_need_a_name",
			ChangeToken:    child1Obj.ChangeToken,
			RecursiveShare: true,
			Permission:     c.perms,
			RawAcm:         c.updatedACM,
		}
		clients[c.sharer].C.Verbose = testing.Verbose()
		_, err = clients[c.sharer].C.UpdateObjectAndStream(uor, bytes.NewBuffer(c.streamContents))
		if err != nil {
			t.Errorf("error updating object: %v", err)
			t.FailNow()
		}

		// can requester get child 1 stream?
		r, err := clients[c.requester].C.GetObjectStream(child1Obj.ID)
		if err != nil {
			t.Errorf("requester could not get child1: %v", err)
			t.FailNow()
		}
		contents, _ := ioutil.ReadAll(r)
		if bytes.Compare(contents, c.streamContents) != 0 {
			t.Errorf("expected stream contents to be: %s\nbut got: %s",
				string(c.streamContents), string(contents))
		}

		// can requester get object child 3?
		tries := 0
		for {
			// We must retry because this is an async operation on the server.
			r, err := clients[c.requester].C.GetObjectStream(child3Obj.ID)
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
			contents, _ = ioutil.ReadAll(r)
			if bytes.Compare(contents, c.streamContents) != 0 {
				t.Errorf("expected stream contents to be: %s\nbut got: %s",
					string(c.streamContents), string(contents))
			}
			break
		}
	}
}

func randomName(name string) string {
	s, _ := util.NewGUID()
	return name + s
}
