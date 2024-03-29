package server_test

import (
	"strings"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

func TestListObjectsSharedToMe(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	excludingChildren := false
	tester1 := 1 // represents Alice
	tester2 := 2 // represents Bob
	uriShares := mountPoint + "/shares"

	ACMtester1Private := `{"banner":"UNCLASSIFIED","classif":"U","dissem_countries":["USA"],"portion":"U","share":{"users":["cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`
	ACMtester2Private := `{"banner":"UNCLASSIFIED","classif":"U","dissem_countries":["USA"],"portion":"U","share":{"users":["cn=test tester02,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`
	ACMtester1And2 := `{"banner":"UNCLASSIFIED","classif":"U","dissem_countries":["USA"],"portion":"U","share":{"users":["cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us","cn=test tester02,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`
	ACMeveryone := `{"banner":"UNCLASSIFIED","classif":"U","dissem_countries":["USA"],"portion":"U","version":"2.1.0"}`

	t.Logf("* Creating hierarchy of objects owned by tester1")
	a1, err := makeFolderWithACMViaJSON("A1", ACMtester1Private, tester1)
	if err != nil {
		t.Errorf("A1 fail: %v", err)
		t.FailNow()
	}
	a2, err := makeFolderWithACMWithParentViaJSON("A2", a1.ID, ACMtester1Private, tester1)
	if err != nil {
		t.Errorf("A2 fail: %v", err)
		t.FailNow()
	}
	a3, err := makeFolderWithACMWithParentViaJSON("A3", a1.ID, ACMtester1And2, tester1)
	if err != nil {
		t.Errorf("A3 fail: %v", err)
		t.FailNow()
	}
	a4, err := makeFolderWithACMWithParentViaJSON("A4", a3.ID, ACMeveryone, tester1)
	if err != nil {
		t.Errorf("A4 fail: %v", err)
		t.FailNow()
	}
	a5, err := makeFolderWithACMWithParentViaJSON("A5", a3.ID, ACMtester1And2, tester1)
	if err != nil {
		t.Errorf("A5 fail: %v", err)
		t.FailNow()
	}
	t.Logf(`
[Tester1 Root]
   [A1] tester1 private
      [A2] tester1 private
      [A3] tester1 and tester2
         [A4] public
         [A5] tester1 and tester2    
    `)

	t.Logf("* Adding CRU privileges to tester2 for A4") // needed for B5 to be created as a child
	a4Share := protocol.ObjectShare{AllowCreate: true, AllowRead: true, AllowUpdate: true, Share: makeUserShare(fakeDN2)}
	a4b := doAddObjectShare(t, a4, &a4Share, tester1)
	a4 = a4b

	t.Logf("* Creating hierarchy of objects owned by tester2")
	b1, err := makeFolderWithACMViaJSON("B1", ACMtester2Private, tester2)
	if err != nil {
		t.Errorf("B1 fail: %v", err)
		t.FailNow()
	}
	b2, err := makeFolderWithACMWithParentViaJSON("B2", b1.ID, ACMeveryone, tester2)
	if err != nil {
		t.Errorf("B2 fail: %v", err)
		t.FailNow()
	}
	b3, err := makeFolderWithACMWithParentViaJSON("B3", b1.ID, ACMtester1And2, tester2)
	if err != nil {
		t.Errorf("B3 fail: %v", err)
		t.FailNow()
	}
	b4, err := makeFolderWithACMWithParentViaJSON("B4", b1.ID, ACMtester1And2, tester2)
	if err != nil {
		t.Errorf("B4 fail: %v", err)
		t.FailNow()
	}
	b5, err := makeFolderWithACMWithParentViaJSON("B5", a4.ID, ACMtester1And2, tester2)
	if err != nil {
		t.Errorf("B5 fail: %v", err)
		t.FailNow()
	}
	b6, err := makeFolderWithACMWithParentViaJSON("B6", b4.ID, ACMtester1And2, tester2)
	if err != nil {
		t.Errorf("B6 fail: %v", err)
		t.FailNow()
	}
	b7, err := makeFolderWithACMWithParentViaJSON("B7", b6.ID, ACMtester2Private, tester2)
	if err != nil {
		t.Errorf("B7 fail: %v", err)
		t.FailNow()
	}
	b8, err := makeFolderWithACMWithParentViaJSON("B8", b7.ID, ACMtester1And2, tester2)
	if err != nil {
		t.Errorf("B8 fail: %v", err)
		t.FailNow()
	}
	b9, err := makeFolderWithACMWithParentViaJSON("B9", b6.ID, ACMtester1And2, tester2)
	if err != nil {
		t.Errorf("B9 fail: %v", err)
		t.FailNow()
	}
	t.Logf(`
[Tester2 Root]                                  [B5] created under [A4] owned by tester1
   [B1] tester2 private
      [B2] public                               [Tester1 Root]
      [B3] tester1 and tester2                     [A1] tester1 private
      [B4] tester1 and tester2                        [A2] tester1 private
         [B6] tester1 and tester2                     [A3] tester1 and tester2
            [B7] tester2 private                         [A4] public    
               [B8] tester1 and tester2                     [B5] tester1 and tester2
            [B9] tester1 and tester2                     [A5] tester1 and tester2
    `)

	t.Logf("* Get objects shared to tester1")
	listSharesTester1Request := makeHTTPRequestFromInterface(t, "GET", uriShares, nil)
	listSharesTester1Response, err := clients[tester1].Client.Do(listSharesTester1Request)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, listSharesTester1Response, "Bad status when listing objects shared to tester01")
	var sharedToTester1 protocol.ObjectResultset
	err = util.FullDecode(listSharesTester1Response.Body, &sharedToTester1)

	tester1Expects := []*protocol.Object{b3, b4, b8}
	tester1Exclude := []*protocol.Object{a1, a2, a3, a4, a5, b1, b2, b7}
	if excludingChildren {
		t.Logf("* Verify tester1 sees B3, B4, B8, but not A1-A5, B1, B2, B5, B6, B7, B9")
		tester1Exclude = append(tester1Exclude, b5, b6, b9)
	} else {
		t.Logf("* Verify tester1 sees B3, B4, B5, B6, B8, B9 but not A1-A5, B1, B2, B7")
		tester1Expects = append(tester1Expects, b5, b6, b9)
	}
	for _, o := range sharedToTester1.Objects {
		found := false
		for _, exclude := range tester1Exclude {
			if strings.Compare(o.ID, exclude.ID) == 0 {
				t.Logf("Object %s was found in tester1 shares when expected to be excluded", o.Name)
				found = true
				break
			}
		}
		if found {
			t.Fail()
		}
	}
	for i, expected := range tester1Expects {
		found := false
		for _, o := range sharedToTester1.Objects {
			if strings.Compare(o.ID, expected.ID) == 0 {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Tester1 expected object[%d] %s with id %s but it was not returned in shares", i, expected.Name, expected.ID)
			t.Fail()
		}
	}
	if t.Failed() {
		t.FailNow()
	}

	t.Logf("* Get objects shared to tester2")
	listSharesTester2Request := makeHTTPRequestFromInterface(t, "GET", uriShares, nil)
	listSharesTester2Response, err := clients[tester2].Client.Do(listSharesTester2Request)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, listSharesTester2Response, "Bad status when listing objects shared to tester02")
	var sharedToTester2 protocol.ObjectResultset
	err = util.FullDecode(listSharesTester2Response.Body, &sharedToTester2)

	t.Logf("* Verify tester2 sees A3, A4, A5 but not A1, A2, or any from B")
	tester2Expects := []string{a3.ID, a4.ID, a5.ID}
	tester2Exclude := []string{a1.ID, a2.ID, b1.ID, b2.ID, b3.ID, b4.ID, b5.ID, b6.ID, b7.ID, b8.ID, b9.ID}
	for _, o := range sharedToTester2.Objects {
		found := false
		for _, excludeID := range tester2Exclude {
			if strings.Compare(o.ID, excludeID) == 0 {
				t.Logf("Object %s was found in tester2 shares when expected to be excluded", o.Name)
				found = true
				break
			}
		}
		if found {
			t.Fail()
		}
	}
	for i, expectedID := range tester2Expects {
		found := false
		for _, o := range sharedToTester2.Objects {
			if strings.Compare(o.ID, expectedID) == 0 {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Tester2 expected object %d with id %s but it was not returned in shares", i, expectedID)
			t.Fail()
		}
	}
	if t.Failed() {
		t.FailNow()
	}

}

func TestListObjectsSharedToMeWithApostropheInDN595(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	userDN := "cn=d'angelo nicole e js0s962,ou=people,ou=sois,ou=dod,o=u.s. government,c=us"
	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	client := 10
	uriShares := mountPoint + "/shares?pageNumber=1&pageSize=30&sortField=modifieddate&"
	ACMeveryone := `{"banner":"UNCLASSIFIED","classif":"U","dissem_countries":["USA"],"portion":"U","version":"2.1.0"}`

	t.Logf("* Add object for d'angelo nicole so that acmgrantee record will exist")

	folderuri := mountPoint + "/objects"
	folder := protocol.Object{}
	folder.Name = "folder for nicole"
	folder.TypeName = "Folder"
	folder.RawAcm = ACMeveryone
	createFolderRequest := makeHTTPRequestFromInterface(t, "POST", folderuri, folder)
	createFolderRequest.Header.Add("USER_DN", userDN)
	createFolderRequest.Header.Add("SSL_CLIENT_S_DN", whitelistedDN)
	createFolderRequest.Header.Add("EXTERNAL_SYS_DN", whitelistedDN)
	createFolderResponse, err := clients[client].Client.Do(createFolderRequest)
	failNowOnErr(t, err, "Unable to do create request")
	statusMustBe(t, 200, createFolderResponse, "Bad status creating object")

	t.Logf("* Get objects shared to tester1")
	listSharesRequest := makeHTTPRequestFromInterface(t, "GET", uriShares, nil)
	listSharesRequest.Header.Add("USER_DN", userDN)
	listSharesRequest.Header.Add("SSL_CLIENT_S_DN", whitelistedDN)
	listSharesRequest.Header.Add("EXTERNAL_SYS_DN", whitelistedDN)
	listSharesResponse, err := clients[client].Client.Do(listSharesRequest)

	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, listSharesResponse, "Bad status when listing objects shared to the user")
	var sharedToUserDN protocol.ObjectResultset
	util.FullDecode(listSharesResponse.Body, &sharedToUserDN)
}
