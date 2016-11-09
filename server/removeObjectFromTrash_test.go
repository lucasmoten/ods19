package server_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/karlseguin/ccache"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/server"
	"decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestHTTPUndeleteObject(t *testing.T) {

	data := "Deletes are hard. Undeletes are harder!"

	if testing.Short() {
		t.Skip()
	}

	clientID := 6

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
	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName:       "Undelete a file",
			RequestDescription:  "We can use the id of a file to rescue it from the trash.",
			ResponseDescription: "This is the restored object.",
		},
	)
	res, err := clients[clientID].Client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	trafficLogs[APISampleFile].Response(t, res)
	defer util.FinishBody(res.Body)
	var objResponse protocol.Object

	err = util.FullDecode(res.Body, &objResponse)
	if err != nil {
		t.Errorf("Could not decode CreateObject response.")
	}
	res.Body.Close()
	// Save out the objectID as a string
	objID := objResponse.ID

	expected := objResponse.Name

	deleteReq, err := testhelpers.NewDeleteObjectRequest(objResponse, "", host)
	res, err = clients[clientID].Client.Do(deleteReq)
	if err != nil {
		t.Errorf("Delete request failed: %v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)

	// We must do another GET to get a valid changeToken
	getReq, err := testhelpers.NewGetObjectRequest(objID, "", host)
	if err != nil {
		t.Errorf("Error from NewGetObjectRequest: %v\n", err)
	}
	res, err = clients[clientID].Client.Do(getReq)
	if err != nil {
		t.Errorf("GetObject request failed: %v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	var getObjectResponse protocol.Object
	err = util.FullDecode(res.Body, &getObjectResponse)
	if err != nil {
		log.Printf("Error decoding json response from getObject to Object: %v", err)
		t.FailNow()
	}
	res.Body.Close()

	// This must be passed valid change token
	undeleteReq, err := testhelpers.NewUndeleteObjectDELETERequest(
		objID, getObjectResponse.ChangeToken, "", host)
	res, err = clients[clientID].Client.Do(undeleteReq)
	if err != nil {
		t.Errorf("Delete request failed: %v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// Assert object has been undeleted.
	var unDeletedObject protocol.Object
	err = util.FullDecode(res.Body, &unDeletedObject)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	res.Body.Close()
	//t.Log("UndeletedObject: ", unDeletedObject)

	if unDeletedObject.Name != expected {
		t.Errorf("Expected returned object name to be %v. Got: %v\n",
			expected, unDeletedObject.Name)
	}

}

func TestUndeleteExpungedObjectFails(t *testing.T) {

	user1, user2 := setupFakeUsers()

	expungedObj := testhelpers.NewTrashedObject(fakeDN1)
	expungedObj.IsExpunged = true

	snippetResp := testhelpers.GetTestSnippetResponse()

	fakeAAC := &aac.FakeAAC{
		SnippetResp: snippetResp,
		Err:         nil,
	}

	fakeDAO := &dao.FakeDAO{
		Object: expungedObj,
		Users:  []models.ODUser{user1, user2},
	}
	s := server.AppServer{
		RootDAO:       fakeDAO,
		UsersLruCache: ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50)),
		AAC:           fakeAAC,
	}

	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	s.AclImpersonationWhitelist = append(s.AclImpersonationWhitelist, whitelistedDN)

	guid, _ := util.NewGUID()
	fullURL := cfg.NginxRootURL + "/objects/" + guid + "/untrash"
	r, err := http.NewRequest(
		"POST", fullURL,
		bytes.NewBuffer([]byte(`{"changeToken": "1234567890"}`)))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Add("USER_DN", fakeDN1)
	r.Header.Add("SSL_CLIENT_S_DN", whitelistedDN)
	r.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.InitRegex()

	s.ServeHTTP(w, r)

	if respCode := w.Code; respCode != 410 {
		t.Errorf("Expected response code 410. Got: %v", respCode)
	}

}

func TestUndeleteObjectWithDeletedAncestorFails(t *testing.T) {

	user1, user2 := setupFakeUsers()

	withAncestorDeleted := testhelpers.NewTrashedObject(fakeDN1)
	withAncestorDeleted.IsAncestorDeleted = true

	snippetResp := testhelpers.GetTestSnippetResponse()

	fakeAAC := &aac.FakeAAC{
		SnippetResp: snippetResp,
		Err:         nil,
	}

	fakeDAO := &dao.FakeDAO{
		Object: withAncestorDeleted,
		Users:  []models.ODUser{user1, user2},
	}

	s := server.AppServer{
		RootDAO:       fakeDAO,
		AAC:           fakeAAC,
		UsersLruCache: ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50)),
	}

	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	s.AclImpersonationWhitelist = append(s.AclImpersonationWhitelist, whitelistedDN)

	guid, _ := util.NewGUID()
	fullURL := cfg.NginxRootURL + "/objects/" + guid + "/untrash"
	r, err := http.NewRequest(
		"POST", fullURL,
		bytes.NewBuffer([]byte(`{"changeToken": "1234567890"}`)))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Add("USER_DN", fakeDN1)
	r.Header.Add("SSL_CLIENT_S_DN", whitelistedDN)
	r.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.InitRegex()

	s.ServeHTTP(w, r)

	if respCode := w.Code; respCode != 405 {
		t.Errorf("Expected response code 405. Got: %v", respCode)
	}

}
