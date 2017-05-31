package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
	"decipher.com/object-drive-server/utils"
)

func TestCreateObjectMalicious(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	clientID := 5

	data := "Initial test data 1"
	//An exe name with some backspace chars to make it display as txt
	tmpName := "initialTestData1.exe\b\b\btxt"
	f, closer, err := testhelpers.GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer closer()

	jsonString := fmt.Sprintf(`
    {
      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" : "lol",
      "id":"deadbeef",
      "typeName": "File",
      "name": "",
      "description": "",
      "acm": "%s",
      "createdBy": "CN=POTUS,C=US"
    }
    `, jsonEscape(testhelpers.ValidACMUnclassified))
	t.Log(jsonString)
	jsonBody := []byte(jsonString)

	req, err := testhelpers.NewCreateObjectPOSTRequestRaw("objects", host, "", f, tmpName, jsonBody)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}

	client := clients[clientID].Client
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	//If it comes back ok, it at least needs to have
	//stopped us from doing something bad
	if res == nil {
		t.Fail()
	}

	var objResponse protocol.Object
	err = util.FullDecode(res.Body, &objResponse)
	res.Body.Close()

	//If invalid overrides are going to be ignored (a completely valid approach!),
	//ensure that they are in fact being ignored, and not taken as updates.
	if res.StatusCode == 200 {
		if objResponse.CreatedBy == "CN=POTUS,C=US" {
			t.Fail()
		}
		if objResponse.ID == "deadbeef" {
			t.Fail()
		}
	}
}

func TestCreateObjectSimple(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	clientID := 5
	data := "Initial test data 1"
	_, obj := doTestCreateObjectSimple(t, data, clientID, nil, nil, ValidAcmCreateObjectSimple)
	doCheckFileNowExists(t, clientID, obj)

	if len(obj.Permissions) == 0 {
		t.Errorf("Should return object permissions")
		t.FailNow()
	}

	if !obj.CallerPermission.AllowRead {
		t.Errorf("expected CallerPermission.AllowRead to be true for creator")
		t.FailNow()
	}

	for _, p := range obj.Permissions {
		t.Logf("%v", p)
	}
}

var ValidAcmCreateObjectSimple = `{"version":"2.1.0","classif":"U","dissem_ctrls":["FOUO"],"portion":"U//FOUO","banner":"UNCLASSIFIED//FOUO","dissem_countries":["USA"],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"]}`

func TestCreateWithACMInObjectFormat(t *testing.T) {

	// Test originates from a sample request from Rob Olson from email
	// on 2016-06-22T03:47Z that was failing when an ACM was provided
	// in object format instead of serialized string.

	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d", clientid)
		fmt.Println()
	}

	// Attempt to post a new object
	uri := host + cfg.NginxRootURL + "/objects"
	body := `--651f24479ab34530af50aa607ce7512c
Content-Disposition: form-data; name="ObjectMetadata"
Content-Type: application/json

{
    "contentType": "image/jpeg", 
    "name": "upload.txt", 
    "exemptFromFOIA": "No", 
    "acm": {
        "classif": "U", 
        "atom_energy": [], 
        "disp_only": "", 
        "disponly_to": [""], 
        "dissem_ctrls": [], 
        "fgi_protect": [], 
        "f_missions": [], 
        "dissem_countries": ["USA"], 
        "sar_id": [], 
        "sci_ctrls": [], 
        "version": "2.1.0", 
        "rel_to": [], 
        "f_atom_energy": [], 
        "f_macs": [], 
        "non_ic": [], 
        "f_oc_org": [], 
        "banner": "UNCLASSIFIED", 
        "f_sci_ctrls": [], 
        "fgi_open": [], 
        "f_accms": [], 
        "f_share": [], 
        "portion": "U", 
        "f_regions": [], 
        "f_clearance": ["u"], 
        "owner_prod": []
    }, 
    "typeName": "File", 
    "containsUSPersonsData": "No", 
    "description": "description"
}
--651f24479ab34530af50aa607ce7512c
Content-Disposition: form-data; name="filestream"; filename="upload.txt"
Content-Type: application/octet-stream

posting a file
--651f24479ab34530af50aa607ce7512c--   
`
	byteBody := []byte(body)
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(byteBody))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=651f24479ab34530af50aa607ce7512c")

	// do the request
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// process Response
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
}

// doGetObjectRequest gets an http status code and an object, and fails on error
func doGetObjectRequest(
	t *testing.T,
	clientID int,
	req *http.Request,
	expectedCode int,
	trafficLog *TrafficLog,
	description *TrafficLogDescription,
) *http.Response {
	if trafficLog != nil {
		trafficLog.Request(t, req, description)
	}
	res, err := clients[clientID].Client.Do(req)
	if err != nil && trafficLog != nil {
		trafficLog.Response(t, res)
	}
	if trafficLog != nil {
		trafficLog.Response(t, res)
	}
	t.Logf("check response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, expectedCode, res, "Bad status when creating object")
	return res
}

// doCreateObjectRequest gets an http status code and an object, and fails on error
func doCreateObjectRequestWithTrafficLog(t *testing.T, clientID int, req *http.Request, expectedCode int, trafficLogDescription *TrafficLogDescription) *protocol.Object {
	if trafficLogDescription != nil {
		trafficLogs[APISampleFile].Request(t, req, trafficLogDescription)
	}
	res, err := clients[clientID].Client.Do(req)
	if trafficLogDescription != nil {
		trafficLogs[APISampleFile].Response(t, res)
	}
	t.Logf("check response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, expectedCode, res, "Bad status when creating object")
	// since we do FullDecode unconditionally, no need to defer a FinishBody in this case.
	var createdObject protocol.Object
	err = util.FullDecode(res.Body, &createdObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	//Returning the res rather than StatusCode, because of statusMustBe, statusExpected, etc.
	return &createdObject
}

// doCreateObjectRequest gets an http status code and an object, and fails on error
func doCreateObjectRequest(t *testing.T, clientID int, req *http.Request, expectedCode int) *protocol.Object {
	return doCreateObjectRequestWithTrafficLog(t, clientID, req, expectedCode, nil)
}

func failWithoutDCTCOdrive(t *testing.T, createdObject *protocol.Object) {
	uriGetProperties := host + cfg.NginxRootURL + "/objects/" + createdObject.ID + "/properties"
	httpGet, _ := http.NewRequest("GET", uriGetProperties, nil)
	foundGrantee := false
	for clientIdx, ci := range clients {
		httpGetResponse, err := clients[clientIdx].Client.Do(httpGet)
		if err != nil {
			t.Logf("error making properties request")
			t.FailNow()
		}
		defer util.FinishBody(httpGetResponse.Body)
		if clientIdx == 0 {
			var retrievedObject protocol.Object
			err = util.FullDecode(httpGetResponse.Body, &retrievedObject)
			if err != nil {
				t.Logf("Error decoding json to Object: %v", err)
				t.Fail()
			}
			t.Logf("* Resulting permissions")
			hasEveryone := false
			for _, permission := range retrievedObject.Permissions {
				t.Logf("%s", permission)
				if strings.ToLower(permission.GroupName) == strings.ToLower(models.EveryoneGroup) {
					hasEveryone = true
				}
				if permission.Grantee == "dctc_odrive" {
					foundGrantee = true
					t.Logf("* found the permission that we want delete for")
					if !permission.AllowDelete {
						t.Logf("but permission for delete is not here")
					}
				}
			}
			if hasEveryone {
				t.Logf("FAIL: Did not expect permission with grantee %s", models.EveryoneGroup)
				t.Fail()
			}
		}
		switch clientIdx {
		case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("FAIL: Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // twl-server-generic and any others that may get added later
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("FAIL: Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}

	}
	if !foundGrantee {
		t.Logf("We did not find dctc_odrive grantee")
		t.FailNow()
	}
	if t.Failed() {
		t.FailNow()
	}
}

// TestCreateWithCantFlattenACM - can't flatten gives 400 - issue 18
func TestCreateWithCantFlattenACM(t *testing.T) {

	tester10 := 0

	t.Logf("* Create object")
	t.Logf("preparing")
	var object protocol.CreateObjectRequest
	object.Name = "TestCreateWithCantFlattenACM"
	object.RawAcm = `{"size":20}`
	permission := protocol.ObjectShare{Share: makeGroupShare("dctc", "DCTC", "ODrive"), AllowCreate: true, AllowRead: true, AllowUpdate: true, AllowDelete: true}
	object.Permissions = append(object.Permissions, permission)

	t.Logf("jsoninfying")
	jsonBody, _ := json.Marshal(object)
	uriCreate := host + cfg.NginxRootURL + "/objects"
	t.Logf("http request and client")
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")

	t.Logf("execute client")
	res, err := clients[tester10].Client.Do(httpCreate)
	t.Logf("check response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 400, res, "Bad status when creating object")
	messageMustContain(t, res, "acm not valid")
}

// TestCreateWithPermissions creates an object as Tester10, and includes a
// permission for create, read, update, and delete granted to ODrive group.
// All users in the group should be able to retrieve it, and update it.
// This test originates from cte/object-drive-server#93
func TestCreateWithPermissions(t *testing.T) {

	tester10 := 0

	t.Logf("* Create object")
	t.Logf("preparing")
	var object protocol.CreateObjectRequest
	object.Name = "TestCreateWithPermissions"
	object.RawAcm = `{"classif":"U"}`
	permission := protocol.ObjectShare{Share: makeGroupShare("dctc", "DCTC", "ODrive"), AllowCreate: true, AllowRead: true, AllowUpdate: true, AllowDelete: true}
	object.Permissions = append(object.Permissions, permission)
	t.Logf("jsoninfying")
	jsonBody, _ := json.Marshal(object)
	uriCreate := host + cfg.NginxRootURL + "/objects"
	t.Logf("http request and client")
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	t.Logf("execute client")
	createdObject := doCreateObjectRequest(t, tester10, httpCreate, 200)

	t.Logf("* Verify everyone in odrive group can read")
	shouldHaveReadForObjectID(t, createdObject.ID, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	failWithoutDCTCOdrive(t, createdObject)
}

func TestCreateWithPermissionsOwnedBy(t *testing.T) {

	tester10 := 0

	ownerNoDisplayName := "group/dctc/dctc/odrive"

	t.Logf("* Create object")
	t.Logf("preparing")
	var object protocol.CreateObjectRequest
	object.Name = "TestCreateWithPermissionsOwnedBy"
	object.RawAcm = `{"classif":"U"}`
	object.OwnedBy = "group/dctc/DCTC/ODrive/DCTC ODrive"
	permission := protocol.ObjectShare{Share: makeGroupShare("dctc", "DCTC", "ODrive"), AllowCreate: true, AllowRead: true, AllowUpdate: true, AllowDelete: true}
	object.Permissions = append(object.Permissions, permission)
	t.Logf("jsoninfying")
	jsonBody, _ := json.MarshalIndent(object, "", "  ")
	uriCreate := host + cfg.NginxRootURL + "/objects"
	t.Logf("http request and client")
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	t.Logf("execute client")
	createdObject := doCreateObjectRequestWithTrafficLog(t, tester10, httpCreate, 200, &TrafficLogDescription{
		OperationName:       "create owned by group",
		RequestDescription:  "add in ownedBy group",
		ResponseDescription: "object added, but immediately owned by the group",
	})
	t.Logf("ownedby: %s", createdObject.OwnedBy)
	t.Logf("* Verify everyone in odrive group can read")
	shouldHaveReadForObjectID(t, createdObject.ID, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	failWithoutDCTCOdrive(t, createdObject)

	if createdObject.OwnedBy != ownerNoDisplayName {
		t.Logf("owned by %s rather than %s", createdObject.OwnedBy, ownerNoDisplayName)
		t.FailNow()
	}
}

// TestCreateWithPermissionsNewUser has tester10 create an object shared to tester11 (who will does not yet exist)
func TestCreateWithPermissionsNewUser(t *testing.T) {

	tester10 := 0

	t.Logf("* Create object")
	t.Logf("preparing")
	var object protocol.CreateObjectRequest
	object.Name = "TestCreateWithPermissionsNewUser"
	object.RawAcm = testhelpers.ValidACMUnclassifiedFOUOSharedToTester11
	permission := protocol.ObjectShare{
		Share: makeUserShare(testhelpers.Tester11DN), AllowCreate: true, AllowRead: true, AllowUpdate: true, AllowDelete: true,
	}
	object.Permissions = append(object.Permissions, permission)

	t.Logf("jsoninfying")
	jsonBody, _ := json.Marshal(object)
	uriCreate := host + cfg.NginxRootURL + "/objects"
	t.Logf("http request and client")
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	t.Logf("execute client")
	_ = doCreateObjectRequest(t, tester10, httpCreate, 200)
}

// TestCreateWithPermissionsNewUser2 has tester02 create an object shared with tester10 on create
// then tester10 shares it to tester12, who does not yet exist.
// We document the share from tester10 to tester12 in the traffic log.
func TestCreateWithPermissionsNewUser2(t *testing.T) {

	tester02 := 2
	tester10 := 0

	//
	// tester02 creates an object initially shared to tester10
	//
	t.Logf("* Create object")
	t.Logf("preparing")
	var object protocol.CreateObjectRequest
	object.Name = "TestCreateWithPermissionsNewUser2"
	object.RawAcm = testhelpers.ValidACMUnclassifiedFOUOSharedToTester10
	permission := protocol.ObjectShare{
		Share:      makeUserShare(testhelpers.Tester10DN),
		AllowRead:  true,
		AllowShare: true,
	}
	object.Permissions = append(object.Permissions, permission)
	t.Logf("jsoninfying")
	jsonBody, _ := json.Marshal(object)
	uriCreate := host + cfg.NginxRootURL + "/objects"
	t.Logf("http request and client")
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	t.Logf("execute client")
	responseObj := doCreateObjectRequest(t, tester02, httpCreate, 200)

	//
	// tester10 shares to tester11 (who will never visit odrive)
	//
	t.Logf("* Create share granting read access to odrive") // will replace models.EveryoneGroup
	shareuri := host + cfg.NginxRootURL + "/shared/" + responseObj.ID
	shareSetting := protocol.ObjectShare{}
	shareSetting.Share = makeUserShare(testhelpers.Tester12DN)
	shareSetting.AllowRead = true
	jsonBody, err := json.MarshalIndent(shareSetting, "", "  ")
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	shareRequest, err := http.NewRequest("POST", shareuri, bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}

	trafficLogs[APISampleFile].Request(t, shareRequest, &TrafficLogDescription{
		OperationName:       "Share to new user that may use odrive in the future",
		RequestDescription:  "Share file owned by tester 2 from tester 10 to tester 11",
		ResponseDescription: "New user has the share",
	})

	shareResponse, err := clients[tester10].Client.Do(shareRequest)
	if err != nil {
		t.Logf("Unable to create share:%v", err)
		t.FailNow()
	}

	trafficLogs[APISampleFile].Response(t, shareResponse)

	defer util.FinishBody(shareResponse.Body)
	if shareResponse.StatusCode != http.StatusOK {
		t.Logf("share creation failed")
		t.FailNow()
	}

}

// TestCreateWithPermissionsNewUser3 has tester10 share an object to tester13 using resource string format,
// which we document in the traffic log.
func TestCreateWithPermissionsNewUser3(t *testing.T) {

	tester10 := 0

	t.Logf("* Create object")
	t.Logf("preparing")
	var object protocol.CreateObjectRequest
	object.Name = "TestCreateWithPermissionsNewUser3"
	object.RawAcm = testhelpers.ValidACMUnclassifiedFOUOSharedToTester13
	object.Permission.Read.AllowedResources = append(object.Permission.Read.AllowedResources, "user/"+testhelpers.Tester13DN)

	t.Logf("jsoninfying")
	jsonBody, _ := json.MarshalIndent(object, "", "  ")
	uriCreate := host + cfg.NginxRootURL + "/objects"
	t.Logf("http request and client")
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	t.Logf("execute client")
	trafficLogsDescription := &TrafficLogDescription{
		OperationName:       "Create object shared to new user on create with resource string format",
		RequestDescription:  "Create file using resource string format",
		ResponseDescription: "New user has the share",
	}
	_ = doCreateObjectRequestWithTrafficLog(t, tester10, httpCreate, 200, trafficLogsDescription)
}

func TestCreateStreamWithPermissions(t *testing.T) {
	genericTestCreateStreamWithPermissions(t, "", http.StatusOK)
}

func TestCreateStreamWithPermissionsOwnedBy(t *testing.T) {
	groupdn := "group/dctc/DCTC/ODrive/DCTC ODrive"
	groupdnnodisplayname := "group/dctc/dctc/odrive"
	obj := genericTestCreateStreamWithPermissions(t, groupdn, http.StatusOK)
	if groupdnnodisplayname != obj.OwnedBy {
		t.Logf("ownedBy was not properly set")
		t.FailNow()
	}
}

func TestCreateStreamWithPermissionsOwnedByEveryone(t *testing.T) {
	groupdn := "group/-Everyone"
	_ = genericTestCreateStreamWithPermissions(t, groupdn, http.StatusPreconditionRequired)
}

// TestCreateStreamWithPermissions creates an object as Tester10, and includes a
// permission for create, read, update, and delete granted to ODrive group.
// All users in the group should be able to retrieve it, and update it.
// This test originates from cte/object-drive-server#93
func genericTestCreateStreamWithPermissions(t *testing.T, ownedBy string, codeExpected int) *protocol.Object {

	tester10 := 0

	t.Logf("* Create object")
	t.Logf("preparing")
	var object protocol.CreateObjectRequest
	object.Name = "TestCreateWithPermissions"
	object.RawAcm = `{"classif":"U"}`
	if len(ownedBy) > 0 {
		object.OwnedBy = ownedBy
	}
	permission := protocol.ObjectShare{Share: makeGroupShare("dctc", "DCTC", "ODrive"), AllowCreate: true, AllowRead: true, AllowUpdate: true, AllowDelete: true}
	object.Permissions = append(object.Permissions, permission)
	t.Logf("jsoninfying")
	jsonBody, _ := json.MarshalIndent(object, "", "  ")

	t.Logf("http request and client")

	data := "Initial test data 2"
	//An exe name with some backspace chars to make it display as txt
	tmpName := "initialTestData2.txt"
	f, closer, err := testhelpers.GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer closer()

	req, err := testhelpers.NewCreateObjectPOSTRequestRaw("objects", host, "", f, tmpName, jsonBody)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}

	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName: "Create an object stream with explicit permissions set",
			RequestDescription: `
			This object is created with a user explicitly put into the DCTC group ODrive
			`,
			ResponseDescription: `
			The object should have have permissions put in according to what we explicitly set,
			rather than solely based on the ACM contents.
			Note that the original DN of the user is converted to lower case ("normalized").
			References to users and groups from permissions have a "flattened" DN which strips non alphanumeric
			(or underscore) characters.
			`,
		},
	)

	client := clients[tester10].Client
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	trafficLogs[APISampleFile].Response(t, res)
	defer util.FinishBody(res.Body)

	if res.StatusCode != codeExpected {
		t.FailNow()
	}

	var createdObject protocol.Object
	if res.StatusCode == http.StatusOK {
		err = util.FullDecode(res.Body, &createdObject)
		res.Body.Close()

		t.Logf("* Verify everyone in odrive group can read")
		shouldHaveReadForObjectID(t, createdObject.ID, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
		failWithoutDCTCOdrive(t, &createdObject)
	}

	return &createdObject
}

func TestCreateFoldersMultiLevelsDeep(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester1 := 1
	depth := 50
	createURI := host + cfg.NginxRootURL + "/objects"
	parentFolder := protocol.Object{}
	for curDepth := 1; curDepth < depth; curDepth++ {
		t.Logf("* Creating folder #%d", curDepth)
		newFolder := protocol.CreateObjectRequest{}
		newFolder.ParentID = parentFolder.ID
		newFolder.Name = fmt.Sprintf("Folders Multi Levels Deep %d", curDepth)
		newFolder.RawAcm = testhelpers.ValidACMUnclassified
		newFolder.TypeName = "Folder"
		createReq := makeHTTPRequestFromInterface(t, "POST", createURI, newFolder)
		createRes, err := clients[tester1].Client.Do(createReq)
		failNowOnErr(t, err, fmt.Sprintf("Unable to create folder #%d", curDepth))
		statusMustBe(t, 200, createRes, "Bad status when creating folder")
		createdFolder := protocol.Object{}
		err = util.FullDecode(createRes.Body, &createdFolder)
		failNowOnErr(t, err, "Error decoding json to Object")
		parentFolder = createdFolder
	}

}

func TestCreateObjectWithParentSetInJSON(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	t.Logf("* Create a folder under root as tester10")
	folder1 := makeFolderViaJSON("Test Folder 1 ", tester10, t)

	t.Logf("* Create a second folder, under the root, but with JSON properties specifying parent as folder1")
	folder2Obj := protocol.CreateObjectRequest{}
	folder2Obj.Name = "Test Folder 2"
	folder2Obj.ParentID = folder1.ID
	folder2Obj.TypeName = "Folder"
	folder2Obj.RawAcm = testhelpers.ValidACMUnclassified
	newobjuri := host + cfg.NginxRootURL + "/objects"
	createFolderReq := makeHTTPRequestFromInterface(t, "POST", newobjuri, folder2Obj)
	createFolderRes, err := clients[tester10].Client.Do(createFolderReq)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createFolderRes, "Bad status when creating folder 2 under root")
}

func TestCreateObjectWithUSPersonsData(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	t.Logf("* Creating object with US Persons Data")
	myobject := protocol.CreateObjectRequest{}
	myobject.Name = "This has US Persons Data"
	myobject.TypeName = "Arbitrary Object"
	myobject.RawAcm = testhelpers.ValidACMUnclassified
	myobject.ContainsUSPersonsData = "Yes"
	newobjuri := host + cfg.NginxRootURL + "/objects"
	createObjectReq := makeHTTPRequestFromInterface(t, "POST", newobjuri, myobject)
	createObjectRes, err := clients[tester10].Client.Do(createObjectReq)

	t.Logf("* Processing Response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createObjectRes, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(createObjectRes.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Checking metadata")
	if strings.Compare(createdObject.ContainsUSPersonsData, myobject.ContainsUSPersonsData) != 0 {
		t.Logf("response ContainsUSPersonsData didn't match request")
		t.FailNow()
	}
	if strings.Compare(createdObject.ContainsUSPersonsData, "Yes") != 0 {
		t.Logf("response ContainsUSPersonsData didn't = 'Yes'")
		t.FailNow()
	}
}

func TestCreateObjectWithUSPersonsDataNotSet(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	t.Logf("* Creating object with US Persons Data")
	myobject := protocol.CreateObjectRequest{}
	myobject.Name = "This has Unknown US Persons Data"
	myobject.TypeName = "Arbitrary Object"
	myobject.RawAcm = testhelpers.ValidACMUnclassified
	newobjuri := host + cfg.NginxRootURL + "/objects"
	createObjectReq := makeHTTPRequestFromInterface(t, "POST", newobjuri, myobject)
	createObjectRes, err := clients[tester10].Client.Do(createObjectReq)

	t.Logf("* Processing Response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createObjectRes, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(createObjectRes.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Checking metadata")
	if strings.Compare(createdObject.ContainsUSPersonsData, "Unknown") != 0 {
		t.Logf("response ContainsUSPersonsData didn't = 'Unknown'")
		t.Logf("Value returned was %s", createdObject.ContainsUSPersonsData)
		t.FailNow()
	}
}

func TestCreateObjectWithFOIAExempt(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	t.Logf("* Creating object with FOIA Exempt")
	myobject := protocol.CreateObjectRequest{}
	myobject.Name = "This has FOIA Exempt"
	myobject.TypeName = "Arbitrary Object"
	myobject.RawAcm = testhelpers.ValidACMUnclassified
	myobject.ExemptFromFOIA = "Yes"
	newobjuri := host + cfg.NginxRootURL + "/objects"
	createObjectReq := makeHTTPRequestFromInterface(t, "POST", newobjuri, myobject)
	createObjectRes, err := clients[tester10].Client.Do(createObjectReq)

	t.Logf("* Processing Response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createObjectRes, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(createObjectRes.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Checking metadata")
	if strings.Compare(createdObject.ExemptFromFOIA, myobject.ExemptFromFOIA) != 0 {
		t.Logf("response ExemptFromFOIA didn't match request")
		t.FailNow()
	}
	if strings.Compare(createdObject.ExemptFromFOIA, "Yes") != 0 {
		t.Logf("response ExemptFromFOIA didn't = 'Yes'")
		t.FailNow()
	}
}

func TestCreateObjectWithFOIAExemptNotSet(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	t.Logf("* Creating object with FOIA Exempt")
	myobject := protocol.CreateObjectRequest{}
	myobject.Name = "This has Unknown FOIA Exemption"
	myobject.TypeName = "Arbitrary Object"
	myobject.RawAcm = testhelpers.ValidACMUnclassified
	newobjuri := host + cfg.NginxRootURL + "/objects"
	createObjectReq := makeHTTPRequestFromInterface(t, "POST", newobjuri, myobject)
	createObjectRes, err := clients[tester10].Client.Do(createObjectReq)

	t.Logf("* Processing Response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createObjectRes, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(createObjectRes.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Checking metadata")
	if strings.Compare(createdObject.ExemptFromFOIA, "Unknown") != 0 {
		t.Logf("response ExemptFromFOIA didn't = 'Unknown'")
		t.Logf("Value returned was %s", createdObject.ExemptFromFOIA)
		t.FailNow()
	}
}

func TestCreateObjectWithPermissionsThatDontGrantToOwner(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	issue221 := `{
    "permissions": [
        {
            "allowShare": false,
            "allowRead": true,
            "share": {
                "users": [
                    "cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
                ]
            },
            "allowUpdate": true,
            "allowDelete": true,
            "allowCreate": true
        }
    ],
    "acm": {
        "fgi_open": [],
        "rel_to": [],
        "sci_ctrls": [],
        "owner_prod": [],
        "portion": "S",
        "disp_only": "",
        "disponly_to": [],
        "banner": "SECRET",
        "non_ic": [],
        "classif": "S",
        "atom_energy": [],
        "dissem_ctrls": [],
        "sar_id": [],
        "version": "2.1.0",
        "fgi_protect": [],
        "share": {
            "users": [
                "cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
            ],
            "projects": {}
        }
    },
    "progress": {
        "percentage": 1,
        "loading": true
    },
    "isShared": true,
    "content": {
        "ext": "png"
    },
    "type": "image/png",
    "file": {},
    "user_dn": "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
    "name": "Screen Shot 2016-08-26 at 4.01.54 PM.png"
}`
	newobjuri := host + cfg.NginxRootURL + "/objects"
	myobject, err := utils.UnmarshalStringToInterface(issue221)
	if err != nil {
		t.Logf("Error converting to interface: %s", err.Error())
		t.FailNow()
	}
	createObjectReq := makeHTTPRequestFromInterface(t, "POST", newobjuri, myobject)
	createObjectRes, err := clients[tester10].Client.Do(createObjectReq)

	t.Logf("* Processing Response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createObjectRes, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(createObjectRes.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")

}

func TestCreateObjectWithPermissions11ThatDontGrantToOwner(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	ghodsissue217 := `{
    "permission": { 
		"create": { 
			"allow": [
				"user/cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
	 		]
		},
		"read": {
			"allow": [
				"user/cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
	 		]			
		},
		"update": {
			"allow": [
				"user/cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
	 		]			
		},
		"delete": {
			"allow": [
				"user/cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
	 		]			
		}
	},	
    "acm": {
        "fgi_open": [],
        "rel_to": [],
        "sci_ctrls": [],
        "owner_prod": [],
        "portion": "S",
        "disp_only": "",
        "disponly_to": [],
        "banner": "SECRET",
        "non_ic": [],
        "classif": "S",
        "atom_energy": [],
        "dissem_ctrls": [],
        "sar_id": [],
        "version": "2.1.0",
        "fgi_protect": [],
        "share": {
            "users": [
                "cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
            ],
            "projects": {}
        }
    },
    "progress": {
        "percentage": 1,
        "loading": true
    },
    "isShared": true,
    "content": {
        "ext": "png"
    },
    "type": "image/png",
    "file": {},
    "user_dn": "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
    "name": "Screen Shot 2016-08-26 at 4.01.54 PM.png"
}`
	newobjuri := host + cfg.NginxRootURL + "/objects"
	myobject, err := utils.UnmarshalStringToInterface(ghodsissue217)
	if err != nil {
		t.Logf("Error converting to interface: %s", err.Error())
		t.FailNow()
	}
	createObjectReq := makeHTTPRequestFromInterface(t, "POST", newobjuri, myobject)
	createObjectRes, err := clients[tester10].Client.Do(createObjectReq)

	t.Logf("* Processing Response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createObjectRes, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(createObjectRes.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")

}

func TestCreateObjectWithPermissionFavoredOverOlderPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	ghodsissue217 := `{
    "permissions": [
        {
            "allowRead": true,
            "share": {
                "users": [
					"cn=test tester09,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
                ],
				"projects": {
					"dctc": {
						"disp_nm": "DCTC",
						"groups": ["ODrive"] 
					}
				}
            }
        }
    ],
	"permission": { 
		"create": { 
			"allow": [
				"user/cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
	 		]
		},
		"read": {
			"allow": [
				"user/cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us",
				"user/cn=test tester06,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
				"user/cn=test tester04,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	 		],
			"deny": [
				"user/cn=test tester04,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
			]
		},
		"update": {
			"allow": [
				"user/cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
	 		]			
		},
		"delete": {
			"allow": [
				"user/cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
	 		]			
		}
	},
    "acm": {
        "fgi_open": [],
        "rel_to": [],
        "sci_ctrls": [],
        "owner_prod": [],
        "portion": "S",
        "disp_only": "",
        "disponly_to": [],
        "banner": "SECRET",
        "non_ic": [],
        "classif": "S",
        "atom_energy": [],
        "dissem_ctrls": [],
        "sar_id": [],
        "version": "2.1.0",
        "fgi_protect": [],
        "share": {
            "users": [
                "cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
            ],
            "projects": {}
        }
    },
    "type": "image/png",
    "file": {},
    "user_dn": "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
    "name": "Screen Shot 2016-08-26 at 4.01.54 PM.png"
}`
	newobjuri := host + cfg.NginxRootURL + "/objects"
	myobject, err := utils.UnmarshalStringToInterface(ghodsissue217)
	if err != nil {
		t.Logf("Error converting to interface: %s", err.Error())
		t.FailNow()
	}
	createObjectReq := makeHTTPRequestFromInterface(t, "POST", newobjuri, myobject)
	createObjectRes, err := clients[tester10].Client.Do(createObjectReq)

	t.Logf("* Processing Response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createObjectRes, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(createObjectRes.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Verify the right users can read")
	shouldHaveReadForObjectID(t, createdObject.ID, 0, 6)
	shouldNotHaveReadForObjectID(t, createdObject.ID, 1, 2, 3, 4, 5, 7, 8, 9)
}

// TestCreateStreamWithNewPermissions creates an object as Tester10, and includes a
// permission for create, read, update, and delete granted to ODrive group.
// All users in the group should be able to retrieve it, and update it.
// This test originates from cte/object-drive-server#93, and is an extnsion for
// DecipherNow/object-drive-server#217
func TestCreateStreamWithNewPermissions(t *testing.T) {

	tester10 := 0

	t.Logf("* Create object")
	t.Logf("preparing")
	var object protocol.CreateObjectRequest
	object.Name = "TestCreateWithNewPermissions"
	object.RawAcm = `{"classif":"U"}`
	object.Permission = protocol.Permission{Create: protocol.PermissionCapability{AllowedResources: []string{"group/dctc/DCTC/ODrive/DCTC ODrive"}}, Read: protocol.PermissionCapability{AllowedResources: []string{"group/dctc/DCTC/ODrive/DCTC ODrive"}}, Update: protocol.PermissionCapability{AllowedResources: []string{"group/dctc/DCTC/ODrive/DCTC ODrive"}}, Delete: protocol.PermissionCapability{AllowedResources: []string{"group/dctc/DCTC/ODrive/DCTC ODrive"}}}
	t.Logf("jsoninfying")
	jsonBody, _ := json.MarshalIndent(object, "", "  ")

	t.Logf("http request and client")

	data := "Initial test data 2"
	//An exe name with some backspace chars to make it display as txt
	tmpName := "initialTestData2.txt"
	f, closer, err := testhelpers.GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer closer()

	req, err := testhelpers.NewCreateObjectPOSTRequestRaw("objects", host, "", f, tmpName, jsonBody)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}

	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName: "Create an object stream with explicit permissions set using API 1.1",
			RequestDescription: `
			This object is created with full CRUDS given to the owner, but explicit CRUD given to
			members of the DCTC ODrive group.
			`,
			ResponseDescription: `
			The object should have have permissions put in according to what we explicitly set,
			rather than solely based on the ACM contents.
			`,
		},
	)

	client := clients[tester10].Client
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
	trafficLogs[APISampleFile].Response(t, res)
	defer util.FinishBody(res.Body)
	if res.StatusCode != http.StatusOK {
		t.FailNow()
	}

	var createdObject protocol.Object
	err = util.FullDecode(res.Body, &createdObject)
	res.Body.Close()

	t.Logf("* Verify everyone in odrive group can read")
	shouldHaveReadForObjectID(t, createdObject.ID, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	failWithoutDCTCOdrive(t, &createdObject)
}

func TestCreateObjectWithPathing(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	t.Logf("* Create a folder under root as tester10")
	folder1 := makeFolderViaJSON("TestCreateObjectWithPathing", tester10, t)

	t.Logf("* Create a folder under that named this/is/an/object which the handler will expand hierarchially")
	folderuri := host + cfg.NginxRootURL + "/objects"
	folderA := protocol.CreateObjectRequest{}
	folderA.Name = "this/is/an/object"
	folderA.NamePathDelimiter = "/"
	folderA.TypeName = "Folder"
	folderA.ParentID = folder1.ID
	folderA.RawAcm = testhelpers.ValidACMUnclassified
	createFolderAReq := makeHTTPRequestFromInterface(t, "POST", folderuri, folderA)
	createFolderARes, err := clients[tester10].Client.Do(createFolderAReq)
	defer util.FinishBody(createFolderARes.Body)
	statusMustBe(t, http.StatusOK, createFolderARes, "bad status creating folders")
	var folder2 protocol.Object
	err = util.FullDecode(createFolderARes.Body, &folder2)
	if err != nil {
		t.Errorf("Error decoding json to object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify that folder2 parent is not null/empty, and not ID of folder1")
	if folder2.ParentID == folder1.ID {
		t.Errorf("Parent of folder2 was folder1")
		t.FailNow()
	}
	if folder2.ParentID == "" {
		t.Errorf("folder2 did not have a parent")
		t.FailNow()
	}

	t.Logf("* Traverse descendents of folder1, looking for expected object names")
	children := listChildren(folder1.ID, tester10, t)
	if children.TotalRows != 1 {
		t.Errorf("Expected folder1 to have 1 child")
		t.FailNow()
	}
	if children.Objects[0].Name != "this" {
		t.Errorf("Expected first child of folder1 to be `this`, but got %s", children.Objects[0].Name)
		t.FailNow()
	}
	children = listChildren(children.Objects[0].ID, tester10, t)
	if children.TotalRows != 1 {
		t.Errorf("Expected folder1 to have 1 child")
		t.FailNow()
	}
	if children.Objects[0].Name != "is" {
		t.Errorf("Expected child of `this` to be `is`, but got %s", children.Objects[0].Name)
		t.FailNow()
	}
	children = listChildren(children.Objects[0].ID, tester10, t)
	if children.TotalRows != 1 {
		t.Errorf("Expected folder1 to have 1 child")
		t.FailNow()
	}
	if children.Objects[0].Name != "an" {
		t.Errorf("Expected child of `is` to be `an`, but got %s", children.Objects[0].Name)
		t.FailNow()
	}
	children = listChildren(children.Objects[0].ID, tester10, t)
	if children.TotalRows != 1 {
		t.Errorf("Expected folder1 to have 1 child")
		t.FailNow()
	}
	if !strings.HasPrefix(children.Objects[0].Name, "object") {
		t.Errorf("Expected child of `an` to start with `object`, but got %s", children.Objects[0].Name)
		t.FailNow()
	}

	t.Logf("* Create a folder under the original folder also named this/is/an/object which the handler will expand hierarchially")
	folderB := protocol.CreateObjectRequest{}
	folderB.Name = "this/is/an/object"
	folderB.NamePathDelimiter = "/"
	folderB.TypeName = "Folder"
	folderB.ParentID = folder1.ID
	folderB.RawAcm = testhelpers.ValidACMUnclassified
	createFolderBReq := makeHTTPRequestFromInterface(t, "POST", folderuri, folderB)
	createFolderBRes, err := clients[tester10].Client.Do(createFolderBReq)
	defer util.FinishBody(createFolderBRes.Body)
	statusMustBe(t, http.StatusOK, createFolderBRes, "bad status creating folders")
	var folder3 protocol.Object
	err = util.FullDecode(createFolderBRes.Body, &folder3)
	if err != nil {
		t.Errorf("Error decoding json to object: %v", err)
		t.FailNow()
	}

	t.Logf("* Verify that folder3 parent is not null/empty, and not ID of folder1  and is not folder2, but has same parent as folder2")
	if folder3.ParentID == folder1.ID {
		t.Errorf("Parent of folder3 was folder1")
		t.FailNow()
	}
	if folder3.ParentID == "" {
		t.Errorf("folder3 did not have a parent")
		t.FailNow()
	}
	if folder3.ID == folder2.ID {
		t.Errorf("Folder3 is the same as folder2")
		t.FailNow()
	}
	if folder3.ParentID != folder2.ParentID {
		t.Errorf("Folder3 parent is not the same as folder2")
		t.FailNow()
	}

	t.Logf("* Create object with pathing is successful")
}

func TestCreateObjectWithPathingForGroup(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	t.Logf("* Create a folder under root owned by group which has pathing")
	DN4TP := strconv.FormatInt(time.Now().Unix(), 10)
	objectwithpathingforgroup := `
{
	"name": "TestCreateObjectWithPathingForGroup` + DN4TP + `/and/sub/folders",
	"namePathDelimiter": "/",
	"typeName": "Folder",
	"ownedBy": "group/dctc/DCTC/ODrive/DCTC ODrive",
    "acm": {
		"declass_dt": "2037-12-01T05:00:00.000",
        "fgi_open": [],
        "rel_to": [],
        "sci_ctrls": [],
        "owner_prod": [],
        "portion": "S",
        "disp_only": "",
        "disponly_to": [],
        "banner": "SECRET",
        "non_ic": [],
        "classif": "S",
        "atom_energy": [],
        "dissem_ctrls": [],
        "sar_id": [],
        "version": "2.1.0",
        "fgi_protect": [],
        "share": {
            "projects": {
				"dctc": {
					"disp_nm": "DCTC",
					"groups": ["ODrive"]
				}
			}
        }
    }
}`
	newobjuri := host + cfg.NginxRootURL + "/objects"
	myobject, err := utils.UnmarshalStringToInterface(objectwithpathingforgroup)
	if err != nil {
		t.Logf("Error converting to interface: %s", err.Error())
		t.FailNow()
	}
	createObjectReq := makeHTTPRequestFromInterface(t, "POST", newobjuri, myobject)
	createObjectRes, err := clients[tester10].Client.Do(createObjectReq)
	defer util.FinishBody(createObjectRes.Body)

	t.Logf("* Processing Response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createObjectRes, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(createObjectRes.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Leaf Node Created ID: %s", createdObject.ID)

	t.Logf("* Verify expected name")
	if createdObject.Name != "folders" {
		t.Errorf("Leaf node object was named %s, expected %s", createdObject.Name, "folders")
		t.FailNow()
	}

	t.Logf("* Verify created by tester10")
	tester10DN := "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	if createdObject.CreatedBy != tester10DN {
		t.Errorf("Object was created by %s, expected %s", createdObject.CreatedBy, tester10DN)
		t.FailNow()
	}

	t.Logf("* Verify owned by")
	groupResourceName := "group/dctc/dctc/odrive"
	if createdObject.OwnedBy != groupResourceName {
		t.Errorf("Object was owned by %s, expected %s", createdObject.OwnedBy, groupResourceName)
		t.FailNow()
	}

	t.Logf("* Verify has a parent")
	parentID := createdObject.ParentID
	if len(parentID) == 0 {
		t.Errorf("Object has no parent")
		t.FailNow()
	}

	t.Logf("* Traverse ancestors of created object, looking for expected object names")
	t.Logf("* First parent %s", parentID)
	parent := getObject(parentID, tester10, t)
	if parent.Name != "sub" {
		t.Errorf("Parent object named %s, expected %s", parent.Name, "sub")
		t.FailNow()
	}
	parentID = parent.ParentID
	if len(parentID) == 0 {
		t.Errorf("Object has no parent")
		t.FailNow()
	}
	t.Logf("* Second parent %s", parentID)
	parent = getObject(parentID, tester10, t)
	if parent.Name != "and" {
		t.Errorf("Parent object named %s, expected %s", parent.Name, "and")
		t.FailNow()
	}
	parentID = parent.ParentID
	if len(parentID) == 0 {
		t.Errorf("Object has no parent")
		t.FailNow()
	}
	t.Logf("* Root parent %s", parentID)
	parent = getObject(parentID, tester10, t)
	if parent.Name != `TestCreateObjectWithPathingForGroup`+DN4TP {
		t.Errorf("Parent object name %s, expected %s", parent.Name, `TestCreateObjectWithPathingForGroup`+DN4TP)
		t.FailNow()
	}
	parentID = parent.ParentID
	if len(parentID) != 0 {
		t.Errorf("Object is not at the root")
		t.FailNow()
	}

	t.Logf("* Create object with pathing is successful")
}

func TestCreateObjectWithACMHavingDate(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	ghodsissue508 := `{
    "permission": { 
		"create": { 
			"allow": [
				"user/cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
	 		]
		},
		"read": {
			"allow": [
				"user/cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
	 		]			
		},
		"update": {
			"allow": [
				"user/cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
	 		]			
		},
		"delete": {
			"allow": [
				"user/cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
	 		]			
		}
	},	
    "acm": {
		"declass_dt": "2037-12-01T05:00:00.000",
        "fgi_open": [],
        "rel_to": [],
        "sci_ctrls": [],
        "owner_prod": [],
        "portion": "S",
        "disp_only": "",
        "disponly_to": [],
        "banner": "SECRET",
        "non_ic": [],
        "classif": "S",
        "atom_energy": [],
        "dissem_ctrls": [],
        "sar_id": [],
        "version": "2.1.0",
        "fgi_protect": [],
        "share": {
            "users": [
                "cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
            ],
            "projects": {}
        }
    },
    "progress": {
        "percentage": 1,
        "loading": true
    },
    "isShared": true,
    "content": {
        "ext": "png"
    },
    "type": "test",
    "file": {},
    "user_dn": "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
    "name": "object-having-acm-with-declass_dt"
}`
	newobjuri := host + cfg.NginxRootURL + "/objects"
	myobject, err := utils.UnmarshalStringToInterface(ghodsissue508)
	if err != nil {
		t.Logf("Error converting to interface: %s", err.Error())
		t.FailNow()
	}
	createObjectReq := makeHTTPRequestFromInterface(t, "POST", newobjuri, myobject)
	createObjectRes, err := clients[tester10].Client.Do(createObjectReq)
	defer util.FinishBody(createObjectRes.Body)

	t.Logf("* Processing Response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createObjectRes, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(createObjectRes.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")
}

func TestCreateObjectAPISample662(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	tester10 := 0
	method := "POST"
	uri := host + cfg.NginxRootURL + "/objects"
	ghodsissue662 := `
--7518615725
Content-Disposition: form-data; name="ObjectMetadata"
Content-Type: application/json

{
  "typeName": "File",
  "name": "gettysburgaddress.txt",
  "description": "Description here",
  "parentId": "",
  "acm": {
    "classif": "U",
    "dissem_countries": [
      "USA"
    ],
    "share": {
      "users": [
        "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
        "CN=test tester02,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
        "CN=test tester03,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
        "CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US"
      ],
      "projects": [
        {
          "ukpn": {
            "disp_nm": "Project Name",
            "groups": [
              "Group Name",
              "Cats",
              "Dogs"
            ]
          },
          "ukpn2": {
            "disp_nm": "Project Name 2",
            "groups": [
              "Group 1",
              "Group 2",
              "Group 3"
            ]
          }
        }
      ]
    },
    "version": "2.1.0"
  },
  "permission": {
    "create": {
      "allow": [
        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
      ]
    },
    "read": {
      "allow": [
        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10",
        "group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1"
      ]
    },
    "update": {
      "allow": [
        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
      ]
    },
    "delete": {
      "allow": [
        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
      ]
    },
    "share": {
      "allow": [
        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
      ]
    }
  },
  "contentType": "text",
  "contentSize": "1511",
  "properties": [
    {
      "name": "Some Property",
      "value": "Some Property Value",
      "classificationPM": "U//FOUO"
    }
  ],
  "containsUSPersonsData": "No",
  "exemptFromFOIA": "No",
  "permissions": [
    {
      "share": {
        "users": [
          "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US"
        ]
      },
      "allowCreate": false,
      "allowRead": true,
      "allowUpdate": true,
      "allowDelete": false,
      "allowShare": false
    },
    {
      "share": {
        "projects": [
          {
            "dctc": {
              "disp_nm": "DCTC",
              "groups": [
                "ODrive_G1"
              ]
            }
          }
        ]
      },
      "allowCreate": false,
      "allowRead": true,
      "allowUpdate": false,
      "allowDelete": false,
      "allowShare": false
    }
  ]
}
--7518615725
Content-Disposition: form-data; name="filestream"; filename="test.txt"
Content-Type: application/octet-stream

This is the content of the file

--7518615725--
    
`
	t.Logf(`* Initial attempt using exact API sample ... "contentSize": "1511"`)
	var requestBuffer *bytes.Buffer
	requestBuffer = bytes.NewBufferString(ghodsissue662)
	req, err := http.NewRequest(method, uri, requestBuffer)
	if err != nil {
		t.Logf("Error setting up HTTP request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "multipart/form-data; boundary=7518615725")
	createObjectRes, err := clients[tester10].Client.Do(req)
	defer util.FinishBody(createObjectRes.Body)
	t.Logf("* Processing Response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 400, createObjectRes, "Bad status when creating object")

	t.Logf(`* Reattempt with the contentSize passed as a number instead of string ... "contentSize": 1511`)
	fixedghodsissue662 := strings.Replace(ghodsissue662, `"1511"`, `1511`, -1)
	var requestBuffer2 *bytes.Buffer
	requestBuffer2 = bytes.NewBufferString(fixedghodsissue662)
	req2, err := http.NewRequest(method, uri, requestBuffer2)
	if err != nil {
		t.Logf("Error setting up HTTP request: %v", err)
		t.FailNow()
	}
	req2.Header.Set("Content-Type", "multipart/form-data; boundary=7518615725")
	createObjectRes2, err := clients[tester10].Client.Do(req2)
	defer util.FinishBody(createObjectRes2.Body)
	t.Logf("* Processing Response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createObjectRes2, "Bad status when creating object")
	var createdObject protocol.Object
	err = util.FullDecode(createObjectRes2.Body, &createdObject)
	failNowOnErr(t, err, "Error decoding json to Object")
}
