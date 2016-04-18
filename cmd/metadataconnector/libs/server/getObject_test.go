package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	cfg "decipher.com/object-drive-server/config"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/dao"
	"decipher.com/object-drive-server/cmd/metadataconnector/libs/server"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/util"
)

func TestAppServerGetObject(t *testing.T) {

	// Set up an ODUser and a test DN.
	dn := "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	user := models.ODUser{
		DistinguishedName: dn,
	}
	user.CreatedBy = dn

	// Create a GUID and construct a URL from it.
	guid, err := util.NewGUID()
	if err != nil {
		t.Errorf("Could not create GUID.")
	}
	objectURL := "/objects/" + guid + "/properties"

	// Create permissions object, with our User as a Grantee.
	perms := []models.ODObjectPermission{
		{Grantee: user.DistinguishedName, AllowRead: true}}
	obj := models.ODObject{Permissions: perms}
	obj.ID = []byte(guid)
	obj.RawAcm.String = "Invalid ACM"
	obj.RawAcm.Valid = true

	// Fake the DAO interface.
	fakeDAO := dao.FakeDAO{
		Object: obj,
		Users:  []models.ODUser{user},
	}

	userCache := server.NewUserCache()
	snippetCache := server.NewSnippetCache()

	checkAccessResponse := aac.CheckAccessResponse{
		Success:   true,
		HasAccess: true,
	}
	// Fake the AAC interface
	fakeAAC := aac.FakeAAC{
		CheckAccessResp: &checkAccessResponse,
	}

	// Fake the AppServer.
	fakeServer := server.AppServer{DAO: &fakeDAO,
		ServicePrefix: cfg.RootURLRegex,
		AAC:           &fakeAAC,
		Users:         userCache,
		Snippets:      snippetCache,
	}
	fakeServer.InitRegex()

	// Simulate the getObject call.
	req, err := http.NewRequest(
		"GET", cfg.RootURL+objectURL, nil)
	req.Header.Add("USER_DN", dn)
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
