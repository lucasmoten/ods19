package server_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/util"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/dao"
	"decipher.com/object-drive-server/cmd/metadataconnector/libs/server"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestListObjectsTrashedJSONResponse(t *testing.T) {

	user := models.ODUser{DistinguishedName: fakeDN1}

	obj1 := testhelpers.NewTrashedObject(user.DistinguishedName)
	resultset := models.ODObjectResultset{
		Objects: []models.ODObject{obj1},
	}
	fakeDAO := dao.FakeDAO{
		ObjectResultSet: resultset,
		Users:           []models.ODUser{user},
	}

	snippetResp := testhelpers.GetTestSnippetResponse()

	fakeAAC := aac.FakeAAC{
		SnippetResp: snippetResp,
		Err:         nil,
	}

	userCache := server.NewUserCache()
	snippetCache := server.NewSnippetCache()

	s := server.AppServer{
		DAO:           &fakeDAO,
		ServicePrefix: cfg.RootURLRegex,
		Users:         userCache,
		Snippets:      snippetCache,
		AAC:           &fakeAAC,
	}
	s.InitRegex()

	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	s.AclImpersonationWhitelist = append(s.AclImpersonationWhitelist, whitelistedDN)

	r, err := http.NewRequest("GET", cfg.RootURL+"/trashed?pageNumber=1&pageSize=50", nil)
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

	req, err := testhelpers.NewCreateObjectPOSTRequest(host, "", tmp)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}
	res, err := httpclients[clientID].Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	var objResponse protocol.Object
	err = util.FullDecode(res.Body, &objResponse)
	if err != nil {
		t.Errorf("Could not decode CreateObject response.")
	}
	res.Body.Close()

	expected := objResponse.Name

	deleteReq, err := testhelpers.NewDeleteObjectRequest(objResponse, "", host)
	_, err = httpclients[clientID].Do(deleteReq)
	if err != nil {
		t.Errorf("Delete request failed: %v\n", err)
	}

	trashURI := host + cfg.NginxRootURL + "/trashed?pageNumber=1&pageSize=1000"

	trashReq, err := http.NewRequest("GET", trashURI, nil)
	if err != nil {
		t.Errorf("Could not create trashReq: %v\n", err)
	}
	trashResp, err := httpclients[clientID].Do(trashReq)
	if err != nil {
		t.Errorf("Unable to do trash request:%v\n", err)
		t.FailNow()
	}

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

	// This operation no longer supports POST
	// // This time use a JSON POST, instead of GET
	// trashPOSTURI := host + cfg.RootURL + "/trash"
	// jsonRequest := `
	//  {"pageNumber": 1, "pageSize": 1000}
	// `
	// buf := bytes.NewBufferString(jsonRequest)

	// jsonTrashReq, err := http.NewRequest("POST", trashPOSTURI, buf)
	// if err != nil {
	// 	t.Errorf("Could not create http request: %v\n", err)
	// 	t.FailNow()
	// }
	// // Must set Content-Type for POST.
	// jsonTrashReq.Header.Set("Content-Type", "application/json")
	// jsonTrashResp, err := httpclients[clientID].Do(jsonTrashReq)

	// jsonTrashDecoder := json.NewDecoder(jsonTrashResp.Body)
	// var jsonTrashResponse protocol.ObjectResultset
	// err = jsonTrashDecoder.Decode(&jsonTrashResponse)
	// if err != nil {
	// 	t.Errorf("Could not decode listObjectsTrashed ObjectResultset response.")
	// }
	// objInJSONTrash := false
	// for _, o := range trashResponse.Objects {
	// 	if o.Name == expected {
	// 		objInJSONTrash = true
	// 		break
	// 	}
	// }
	// if !objInJSONTrash {
	// 	t.Errorf("Expected object to be in trash for user.")
	// }

}
