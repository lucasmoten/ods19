package server_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/karlseguin/ccache"

	"bitbucket.di2e.net/dime/object-drive-server/services/aac"
	"bitbucket.di2e.net/dime/object-drive-server/services/kafka"
	"bitbucket.di2e.net/dime/object-drive-server/util"

	"bitbucket.di2e.net/dime/object-drive-server/dao"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/server"
)

func TestListObjectsTrashedJSONResponse(t *testing.T) {

	user := models.ODUser{DistinguishedName: fakeDN1}

	obj1 := NewTrashedObject(user.DistinguishedName)
	resultset := models.ODObjectResultset{
		Objects: []models.ODObject{obj1},
	}
	fakeDAO := dao.FakeDAO{
		ObjectResultSet: resultset,
		Users:           []models.ODUser{user},
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

	fakeAAC := aac.FakeAAC{
		SnippetResp:        &snippetResponse,
		UserAttributesResp: &attributesResponse,
		Err:                nil,
	}

	fakeQueue := kafka.NewFakeAsyncProducer(nil)
	s := server.AppServer{
		RootDAO:         &fakeDAO,
		UsersLruCache:   ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50)),
		UserAOsLruCache: ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50)),
		TypeLruCache:    ccache.New(ccache.Configure().MaxSize(100).ItemsToPrune(5)),
		AAC:             &fakeAAC,
		EventQueue:      fakeQueue,
		Conf:            FakeServerSettingsConfiguration(),
	}
	s.InitRegex()

	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	s.ACLImpersonationWhitelist = append(s.ACLImpersonationWhitelist, whitelistedDN)

	r, err := http.NewRequest("GET", mountPoint+"/trashed?pageNumber=1&pageSize=50", nil)
	r.Header.Set("USER_DN", user.DistinguishedName)
	r.Header.Set("SSL_CLIENT_S_DN", whitelistedDN)

	if err != nil {
		t.Errorf("Error setting up http request: %v", err)
	}
	w := httptest.NewRecorder()

	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fail()
	}

	protocolReponse, err := protocol.NewObjectResultsetFromJSONBody(w.Body)
	if err != nil {
		t.Errorf("Error parsing JSON response: %v\n", err)
	}
	if len(protocolReponse.Objects) != len(resultset.Objects) {
		t.Errorf("Expected length of json reponse Objects array to be the same as DAO resultset.")
	}

}

func TestHTTPListObjectsTrashed(t *testing.T) {

	data := "Roads? Where we're going we don't need roads."

	if testing.Short() {
		t.Skip()
	}

	clientID := 5

	tmp, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}

	tmp.WriteString(data)
	defer func() {
		name := tmp.Name()
		tmp.Close()
		err = os.Remove(name)
	}()

	req, err := NewCreateObjectPOSTRequest("", tmp)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}
	res, err := clients[clientID].Client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	var objResponse protocol.Object
	err = util.FullDecode(res.Body, &objResponse)
	if err != nil {
		t.Errorf("Could not decode CreateObject response.")
	}
	res.Body.Close()

	expected := objResponse.Name

	deleteReq, err := NewDeleteObjectRequest(objResponse, "")
	deleteRes, err := clients[clientID].Client.Do(deleteReq)
	if err != nil {
		t.Errorf("Delete request failed: %v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(deleteRes.Body)
	trashURI := mountPoint + "/trashed?pageNumber=1&pageSize=1000"

	trashReq, err := http.NewRequest("GET", trashURI, nil)
	if err != nil {
		t.Errorf("Could not create trashReq: %v\n", err)
	}
	trashResp, err := clients[clientID].Client.Do(trashReq)
	if err != nil {
		t.Errorf("Unable to do trash request:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(trashResp.Body)

	var trashResponse protocol.ObjectResultset
	err = util.FullDecode(trashResp.Body, &trashResponse)
	if err != nil {
		t.Errorf("Could not decode listObjectsTrashed ObjectResultset response.")
	}

	objInTrash := false
	for _, o := range trashResponse.Objects {
		if o.Name == expected {
			objInTrash = true
			break
		}
	}

	if !objInTrash {
		t.Errorf("Expected object to be in trash for user.")
	}

}
