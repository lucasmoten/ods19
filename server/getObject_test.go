package server_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karlseguin/ccache"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util/testhelpers"

	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/server"
	"decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/util"
)

func TestGetObjectBreadcrumbs(t *testing.T) {

	clientID := 5
	folderA := makeFolderViaJSON("folderA", clientID, t)
	folderB := makeFolderWithParentViaJSON("folderB", folderA.ID, clientID, t)
	folderC := makeFolderWithParentViaJSON("folderC", folderB.ID, clientID, t)
	folderD := makeFolderWithParentViaJSON("folderD", folderC.ID, clientID, t)

	expected := []protocol.Breadcrumb{
		protocol.Breadcrumb{ID: folderA.ID, ParentID: folderA.ParentID, Name: folderA.Name},
		protocol.Breadcrumb{ID: folderB.ID, ParentID: folderB.ParentID, Name: folderB.Name},
		protocol.Breadcrumb{ID: folderC.ID, ParentID: folderC.ParentID, Name: folderC.Name},
	}

	req, _ := testhelpers.NewGetObjectRequest(folderD.ID, "", host)
	resp, _ := clients[clientID].Client.Do(req)
	data, _ := ioutil.ReadAll(resp.Body)

	var obj protocol.Object
	json.Unmarshal(data, &obj)

	if len(obj.Breadcrumbs) != 3 {
		t.Errorf("expected Breadcrumbs length 1, got %v\n", len(obj.Breadcrumbs))
	}

	if obj.Breadcrumbs[0].ParentID != "" {
		t.Errorf("first breadcrumb must have blank parentID")
	}

	t.Log("All names in folderD's breadcrumbs are visible")
	for i, _ := range obj.Breadcrumbs {
		if obj.Breadcrumbs[i].ID != expected[i].ID {
			t.Errorf("breadcrumb ID mismatch; expected: %s\tgot %s", expected[i].ID, obj.Breadcrumbs[i].ID)
		}
		if obj.Breadcrumbs[i].ParentID != expected[i].ParentID {
			t.Errorf("breadcrumb ParentID mismatch; expected: %s\tgot %s", expected[i].ParentID, obj.Breadcrumbs[i].ParentID)
		}
		if obj.Breadcrumbs[i].Name != expected[i].Name {
			t.Errorf("breadcrumb Name mismatch; expected: %s\tgot %s", expected[i].Name, obj.Breadcrumbs[i].Name)
		}
	}

}

func TestGetObject_DeletedAncestorReturns405(t *testing.T) {
	clientID := 4

	folderA := makeFolderViaJSON("folderA", clientID, t)
	folderB := makeFolderWithParentViaJSON("folderB", folderA.ID, clientID, t)
	folderC := makeFolderWithParentViaJSON("folderC", folderB.ID, clientID, t)
	folderD := makeFolderWithParentViaJSON("folderD", folderC.ID, clientID, t)

	req, _ := testhelpers.NewDeleteObjectRequest(*folderB, "", host)
	_, _ = clients[clientID].Client.Do(req)

	req, _ = testhelpers.NewGetObjectRequest(folderD.ID, "", host)
	resp, _ := clients[clientID].Client.Do(req)
	defer util.FinishBody(resp.Body)

	if resp.StatusCode != 405 {
		t.Errorf("bad status: expected 405, but got %v", resp.StatusCode)
	}
}

func TestGetObject_TSClassificationIsRedactedInBreadcrumbs(t *testing.T) {

	tester10 := 0
	folderA, _ := makeFolderWithACMWithParentViaJSON("folderA", "", testhelpers.ValidACMTopSecretSharedToTester01, tester10)
	folderB, _ := makeFolderWithACMWithParentViaJSON("folderB", folderA.ID, testhelpers.ValidACMTopSecretSharedToTester01, tester10)
	folderC, _ := makeFolderWithACMWithParentViaJSON("folderC", folderB.ID, testhelpers.ValidACMTopSecretSharedToTester01, tester10)
	folderD, _ := makeFolderWithACMWithParentViaJSON("folderD", folderC.ID, testhelpers.ValidACMUnclassified, tester10)

	redacted := []protocol.Breadcrumb{
		protocol.Breadcrumb{ID: folderA.ID, ParentID: folderA.ParentID, Name: folderA.Name},
		protocol.Breadcrumb{ID: folderB.ID, ParentID: folderB.ParentID, Name: folderB.Name},
		protocol.Breadcrumb{ID: folderC.ID, ParentID: folderC.ParentID, Name: folderC.Name},
	}

	tester01 := 1
	req, _ := testhelpers.NewGetObjectRequest(folderD.ID, "", host)
	resp, _ := clients[tester01].Client.Do(req)
	data, _ := ioutil.ReadAll(resp.Body)
	var obj protocol.Object
	json.Unmarshal(data, &obj)

	t.Log("Every name in folderD's breadcrumbs should be redacted based on classification")
	for i, _ := range obj.Breadcrumbs {
		if obj.Breadcrumbs[i].ID != redacted[i].ID {
			t.Errorf("breadcrumb ID mismatch; expected: %s\tgot %s", redacted[i].ID, obj.Breadcrumbs[i].ID)
		}
		if obj.Breadcrumbs[i].ParentID != redacted[i].ParentID {
			t.Errorf("breadcrumb ParentID mismatch; expected: %s\tgot %s", redacted[i].ParentID, obj.Breadcrumbs[i].ParentID)
		}
		if obj.Breadcrumbs[i].Name == redacted[i].Name {
			t.Errorf("name should have been redacted: %s", redacted[i].Name)
		}
	}

}

func TestGetObject_PrivateObjectsRedactedInBreadcrumbs(t *testing.T) {
	tester10 := 0
	folderA, _ := makeFolderWithACMWithParentViaJSON("folderA", "", testhelpers.ValidACMUnclassifiedFOUOSharedToTester10, tester10)
	folderB, _ := makeFolderWithACMWithParentViaJSON("folderB", folderA.ID, testhelpers.ValidACMUnclassifiedFOUOSharedToTester10, tester10)
	folderC, _ := makeFolderWithACMWithParentViaJSON("folderC", folderB.ID, testhelpers.ValidACMUnclassifiedFOUOSharedToTester01, tester10)
	folderD, _ := makeFolderWithACMWithParentViaJSON("folderD", folderC.ID, testhelpers.ValidACMUnclassifiedFOUOSharedToTester01, tester10)

	t.Log("Only the first two folders, A and B, should be redacted based on sharing permissions")
	crumbs := []protocol.Breadcrumb{
		protocol.Breadcrumb{ID: folderA.ID, ParentID: folderA.ParentID, Name: folderA.Name},
		protocol.Breadcrumb{ID: folderB.ID, ParentID: folderB.ParentID, Name: folderB.Name},
		protocol.Breadcrumb{ID: folderC.ID, ParentID: folderC.ParentID, Name: folderC.Name},
	}

	tester01 := 1
	req, _ := testhelpers.NewGetObjectRequest(folderD.ID, "", host)
	resp, _ := clients[tester01].Client.Do(req)
	data, _ := ioutil.ReadAll(resp.Body)
	var obj protocol.Object
	json.Unmarshal(data, &obj)

	for i, _ := range obj.Breadcrumbs {

		t.Logf("folderC is not redacted, so we see the name we defined above in breadcrumbs for folderD")
		if i > 1 {
			if obj.Breadcrumbs[i].ID != crumbs[i].ID {
				t.Errorf("breadcrumb ID mismatch; expected: %s\tgot %s", crumbs[i].ID, obj.Breadcrumbs[i].ID)
			}
			if obj.Breadcrumbs[i].ParentID != crumbs[i].ParentID {
				t.Errorf("breadcrumb ParentID mismatch; expected: %s\tgot %s", crumbs[i].ParentID, obj.Breadcrumbs[i].ParentID)
			}
			if obj.Breadcrumbs[i].Name != crumbs[i].Name {
				t.Errorf("name should not have been redacted: %s", crumbs[i].Name)
			}
			continue
		}

		t.Logf("folders A and B should be redacted in breadcrumbs for folderD")
		if obj.Breadcrumbs[i].ID != crumbs[i].ID {
			t.Errorf("breadcrumb ID mismatch; expected: %s\tgot %s", crumbs[i].ID, obj.Breadcrumbs[i].ID)
		}
		if obj.Breadcrumbs[i].ParentID != crumbs[i].ParentID {
			t.Errorf("breadcrumb ParentID mismatch; expected: %s\tgot %s", crumbs[i].ParentID, obj.Breadcrumbs[i].ParentID)
		}
		if obj.Breadcrumbs[i].Name == crumbs[i].Name {
			t.Errorf("name should have been redacted: %s", crumbs[i].Name)
		}
	}

}
func TestAppServerGetObjectAgainstFake(t *testing.T) {

	// Set up an ODUser and a test DN.
	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	user := newUserForDN(whitelistedDN)

	// Create a GUID and construct a URL from it.
	guid := newGUID(t)

	objectURL := "/objects/" + guid + "/properties"

	// Create permissions object, with our User as a Grantee.
	readPermission := models.ODObjectPermission{Grantee: user.DistinguishedName}
	readPermission.AllowRead = true
	models.SetEncryptKey("", &readPermission)
	perms := []models.ODObjectPermission{readPermission}
	obj := models.ODObject{Permissions: perms}
	obj.ID = []byte(guid)
	obj.RawAcm.String, obj.RawAcm.Valid = testhelpers.ValidACMUnclassified, true

	fakeServer := setupFakeServerWithObjectForUser(user, obj)

	// Simulate the getObject call.
	req, err := http.NewRequest("GET", config.RootURL+objectURL, nil)
	req.Header.Add("USER_DN", whitelistedDN)
	req.Header.Add("SSL_CLIENT_S_DN", whitelistedDN)

	if err != nil {
		t.Errorf("Error construction HTTP request")
	}
	w := httptest.NewRecorder()
	fakeServer.ServeHTTP(w, req)

	// Assertions.
	if w.Code != http.StatusOK {
		t.Errorf("Expected OK, got %v", w.Code)
	}

}

func setupFakeServerWithObjectForUser(user models.ODUser, obj models.ODObject) *server.AppServer {

	fakeDAO := dao.FakeDAO{
		Object: obj,
		Users:  []models.ODUser{user},
	}

	snippetCache := server.NewSnippetCache()

	snippetResponse := aac.SnippetResponse{
		Success:  true,
		Snippets: testhelpers.SnippetTP10,
	}
	acmInfo := aac.AcmInfo{
		Acm: testhelpers.ValidACMUnclassified,
	}
	acmResponse := aac.AcmResponse{
		Success:  true,
		AcmValid: true,
		AcmInfo:  &acmInfo,
	}
	fakeAAC := aac.FakeAAC{
		ACMResp: &acmResponse,
		CheckAccessResp: &aac.CheckAccessResponse{
			Success:   true,
			HasAccess: true,
		},
		SnippetResp: &snippetResponse,
	}

	fakeServer := server.AppServer{RootDAO: &fakeDAO,
		ServicePrefix: config.RootURLRegex,
		AAC:           &fakeAAC,
		UsersLruCache: ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50)),
		Snippets:      snippetCache,
		Auditor:       nil,
	}

	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	fakeServer.AclImpersonationWhitelist = append(fakeServer.AclImpersonationWhitelist, whitelistedDN)

	fakeServer.InitRegex()
	return &fakeServer
}

func newUserForDN(dn string) models.ODUser {
	user := models.ODUser{
		DistinguishedName: dn,
	}
	user.CreatedBy = dn
	return user
}

func newGUID(t *testing.T) string {
	guid, err := util.NewGUID()
	if err != nil {
		t.Errorf("Could not create GUID.")
	}
	return guid
}
