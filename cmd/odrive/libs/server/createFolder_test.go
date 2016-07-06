package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"

	"decipher.com/object-drive-server/protocol"
)

func TestCreateFolderProtocol(t *testing.T) {

	jsonNoParent := fmt.Sprintf(`
    { "typeName": "Folder", "name": "",  "parentId": "", "acm": "%s", "contentType": "", "contentSize": 0 }`, jsonEscape(testhelpers.ValidACMUnclassified))

	t.Log(jsonNoParent)
	s := NewFakeServerWithDAOUsers()

	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	s.AclImpersonationWhitelist = append(s.AclImpersonationWhitelist, whitelistedDN)

	r, err := http.NewRequest("POST", cfg.RootURL+"/objects", bytes.NewBuffer([]byte(jsonNoParent)))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Add("USER_DN", fakeDN1)
	r.Header.Add("SSL_CLIENT_S_DN", whitelistedDN)
	r.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected OK, got %v", w.Code)
	}
	var resp protocol.Object
	decoder := json.NewDecoder(w.Body)
	err = decoder.Decode(&resp)
	if err != nil {
		t.Errorf("Could not decode createFolder response as protocol.Object: %s", err)
	}

}

func TestCreateFolderAtRoot(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d\n", clientid)
	}

	// URL
	uri := host + cfg.NginxRootURL + "/objects"
	if verboseOutput {
		fmt.Printf("(Verbose Mode) uri: %s\n", uri)
	}

	// Body
	folder := protocol.Object{}
	folder.Name = "Test Folder At Root " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.TypeName = "Folder"
	folder.RawAcm = testhelpers.ValidACMUnclassified
	// Cannot use nil for string
	folder.ParentID = ""
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}

	// Request
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpclients[clientid].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	var createdFolder protocol.Object
	err = util.FullDecode(res.Body, &createdFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v\n", err)
		t.FailNow()
	}
	if verboseOutput {
		jsonData, err := json.MarshalIndent(createdFolder, "", "  ")
		if err != nil {
			log.Printf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		fmt.Println("Here is the response body:")
		fmt.Println(string(jsonData))
	}
}

func TestCreateFolderUnderFolderAtRoot(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d", clientid)
		fmt.Println()
	}

	// URL
	uri := host + cfg.NginxRootURL + "/objects"
	if verboseOutput {
		fmt.Printf("(Verbose Mode) uri: %s\n", uri)
	}

	// Body
	folder := protocol.Object{}
	folder.Name = "Test Folder At Root " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.TypeName = "Folder"
	folder.ParentID = ""
	folder.RawAcm = testhelpers.ValidACMUnclassified
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v\n", err)
		t.FailNow()
	}

	// Request
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v\n", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpclients[clientid].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	var createdFolder protocol.Object
	err = util.FullDecode(res.Body, &createdFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	if verboseOutput {
		jsonData, err := json.MarshalIndent(createdFolder, "", "  ")
		if err != nil {
			log.Printf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		fmt.Println("Here is the response body:")
		fmt.Println(string(jsonData))
	}

	// - This creates the subfolder
	folder.ParentID = createdFolder.ID
	folder.Name = "Test Subfolder " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.RawAcm = testhelpers.ValidACMUnclassified
	jsonBody, err = json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}

	// Request
	req, err = http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err = httpclients[clientid].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	var createdSubFolder protocol.Object
	err = util.FullDecode(res.Body, &createdSubFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	if verboseOutput {
		jsonData, err := json.MarshalIndent(createdSubFolder, "", "  ")
		if err != nil {
			log.Printf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		fmt.Println("Here is the response body:")
		fmt.Println(string(jsonData))
	}
}

func TestCreateFolderUnderFolderAtRootAsDifferentUserWithoutPermission(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid1 := 0
	clientid2 := 1

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d to create folder in root", clientid1)
		fmt.Printf("(Verbose Mode) Using client id %d to create subfolder", clientid2)
		fmt.Println()
	}

	// URL
	uri := host + cfg.NginxRootURL + "/objects"
	if verboseOutput {
		fmt.Printf("(Verbose Mode) uri: %s", uri)
		fmt.Println()
	}

	// Body
	folder := protocol.Object{}
	folder.Name = "Test Folder At Root " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.TypeName = "Folder"
	folder.ParentID = ""
	folder.RawAcm = testhelpers.ValidACMUnclassified
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}

	// Request
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpclients[clientid1].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	var createdFolder protocol.Object
	err = util.FullDecode(res.Body, &createdFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	if verboseOutput {
		jsonData, err := json.MarshalIndent(createdFolder, "", "  ")
		if err != nil {
			log.Printf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		fmt.Println("Here is the response body:")
		fmt.Println(string(jsonData))
	}

	// - This creates the subfolder
	folder.ParentID = createdFolder.ID
	folder.Name = "Test Subfolder " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.RawAcm = testhelpers.ValidACMUnclassified
	jsonBody, err = json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}

	// Request
	req, err = http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err = httpclients[clientid2].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode == http.StatusOK {
		log.Printf("The second user was allowed to create a folder even without grant!!!")
		t.FailNow()
	}
}

func TestCreateFolderUnderFolderAtRootAsDifferentUserWithPermission(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid1 := 0 // CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US
	clientid2 := 1 // CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d to create folder in root", clientid1)
		fmt.Printf("(Verbose Mode) Using client id %d to create subfolder", clientid2)
		fmt.Println()
	}

	// URL
	uri := host + cfg.NginxRootURL + "/objects"
	if verboseOutput {
		fmt.Printf("(Verbose Mode) uri: %s", uri)
		fmt.Println()
	}

	// Body
	folder := protocol.Object{}
	folder.Name = "Test Folder At Root " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.TypeName = "Folder"
	folder.ParentID = ""
	folder.RawAcm = testhelpers.ValidACMUnclassified
	grant2client2 := protocol.Permission{}
	grant2client2.Grantee = fakeDN1
	grant2client2.AllowRead = true
	grant2client2.AllowCreate = true
	folder.Permissions = append(folder.Permissions, grant2client2)
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}

	// Request
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpclients[clientid1].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("req1 bad status: %s", res.Status)
		t.FailNow()
	}
	var createdFolder protocol.Object
	err = util.FullDecode(res.Body, &createdFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	if verboseOutput {
		jsonData, err := json.MarshalIndent(createdFolder, "", "  ")
		if err != nil {
			log.Printf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		fmt.Println("Here is the response body:")
		fmt.Println(string(jsonData))
	}

	// - This creates the subfolder
	folder.ParentID = createdFolder.ID
	folder.Name = "Test Subfolder " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.RawAcm = testhelpers.ValidACMUnclassified
	jsonBody, err = json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}

	// Request
	req, err = http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err = httpclients[clientid2].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("The second user was not allowed to create a folder but they should have permissions granted")
		t.FailNow()
	}
}

func TestCreateFolderWithoutName(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid1 := 0 // CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d to create folder in root", clientid1)
		fmt.Println()
	}

	// URL
	uri := host + cfg.NginxRootURL + "/objects"
	if verboseOutput {
		fmt.Printf("(Verbose Mode) uri: %s", uri)
		fmt.Println()
	}

	// Body
	folder := protocol.Object{}
	folder.TypeName = "Folder"
	folder.RawAcm = testhelpers.ValidACMUnclassified
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}

	// Request
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpclients[clientid1].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("req1 bad status: %s", res.Status)
		t.FailNow()
	}
	var createdFolder protocol.Object
	err = util.FullDecode(res.Body, &createdFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}

	if len(createdFolder.Name) == 0 {
		t.Fail()
	}
	if !strings.HasPrefix(createdFolder.Name, "New") {
		t.Fail()
	}

}
