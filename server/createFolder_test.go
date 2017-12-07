package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/deciphernow/object-drive-server/server"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/deciphernow/object-drive-server/utils"

	"github.com/deciphernow/object-drive-server/protocol"
)

func TestCreateFolderProtocol(t *testing.T) {

	jsonNoParent := fmt.Sprintf(`
    { "typeName": "Folder", "name": "",  "parentId": "", "acm": "%s", "contentType": "", "contentSize": 0 }`, jsonEscape(server.ValidACMUnclassified))

	t.Log(jsonNoParent)
	s := NewFakeServerWithDAOUsers()

	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	s.ACLImpersonationWhitelist = append(s.ACLImpersonationWhitelist, whitelistedDN)

	r, err := http.NewRequest("POST", mountPoint+"/objects", bytes.NewBuffer([]byte(jsonNoParent)))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Add("USER_DN", fakeDN0)
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
	uri := mountPoint + "/objects"
	if verboseOutput {
		fmt.Printf("(Verbose Mode) uri: %s\n", uri)
	}

	// Body
	folder := protocol.Object{}
	folder.Name = "Test Folder At Root " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.TypeName = "Folder"
	folder.RawAcm = server.ValidACMUnclassified
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
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
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
	uri := mountPoint + "/objects"
	if verboseOutput {
		fmt.Printf("(Verbose Mode) uri: %s\n", uri)
	}

	// Body
	folder := protocol.Object{}
	folder.Name = "Test Folder At Root " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.TypeName = "Folder"
	folder.ParentID = ""
	folder.RawAcm = server.ValidACMUnclassified
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
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)

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
	folder.RawAcm = server.ValidACMUnclassified
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
	res, err = clients[clientid].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)

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
	uri := mountPoint + "/objects"
	if verboseOutput {
		fmt.Printf("(Verbose Mode) uri: %s", uri)
		fmt.Println()
	}

	// Body
	folder := protocol.Object{}
	folder.Name = "Test Folder At Root " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.TypeName = "Folder"
	folder.ParentID = ""
	folder.RawAcm = server.ValidACMUnclassified
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
	createdFolder := doCreateObjectRequest(t, clientid1, req, 200)
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
	folder.RawAcm = server.ValidACMUnclassified
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

	res, err := clients[clientid2].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)

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
		t.Logf("(Verbose Mode) Using client id %d to create folder in root", clientid1)
		t.Logf("(Verbose Mode) Using client id %d to create subfolder", clientid2)
	}

	// URL
	uri := mountPoint + "/objects"
	if verboseOutput {
		t.Logf("(Verbose Mode) uri: %s", uri)
	}

	// Body
	t.Logf("* Creating folder at root")
	folder := protocol.CreateObjectRequest{}
	folder.Name = "Test Folder At Root " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.TypeName = "Folder"
	folder.ParentID = ""
	folder.RawAcm = server.ValidACMUnclassified
	grant2client2 := protocol.ObjectShare{}
	grant2client2.Share = makeUserShare(fakeDN1)
	grant2client2.AllowRead = true
	grant2client2.AllowCreate = true
	folder.Permissions = append(folder.Permissions, grant2client2)
	// grant2self is permission for the owner. This same permission, minus the read access
	// is implicitly granted by the server for owner when creating an object. Since a
	// permission for client2 is being established with read access, the object isn't
	// going to be shared with everyone per the ACM on server.ValidACMUnclassified
	// so read access needs to be established
	grant2self := protocol.ObjectShare{}
	grant2self.Share = makeUserShare(fakeDN0)
	grant2self.AllowCreate = true
	grant2self.AllowRead = true
	grant2self.AllowUpdate = true
	grant2self.AllowDelete = true
	grant2self.AllowShare = true
	folder.Permissions = append(folder.Permissions, grant2self)
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	t.Logf("posting request")
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := clients[clientid1].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	t.Logf("checking response")
	if res.StatusCode != http.StatusOK {
		t.Logf("response status %s, expected %d", res.Status, http.StatusOK)
		t.FailNow()
	}
	t.Logf("decoding response to protocol object")
	var createdFolder protocol.Object
	err = util.FullDecode(res.Body, &createdFolder)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	if verboseOutput {
		t.Logf("marshalling object to json string")
		jsonData, err := json.MarshalIndent(createdFolder, "", "  ")
		if err != nil {
			t.Logf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		t.Logf("Here is the response body:")
		t.Logf(string(jsonData))
	}

	t.Logf("* Creating subfolder")
	folder.ParentID = createdFolder.ID
	folder.Name = "Test Subfolder " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.RawAcm = server.ValidACMUnclassified
	jsonBody, err = json.Marshal(folder)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	t.Logf("request")
	req, err = http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err = clients[clientid2].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	t.Logf("validate response")
	if res.StatusCode != http.StatusOK {
		t.Logf("The second user was not allowed to create a folder but they should have permissions granted")
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
	uri := mountPoint + "/objects"
	if verboseOutput {
		fmt.Printf("(Verbose Mode) uri: %s", uri)
		fmt.Println()
	}

	// Body
	folder := protocol.Object{}
	folder.TypeName = "Folder"
	folder.RawAcm = server.ValidACMUnclassified
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
	res, err := clients[clientid1].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)

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

func TestCreateFolderWithPermissionForEveryone264(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	ghodsissue264 := `{
		"acm":"{\"version\":\"2.1.0\",\"classif\":\"U\",\"owner_prod\":[],\"atom_energy\":[],\"sar_id\":[],\"sci_ctrls\":[],\"disponly_to\":[\"\"],\"dissem_ctrls\":[],\"non_ic\":[],\"rel_to\":[],\"fgi_open\":[],\"fgi_protect\":[],\"portion\":\"U\",\"banner\":\"UNCLASSIFIED\",\"dissem_countries\":[\"USA\"],\"f_clearance\":[\"u\"],\"f_sci_ctrls\":[],\"f_accms\":[],\"f_oc_org\":[],\"f_regions\":[],\"f_missions\":[],\"f_share\":[],\"f_sar_id\":[],\"f_atom_energy\":[],\"f_macs\":[],\"disp_only\":\"\"}"
		,"permission":{
			"create":{
				"allow":[
					"user/cn=twl-server-generic2,ou=dae,ou=dia,o=u.s. government,c=us/twl-server-generic2"
					]
				}
			,"read":{
				"allow":[
					"user/cn=twl-server-generic2,ou=dae,ou=dia,o=u.s. government,c=us/twl-server-generic2"
					,"group/-Everyone/-Everyone"
					]
				}
			,"update":{
				"allow":[
					"user/cn=twl-server-generic2,ou=dae,ou=dia,o=u.s. government,c=us/twl-server-generic2"
					]
				}
			,"delete":{
				"allow":[
					"user/cn=twl-server-generic2,ou=dae,ou=dia,o=u.s. government,c=us/twl-server-generic2"
					]
				}
			,"share":{
				"allow":[
					"user/cn=twl-server-generic2,ou=dae,ou=dia,o=u.s. government,c=us/twl-server-generic2"
					]
				}
			}
		,"typeName":"Folder"
		,"name":"TEST1"
		,"description":"TEST1"
	}`
	newobjuri := mountPoint + "/objects"
	myobject, err := utils.UnmarshalStringToInterface(ghodsissue264)
	if err != nil {
		t.Logf("Error converting to interface: %s", err.Error())
		t.FailNow()
	}
	createObjectReq := makeHTTPRequestFromInterface(t, "POST", newobjuri, myobject)
	createObjectRes, err := clients[tester10].Client.Do(createObjectReq)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(createObjectRes.Body)

	// Response validation
	if createObjectRes.StatusCode != http.StatusOK {
		log.Printf("req1 bad status: %s", createObjectRes.Status)
		t.FailNow()
	}
	var createdFolder protocol.Object
	err = util.FullDecode(createObjectRes.Body, &createdFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}

	if createdFolder.Name != "TEST1" {
		t.Fail()
	}

	// Validating permissions against expectations
	theCreator := "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	explicitUser := "user/cn=twl-server-generic2,ou=dae,ou=dia,o=u.s. government,c=us"
	everyone := "group/-everyone"
	if !hasExpectedValues([]string{theCreator, explicitUser}, createdFolder.Permission.Create.AllowedResources) {
		t.Logf("Missing values expected in create")
		t.Fail()
	}
	if !hasExpectedValues([]string{theCreator, explicitUser, everyone}, createdFolder.Permission.Read.AllowedResources) {
		t.Logf("Missing values expected in read")
		t.Fail()
	}
	if !hasExpectedValues([]string{theCreator, explicitUser}, createdFolder.Permission.Update.AllowedResources) {
		t.Logf("Missing values expected in update")
		t.Fail()
	}
	if !hasExpectedValues([]string{theCreator, explicitUser}, createdFolder.Permission.Delete.AllowedResources) {
		t.Logf("Missing values expected in delete")
		t.Fail()
	}
	if !hasExpectedValues([]string{theCreator, explicitUser}, createdFolder.Permission.Share.AllowedResources) {
		log.Printf("Missing values expected in share")
		t.Fail()
	}

}

func hasExpectedValues(expected []string, actual []string) bool {
	for _, e := range expected {
		efound := false
		for _, a := range actual {
			if e == a {
				efound = true
				break
			}
		}
		if !efound {
			return false
		}
	}
	return true
}

func TestCreateFolderWithInvalidResourceString827(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	ghodsissue827 := `{
		"acm":"{\"version\":\"2.1.0\",\"classif\":\"U\",\"portion\":\"U\",\"banner\":\"UNCLASSIFIED\",\"dissem_countries\":[\"USA\"]}"
		,"permission":{
			"create":{
				"allow":[
					"user/cn=twl-server-generic2,ou=dae,ou=dia,o=u.s. government,c=us/twl-server-generic2"
					]
				}
			,"read":{
				"allow":[
					"user/cn=twl-server-generic2,ou=dae,ou=dia,o=u.s. government,c=us/twl-server-generic2"
					,"/group/-Everyone/-Everyone"
					,"/group/dctc/dctc/odrive"
					]
				}
			,"update":{
				"allow":[
					"user/cn=twl-server-generic2,ou=dae,ou=dia,o=u.s. government,c=us/twl-server-generic2"
					]
				}
			,"delete":{
				"allow":[
					"user/cn=twl-server-generic2,ou=dae,ou=dia,o=u.s. government,c=us/twl-server-generic2"
					]
				}
			,"share":{
				"allow":[
					"user/cn=twl-server-generic2,ou=dae,ou=dia,o=u.s. government,c=us/twl-server-generic2"
					]
				}
			}
		,"typeName":"Folder"
		,"name":"TEST827"
		,"description":"TEST827"
	}`
	newobjuri := mountPoint + "/objects"
	myobject, err := utils.UnmarshalStringToInterface(ghodsissue827)
	if err != nil {
		t.Logf("Error converting to interface: %s", err.Error())
		t.FailNow()
	}
	createObjectReq := makeHTTPRequestFromInterface(t, "POST", newobjuri, myobject)
	createObjectRes, err := clients[tester10].Client.Do(createObjectReq)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(createObjectRes.Body)

	// Response validation
	if createObjectRes.StatusCode != http.StatusBadRequest {
		t.Logf("req1 bad status: %s", createObjectRes.Status)
		t.FailNow()
	}

	responseBodyBytes := make([]byte, 500)
	l, err := createObjectRes.Body.Read(responseBodyBytes)
	if err != nil && err != io.EOF {
		t.Logf("Error reading response: %v", err)
		t.FailNow()
	}
	responseBodyString := strings.TrimSpace(string(responseBodyBytes[:l]))

	expectedBodyString1 := "Could not map request to internal struct type. unhandled format for resource string `/group/-Everyone/-Everyone`, must begin with `user/` or `group/`"
	expectedBodyString2 := "Could not map request to internal struct type. unhandled format for resource string `/group/dctc/dctc/odrive`, must begin with `user/` or `group/`"
	if responseBodyString != expectedBodyString1 && responseBodyString != expectedBodyString2 {
		t.Logf("Response wasn't an expected value: %s", responseBodyBytes)
		t.Fail()
	}

}
