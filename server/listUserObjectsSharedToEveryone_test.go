package server_test

import (
	"strings"
	"testing"

	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

func TestListObjectsSharedToEveryone(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	tester1 := 1

	t.Logf("* Create folder as Tester01 shared to everyone")
	folder1 := makeFolderViaJSON("TestListObjectsSharedToEveryone - Everyone", tester1, t)

	t.Logf("* Create folder as Tester01 not shared to everyone")
	folder2, err := makeFolderWithACMViaJSON("TestListObjectsSharedToEveryone - Not Everyone", ValidACMUnclassifiedFOUOSharedToTester01And02, tester1)

	t.Logf("* Get list of objects shared to everyone")
	uriEveryone := mountPoint + "/sharedpublic"
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
		t.Logf("Object shared to everyone (%s) was not found in call to %s", folder1.ID, uriEveryone)
		t.Fail()
	}
	if found2 {
		t.Logf("Object not shared to everyone appeared in call to %s", uriEveryone)
		t.Fail()
	}

}

func TestListObjectsSharedToEveryoneExcludeChildren(t *testing.T) {
	// Skipping this test now since no longer excluding children for performance reasons
	t.Skip()

	if testing.Short() {
		t.Skip()
	}

	tester1 := 1 // represents Alice
	uriEveryone := mountPoint + "/sharedpublic"

	ACMtester1Private := `{"banner":"UNCLASSIFIED","classif":"U","dissem_countries":["USA"],"portion":"U","share":{"users":["cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`
	ACMtester1And2 := `{"banner":"UNCLASSIFIED","classif":"U","dissem_countries":["USA"],"portion":"U","share":{"users":["cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us","cn=test tester02,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`
	ACMeveryone := `{"banner":"UNCLASSIFIED","classif":"U","dissem_countries":["USA"],"portion":"U","version":"2.1.0"}`

	t.Logf("* Creating hierarchy of objects owned by tester1")
	a1, _ := makeFolderWithACMViaJSON("A1", ACMtester1Private, tester1)
	a2, _ := makeFolderWithACMWithParentViaJSON("A2", a1.ID, ACMtester1Private, tester1)
	a3, _ := makeFolderWithACMWithParentViaJSON("A3", a1.ID, ACMtester1And2, tester1)
	a4, _ := makeFolderWithACMWithParentViaJSON("A4", a1.ID, ACMeveryone, tester1)
	a5, _ := makeFolderWithACMViaJSON("A5", ACMtester1And2, tester1)
	a6, _ := makeFolderWithACMWithParentViaJSON("A6", a5.ID, ACMtester1Private, tester1)
	a7, _ := makeFolderWithACMWithParentViaJSON("A7", a5.ID, ACMtester1And2, tester1)
	a8, _ := makeFolderWithACMWithParentViaJSON("A8", a5.ID, ACMeveryone, tester1)
	a9, _ := makeFolderWithACMViaJSON("A9", ACMeveryone, tester1)
	a10, _ := makeFolderWithACMWithParentViaJSON("A10", a9.ID, ACMtester1Private, tester1)
	a11, _ := makeFolderWithACMWithParentViaJSON("A11", a9.ID, ACMtester1And2, tester1)
	a12, _ := makeFolderWithACMWithParentViaJSON("A12", a9.ID, ACMeveryone, tester1)
	t.Logf(`
[Tester1 Root]
   [A1] tester1 private
      [A2] tester1 private
	  [A3] tester1 and tester2
	  [A4] public
   [A5] tester1 and tester2
	  [A6] tester1 private
	  [A7] tester1 and tester2
	  [A8] public
   [A9] public
	  [A10] tester1 private
	  [A11] tester1 and tester2
	  [A12] public 
    `)

	t.Logf("* Get objects shared to everyone")
	listSharedTester1Request := makeHTTPRequestFromInterface(t, "GET", uriEveryone, nil)
	listSharedTester1Response, err := clients[tester1].Client.Do(listSharedTester1Request)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, listSharedTester1Response, "Bad status when listing objects shared to everyone")
	var sharedToEveryone protocol.ObjectResultset
	err = util.FullDecode(listSharedTester1Response.Body, &sharedToEveryone)

	t.Logf("* Verify response includes A4, A8, A9, but not A1-A3, A5-A7, A10-A12")
	tester1Expects := []string{a4.ID, a8.ID, a9.ID}
	tester1Exclude := []string{a1.ID, a2.ID, a3.ID, a5.ID, a6.ID, a7.ID, a10.ID, a11.ID, a12.ID}
	for _, o := range sharedToEveryone.Objects {
		found := false
		for _, excludeID := range tester1Exclude {
			if strings.Compare(o.ID, excludeID) == 0 {
				t.Logf("Object %s was found shared to everyone when expected to be excluded", o.Name)
				found = true
				break
			}
		}
		if found {
			t.Fail()
		}
	}
	for i, expectedID := range tester1Expects {
		found := false
		for _, o := range sharedToEveryone.Objects {
			if strings.Compare(o.ID, expectedID) == 0 {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Tester1 expected object[%d] with id %s but it was not returned in shared to everyone", i, expectedID)
			t.Fail()
		}
	}
	if t.Failed() {
		t.FailNow()
	}

}
