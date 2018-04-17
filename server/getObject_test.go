package server_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/karlseguin/ccache"

	"github.com/deciphernow/object-drive-server/config"
	"github.com/deciphernow/object-drive-server/protocol"

	"github.com/deciphernow/object-drive-server/dao"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/server"
	"github.com/deciphernow/object-drive-server/services/aac"
	"github.com/deciphernow/object-drive-server/services/kafka"
	"github.com/deciphernow/object-drive-server/util"
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

	req, _ := NewGetObjectRequest(folderD.ID, "")
	resp, _ := clients[clientID].Client.Do(req)
	data, _ := ioutil.ReadAll(resp.Body)

	var obj protocol.Object
	json.Unmarshal(data, &obj)

	if len(obj.Breadcrumbs) != 3 {
		t.Errorf("expected Breadcrumbs length 1, got %v\n", len(obj.Breadcrumbs))
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

func TestGetObject_DeletedAncestorReturns409(t *testing.T) {
	clientID := 4

	folderA := makeFolderViaJSON("folderA", clientID, t)
	folderB := makeFolderWithParentViaJSON("folderB", folderA.ID, clientID, t)
	folderC := makeFolderWithParentViaJSON("folderC", folderB.ID, clientID, t)
	folderD := makeFolderWithParentViaJSON("folderD", folderC.ID, clientID, t)

	req, _ := NewDeleteObjectRequest(*folderB, "")
	_, _ = clients[clientID].Client.Do(req)

	req, _ = NewGetObjectRequest(folderD.ID, "")
	resp, _ := clients[clientID].Client.Do(req)
	defer util.FinishBody(resp.Body)

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("bad status: expected 409, but got %v", resp.StatusCode)
	}
}

func TestGetObject_TSClassificationIsRedactedInBreadcrumbs(t *testing.T) {

	tester10 := 0
	folderA, _ := makeFolderWithACMWithParentViaJSON("folderA", "", ValidACMTopSecretSharedToTester01, tester10)
	folderB, _ := makeFolderWithACMWithParentViaJSON("folderB", folderA.ID, ValidACMTopSecretSharedToTester01, tester10)
	folderC, _ := makeFolderWithACMWithParentViaJSON("folderC", folderB.ID, ValidACMTopSecretSharedToTester01, tester10)
	folderD, _ := makeFolderWithACMWithParentViaJSON("folderD", folderC.ID, ValidACMUnclassified, tester10)

	tester01 := 1
	req, _ := NewGetObjectRequest(folderD.ID, "")
	resp, _ := clients[tester01].Client.Do(req)
	data, _ := ioutil.ReadAll(resp.Body)
	var obj protocol.Object
	json.Unmarshal(data, &obj)

	t.Log("all parent breadcrumbs redacted for folderD")
	if len(obj.Breadcrumbs) != 0 {
		t.Errorf("expected Breadcrumbs length 1, got %v\n", len(obj.Breadcrumbs))
	}

}

func TestGetObject_PrivateObjectsRedactedInBreadcrumbs(t *testing.T) {
	tester10 := 0
	folderA, _ := makeFolderWithACMWithParentViaJSON("folderA", "", ValidACMUnclassifiedFOUOSharedToTester10, tester10)
	folderB, _ := makeFolderWithACMWithParentViaJSON("folderB", folderA.ID, ValidACMUnclassifiedFOUOSharedToTester10, tester10)
	folderC, _ := makeFolderWithACMWithParentViaJSON("folderC", folderB.ID, ValidACMUnclassifiedFOUOSharedToTester01, tester10)
	folderD, _ := makeFolderWithACMWithParentViaJSON("folderD", folderC.ID, ValidACMUnclassifiedFOUOSharedToTester01, tester10)

	t.Log("Only folderC is not redacted in our breadcrumbs for folderD")
	crumbs := []protocol.Breadcrumb{
		protocol.Breadcrumb{ID: folderC.ID, ParentID: folderC.ParentID, Name: folderC.Name},
	}

	tester01 := 1
	req, _ := NewGetObjectRequest(folderD.ID, "")
	resp, _ := clients[tester01].Client.Do(req)
	data, _ := ioutil.ReadAll(resp.Body)
	var obj protocol.Object
	json.Unmarshal(data, &obj)

	if len(obj.Breadcrumbs) != 1 {
		t.Errorf("expected Breadcrumbs length 1, got %v\n", len(obj.Breadcrumbs))
	}

	i := 0
	t.Logf("folderC is not redacted, so we see the name we defined above in breadcrumbs for folderD")
	if obj.Breadcrumbs[i].ID != crumbs[i].ID {
		t.Errorf("breadcrumb ID mismatch; expected: %s\tgot %s", crumbs[i].ID, obj.Breadcrumbs[i].ID)
	}
	if obj.Breadcrumbs[i].ParentID != crumbs[i].ParentID {
		t.Errorf("breadcrumb ParentID mismatch; expected: %s\tgot %s", crumbs[i].ParentID, obj.Breadcrumbs[i].ParentID)
	}
	if obj.Breadcrumbs[i].Name != crumbs[i].Name {
		t.Errorf("name should not have been redacted: %s", crumbs[i].Name)
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
	key, err := config.MaybeDecrypt(config.GetEnvOrDefault("OD_ENCRYPT_MASTERKEY", ""))
	if err != nil {
		t.Logf("unable to decrypt masterkey: %v", err)
		t.FailNow()
	}
	models.SetEncryptKey(key, &readPermission)
	perms := []models.ODObjectPermission{readPermission}
	obj := models.ODObject{Permissions: perms}
	obj.ID = []byte(guid)
	obj.RawAcm.String, obj.RawAcm.Valid = server.ValidACMUnclassified, true

	fakeServer := setupFakeServerWithObjectForUser(user, obj)

	// Simulate the getObject call.
	req, err := http.NewRequest("GET", mountPoint+objectURL, nil)
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

	snippetResponse := aac.SnippetResponse{
		Success:  true,
		Snippets: SnippetTP10,
		Found:    true,
	}
	attributesResponse := aac.UserAttributesResponse{
		Success:        true,
		UserAttributes: "{\"diasUserGroups\":{\"projects\":[{\"projectName\":\"DCTC\",\"groupNames\":[\"ODrive\"]}]}}",
	}
	acmInfo := aac.AcmInfo{
		Acm: server.ValidACMUnclassified,
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
		SnippetResp:        &snippetResponse,
		UserAttributesResp: &attributesResponse,
	}
	fakeQueue := kafka.NewFakeAsyncProducer(nil)
	fakeServer := server.AppServer{RootDAO: &fakeDAO,
		AAC:           &fakeAAC,
		UsersLruCache: ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50)),
		EventQueue:    fakeQueue,
	}

	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	fakeServer.ACLImpersonationWhitelist = append(fakeServer.ACLImpersonationWhitelist, whitelistedDN)

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

func TestGetObject_UserNotInDBAndObjectDoesNotExist(t *testing.T) {
	t.Skip()
	objectid := "abcdef0123456789abcdef0123456789"
	uri := mountPoint + "/objects/" + objectid + "/properties"
	server := 10
	userdn := "cn=fake user,ou=people,ou=sois,ou=dod,o=u.s. government,c=us"
	twldn := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		t.Logf("error %s", err.Error())
		t.FailNow()
	}
	req.Header.Add("USER_DN", userdn)
	req.Header.Add("SSL_CLIENT_S_DN", twldn)
	req.Header.Add("EXTERNAL_SYS_DN", twldn)
	res, err := clients[server].Client.Do(req)
	data, _ := ioutil.ReadAll(res.Body)
	t.Logf("Length of data is %d", len(data))
}

func TestGetObject_UserNotInDIASAndObjectDoesNotExist(t *testing.T) {
	t.Skip()
	objectid := "abcdef0123456789abcdef0123456789"
	uri := mountPoint + "/objects/" + objectid + "/properties"
	server := 10
	userdn := "cn=fake user,ou=person,ou=sois,ou=dod,o=u.s. government,c=us"
	twldn := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		t.Logf("error %s", err.Error())
		t.FailNow()
	}
	req.Header.Add("USER_DN", userdn)
	req.Header.Add("SSL_CLIENT_S_DN", twldn)
	req.Header.Add("EXTERNAL_SYS_DN", twldn)
	res, err := clients[server].Client.Do(req)
	data, _ := ioutil.ReadAll(res.Body)
	t.Logf("Length of data is %d", len(data))
}

func TestGetObject_500UsersAndObjectDoesNotExist(t *testing.T) {
	t.Skip()
	userdn := "cn=fake user,ou=people,ou=sois,ou=dod,o=u.s. government,c=us"
	qty := 500
	var wg sync.WaitGroup
	wg.Add(qty)
	for i := 1; i <= qty; i++ {
		newuser := strings.Replace(userdn, "fake user", fmt.Sprintf("fake user A %d", i), -1)
		go func(userdn string) {
			defer wg.Done()
			objectid := "abcdef0123456789abcdef0123456789"
			objectid = newGUID(t)
			uri := mountPoint + "/objects/" + objectid + "/properties"
			server := 10
			twldn := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
			req, err := http.NewRequest("GET", uri, nil)
			if err != nil {
				t.Logf("error %s", err.Error())
				t.FailNow()
			}
			req.Header.Add("USER_DN", userdn)
			req.Header.Add("SSL_CLIENT_S_DN", twldn)
			req.Header.Add("EXTERNAL_SYS_DN", twldn)
			t.Logf("fetching object %s as %s", objectid, userdn)
			res, err := clients[server].Client.Do(req)
			if err != nil {
				t.Logf("error doing client request: %s", err.Error())
			}
			if res != nil && res.Body != nil {
				_, _ = ioutil.ReadAll(res.Body)
			}
		}(newuser)
	}
	wg.Wait()
}

func TestGetObject_100UsersAndObjectDoesNotExistAsNewClient(t *testing.T) {
	t.Skip()
	userdn := "cn=fake user,ou=people,ou=sois,ou=dod,o=u.s. government,c=us"
	qty := 100
	var wg sync.WaitGroup
	wg.Add(qty)
	for i := 1; i <= qty; i++ {
		newuser := strings.Replace(userdn, "fake user", fmt.Sprintf("fake user B %d", i), -1)
		go func(userdn string) {
			defer wg.Done()
			objectid := "abcdef0123456789abcdef0123456789"
			objectid = newGUID(t)
			uri := mountPoint + "/objects/" + objectid + "/stream"
			server := 10
			twldn := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
			req, err := http.NewRequest("GET", uri, nil)
			if err != nil {
				t.Logf("error %s", err.Error())
				t.FailNow()
			}
			req.Header.Add("USER_DN", userdn)
			req.Header.Add("SSL_CLIENT_S_DN", twldn)
			req.Header.Add("EXTERNAL_SYS_DN", twldn)
			t.Logf("fetching object %s as %s", objectid, userdn)

			newTransport := &http.Transport{TLSClientConfig: clients[server].Config}
			newClient := &http.Client{Transport: newTransport}
			res, err := newClient.Do(req)
			if err != nil {
				t.Logf("error doing client request: %s", err.Error())
			}
			if res != nil {
				p := make([]byte, 70) // to read 7 of the 10 bytes expected, leaving some more needing read
				n, err := res.Body.Read(p)
				if err != nil && err != io.EOF {
					t.Logf("error reading body: %s", err.Error())
				}
				s := string(p[:n])
				t.Logf("response for %s: %s", objectid, s)
				_, _ = ioutil.ReadAll(res.Body) // if this isn't called, we end up exhausting file handles
			}
		}(newuser)
	}
	wg.Wait()
}
