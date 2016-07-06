package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/util/testhelpers"

	"decipher.com/object-drive-server/cmd/odrive/libs/dao"
	"decipher.com/object-drive-server/cmd/odrive/libs/server"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/util"
)

func TestAppServerGetObject(t *testing.T) {

}

func TestAppServerGetObjectAgainstFake(t *testing.T) {

	// Set up an ODUser and a test DN.
	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	user := newUserForDN(whitelistedDN)

	// Create a GUID and construct a URL from it.
	guid := newGUID(t)

	objectURL := "/objects/" + guid + "/properties"

	// Create permissions object, with our User as a Grantee.
	perms := []models.ODObjectPermission{
		{Grantee: user.DistinguishedName, AllowRead: true}}
	obj := models.ODObject{Permissions: perms}
	obj.ID = []byte(guid)
	obj.RawAcm.String, obj.RawAcm.Valid = testhelpers.ValidACMUnclassified, true

	fakeServer := setupFakeServerWithObjectForUser(user, obj)

	// Simulate the getObject call.
	req, err := http.NewRequest(
		"GET", cfg.RootURL+objectURL, nil)
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

	userCache := server.NewUserCache()
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
		ServicePrefix: cfg.RootURLRegex,
		AAC:           &fakeAAC,
		Users:         userCache,
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
