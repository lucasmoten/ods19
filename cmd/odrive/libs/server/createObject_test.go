package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

func jsonEscape(i string) string {
	o := i
	o = strings.Replace(o, "\"", "\\\"", -1)
	return o
}

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
	_, obj := doTestCreateObjectSimple(t, data, clientID)
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
		logPermission(t, p)
	}
}

var ValidAcmCreateObjectSimple = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":["FOUO"],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U//FOUO","banner":"UNCLASSIFIED//FOUO","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

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
func doGetObjectRequest(t *testing.T, clientID int, req *http.Request, expectedCode int) *http.Response {
	res, err := clients[clientID].Client.Do(req)
	t.Logf("check response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, expectedCode, res, "Bad status when creating object")
	return res
}

// doCreateObjectRequest gets an http status code and an object, and fails on error
func doCreateObjectRequest(t *testing.T, clientID int, req *http.Request, expectedCode int) *protocol.Object {
	res, err := clients[clientID].Client.Do(req)
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
				logPermission(t, permission)
				if permission.Grantee == models.EveryoneGroup {
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

// TestCreateWithPermissions creates an object as Tester10, and includes a
// permission for create, read, update, and delete granted to ODrive group.
// All users in the group should be able to retrieve it, and update it.
// This test originates from cte/object-drive-server#93
func TestCreateWithPermissions(t *testing.T) {

	tester10 := 0

	t.Logf("* Create object")
	t.Logf("preparing")
	var object protocol.Object
	object.Name = "TestCreateWithPermissions"
	object.RawAcm = `{"classif":"U"}`
	permission := protocol.Permission{Grantee: "dctc_odrive", AllowCreate: true, AllowRead: true, AllowUpdate: true, AllowDelete: true}
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

// TestCreateStreamWithPermissions creates an object as Tester10, and includes a
// permission for create, read, update, and delete granted to ODrive group.
// All users in the group should be able to retrieve it, and update it.
// This test originates from cte/object-drive-server#93
func TestCreateStreamWithPermissions(t *testing.T) {

	tester10 := 0

	t.Logf("* Create object")
	t.Logf("preparing")
	var object protocol.Object
	object.Name = "TestCreateWithPermissions"
	object.RawAcm = `{"classif":"U"}`
	permission := protocol.Permission{Grantee: "dctc_odrive", AllowCreate: true, AllowRead: true, AllowUpdate: true, AllowDelete: true}
	object.Permissions = append(object.Permissions, permission)
	t.Logf("jsoninfying")
	jsonBody, _ := json.Marshal(object)

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

	client := clients[tester10].Client
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
		t.FailNow()
	}
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
		newFolder := protocol.Object{}
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
	folder2Obj := protocol.Object{}
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
	myobject := protocol.Object{}
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
	myobject := protocol.Object{}
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
	myobject := protocol.Object{}
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
	myobject := protocol.Object{}
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
