package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/util"
	"bitbucket.di2e.net/dime/object-drive-server/utils"

	"bitbucket.di2e.net/dime/object-drive-server/protocol"
)

// input298 has variable input for fields id and changeToken.
var input298 = `
{
    "id": "%s",
    "acm": {
        "f_missions": [],
        "fgi_open": [],
        "rel_to": [],
        "dissem_countries": [
            "USA"
        ],
        "sci_ctrls": [],
        "f_clearance": [
            "u"
        ],
        "owner_prod": [],
        "f_regions": [],
        "f_share": [
            "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
            "cnlingchenoupeopleoujitfct_twlousix3systemsou_s_governmentcus"
        ],
        "portion": "U",
        "disp_only": "",
        "f_sci_ctrls": [],
        "disponly_to": [
            ""
        ],
        "banner": "UNCLASSIFIED",
        "non_ic": [],
        "f_accms": [],
        "f_sar_id": [],
        "f_oc_org": [],
        "classif": "U",
        "atom_energy": [],
        "dissem_ctrls": [],
        "sar_id": [],
        "version": "2.1.0",
        "fgi_protect": [],
        "f_macs": [],
        "f_atom_energy": [],
        "share": {
            "users": [
                "cn=ling chen,ou=people,ou=jitfct.twl,ou=six 3 systems,o=u.s. government,c=us",
                "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
                "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
            ],
            "projects": {}
        }
    },
    "permission": {
        "create": {
            "allow": [
                "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
            ]
        },
        "read": {
            "allow": [
                "user/cn=ling chen,ou=people,ou=jitfct.twl,ou=six 3 systems,o=u.s. government,c=us",
                "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
            ]
        },
        "update": {
            "allow": [
                "user/cn=ling chen,ou=people,ou=jitfct.twl,ou=six 3 systems,o=u.s. government,c=us",
                "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
            ]
        },
        "delete": {
            "allow": [
                "user/cn=ling chen,ou=people,ou=jitfct.twl,ou=six 3 systems,o=u.s. government,c=us",
                "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
            ]
        },
        "share": {
            "allow": [
                "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
            ]
        }
    },
    "changeToken": "%s"
}
`

func TestUpdateObject298(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	clientid := 0

	// Create 1 folders under root
	folder := makeFolderViaJSON("Test Folder for Update ", clientid, t)

	// Attempt to rename the folder
	updateuri := mountPoint + "/objects/" + folder.ID + "/properties"
	jsonBody := []byte(fmt.Sprintf(input298, folder.ID, folder.ChangeToken))

	req, err := http.NewRequest("POST", updateuri, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName:       "Properties update",
			RequestDescription:  "Ask for updated properties",
			ResponseDescription: "Get response",
		},
	)
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	trafficLogs[APISampleFile].Response(t, res)
	defer util.FinishBody(res.Body)
	// process Response
	if res.StatusCode != http.StatusOK {
		t.Logf("bad status: %s", res.Status)
		t.FailNow()
	}
	var updatedFolder protocol.Object
	err = util.FullDecode(res.Body, &updatedFolder)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	if testing.Verbose() {
		jsonData, err := json.MarshalIndent(updatedFolder, "", "  ")
		if err != nil {
			t.Logf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		t.Logf("Here is the response body:")
		t.Logf(string(jsonData))
	}
}

func TestUpdateObject(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	clientid := 0

	// Create 1 folders under root
	folder := makeFolderViaJSON("Test Folder for Update ", clientid, t)

	// Attempt to rename the folder
	updateuri := mountPoint + "/objects/" + folder.ID + "/properties"
	folder.Name = "Test Folder Updated " + strconv.FormatInt(time.Now().Unix(), 10)
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("POST", updateuri, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// process Response
	if res.StatusCode != http.StatusOK {
		t.Logf("bad status: %s", res.Status)
		t.FailNow()
	}
	var updatedFolder protocol.Object
	err = util.FullDecode(res.Body, &updatedFolder)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	if testing.Verbose() {
		jsonData, err := json.MarshalIndent(updatedFolder, "", "  ")
		if err != nil {
			t.Logf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		t.Logf("Here is the response body:")
		t.Logf(string(jsonData))
	}

}

func TestUpdateObjectToHaveNoName(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	clientid := 0

	// Create 1 folders under root
	folder := makeFolderViaJSON("Test Folder for Update ", clientid, t)

	// Attempt to rename the folder
	updateObjectRequest := protocol.UpdateObjectRequest{}
	updateObjectRequest.ID = folder.ID
	updateObjectRequest.Name = ""
	updateObjectRequest.ChangeToken = folder.ChangeToken
	updatedFolder, err := clients[clientid].C.UpdateObject(updateObjectRequest)
	if err != nil {
		t.Errorf("error calling update folder: %v", err)
		t.FailNow()
	}
	if strings.Compare(updatedFolder.Name, folder.Name) != 0 {
		t.Logf("Folder name is %s, expected it to be %s", updatedFolder.Name, folder.Name)
		t.FailNow()
	}
	if testing.Verbose() {
		jsonData, err := json.MarshalIndent(updatedFolder, "", "  ")
		if err != nil {
			t.Logf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		t.Logf("Here is the response body:")
		t.Logf(string(jsonData))
	}
}

func TestUpdateObjectToChangeOwnedBy(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	clientid := 0

	t.Logf("Create 1 folders under root")
	folder := makeFolderViaJSON("Test Folder for Update ", clientid, t)
	expectedOwner := folder.OwnedBy

	t.Logf("Attempt to change owner")
	updateuri := mountPoint + "/objects/" + folder.ID + "/properties"
	folder.OwnedBy = fakeDN2
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("POST", updateuri, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	t.Logf("do the request")
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)

	t.Logf("Need to parse the body and verify it didnt change")
	var updatedObject protocol.Object
	err = util.FullDecode(res.Body, &updatedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	if strings.Compare(updatedObject.OwnedBy, folder.OwnedBy) == 0 {
		t.Logf("Owner was changed to %s", updatedObject.OwnedBy)
		t.FailNow()
	}
	if strings.Compare(updatedObject.OwnedBy, expectedOwner) != 0 {
		t.Logf("Owner is not %s. It is %s", expectedOwner, updatedObject.OwnedBy)
		t.FailNow()
	}
}

func TestUpdateObjectPreventAcmShareChange(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester1 := 1
	tester2 := 2

	t.Logf("* Create folder as Tester01")
	folder := makeFolderViaJSON("TestUpdateObjectPreventAcmShareChange", tester1, t)

	t.Logf("* Tester01 Add a share allowing Tester02 to update")
	shareSetting := protocol.ObjectShare{}
	shareSetting.Share = makeUserShare(fakeDN2)
	shareSetting.AllowUpdate = true
	updatedFolder := doAddObjectShare(t, folder, &shareSetting, tester1)

	updateuri := mountPoint + "/objects/" + folder.ID + "/properties"

	t.Logf("* Tester02 updates name but leave ACM alone")
	updatedFolder.Name += " changed name"
	updateReq1 := makeHTTPRequestFromInterface(t, "POST", updateuri, updatedFolder)
	updateRes1, err := clients[tester2].Client.Do(updateReq1)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, updateRes1, "Bad status when updating object")
	err = util.FullDecode(updateRes1.Body, &updatedFolder)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Tester02 update name again, as well as ACM without changing share")
	updatedFolder.Name += " again"
	updatedFolder.RawAcm = ValidACMUnclassifiedFOUO
	updateReq2 := makeHTTPRequestFromInterface(t, "POST", updateuri, updatedFolder)
	updateRes2, err := clients[tester2].Client.Do(updateReq2)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, updateRes2, "Bad status when updating object")
	err = util.FullDecode(updateRes2.Body, &updatedFolder)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Tester02 update name + acm with a different share. Expect error")
	updatedFolder.Name += " and share"
	updatedFolder.RawAcm = ValidACMUnclassifiedFOUOSharedToTester01And02
	updateReq3 := makeHTTPRequestFromInterface(t, "POST", updateuri, updatedFolder)
	updateRes3, err := clients[tester2].Client.Do(updateReq3)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 403, updateRes3, "Bad status when updating object")
	messageMustContain(t, updateRes3, "User does not have permission to change the share for this object")
	ioutil.ReadAll(updateRes3.Body)
	updateRes3.Body.Close()

	t.Logf("* Tester01 Add a share allowing Tester02 to share")
	shareSetting2 := protocol.ObjectShare{}
	shareSetting2.Share = makeUserShare(fakeDN2)
	shareSetting2.AllowShare = true
	updatedFolder = doAddObjectShare(t, updatedFolder, &shareSetting2, tester1)

	t.Logf("* Tester02 update name + acm with a different share. Expect success")
	updatedFolder.Name += " and share"
	updatedFolder.RawAcm = ValidACMUnclassifiedFOUOSharedToTester01And02
	updateReq4 := makeHTTPRequestFromInterface(t, "POST", updateuri, updatedFolder)
	updateRes4, err := clients[tester2].Client.Do(updateReq4)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, updateRes4, "Bad status when updating object")
	err = util.FullDecode(updateRes4.Body, &updatedFolder)
	failNowOnErr(t, err, "Error decoding json to Object")

}

func TestUpdateObjectWithDifferentIDInJSON(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0
	tester1 := 1

	t.Logf("* Create folder1 under root as tester10")
	folder1 := makeFolderViaJSON("Test Folder 1 ", tester10, t)
	t.Logf("* Create two folder2 under root as tester1")
	folder2 := makeFolderViaJSON("Test Folder 2 ", tester1, t)

	t.Logf("* Attempt to Update folder1, using folder2's id")
	updateuri := mountPoint + "/objects/" + folder1.ID + "/properties"
	folder1.ID = folder2.ID
	folder1.Name = "Please dont let me save this"

	updateFolderReq := makeHTTPRequestFromInterface(t, "POST", updateuri, folder1)
	updateFolderRes, err := clients[tester10].Client.Do(updateFolderReq)
	failNowOnErr(t, err, "Unable to do request")
	statusExpected(t, 400, updateFolderRes, "Bad status when updating folder 1 using folder2 id")
	var updatedFolder protocol.Object
	err = util.FullDecode(updateFolderRes.Body, &updatedFolder)
	if t.Failed() {
		t.Logf("  Name of object updated is .. %s", updatedFolder.Name)

		geturi := mountPoint + "/objects/" + folder2.ID + "/properties"
		getObjReq := makeHTTPRequestFromInterface(t, "GET", geturi, nil)
		getObjRes, err := clients[tester10].Client.Do(getObjReq)
		failNowOnErr(t, err, "Unable to do request")
		statusExpected(t, 200, getObjRes, "Bad status when getting folder 2")
		var retrievedFolder protocol.Object
		err = util.FullDecode(getObjRes.Body, &retrievedFolder)
		t.Logf(" Folder 2 name is .. %s", retrievedFolder.Name)

	}
}

func TestRenameObject(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	t.Logf("* Create folder under root as tester10")
	folder1 := makeFolderViaJSON("Test Folder 1 ", tester10, t)

	t.Logf("* Attempt to rename it")
	changedName := "Renamed"
	updateuri := mountPoint + "/objects/" + folder1.ID + "/properties"
	folder1.Name = changedName

	updateFolderReq := makeHTTPRequestFromInterface(t, "POST", updateuri, folder1)
	updateFolderRes, err := clients[tester10].Client.Do(updateFolderReq)
	failNowOnErr(t, err, "Unable to do request")
	statusExpected(t, 200, updateFolderRes, "Bad status when renaming folder")
	var updatedFolder protocol.Object
	err = util.FullDecode(updateFolderRes.Body, &updatedFolder)
	if strings.Compare(updatedFolder.Name, changedName) != 0 {
		t.Logf(" Name is %s expected %s", updatedFolder.Name, changedName)
		t.FailNow()
	}
}

func TestUpdateObjectCallerPermissions(t *testing.T) {
	tester9 := 9
	t.Logf("* Create folder as tester9")
	folder := makeFolderViaJSON("caller_permissions_test", tester9, t)
	t.Logf("* Update folder by changing name")
	uri := mountPoint + "/objects/" + folder.ID + "/properties"
	folder.Name = "Renamed"
	req := makeHTTPRequestFromInterface(t, "POST", uri, folder)
	resp, err := clients[tester9].Client.Do(req)
	failNowOnErr(t, err, "unable to do request")
	var updated protocol.Object
	util.FullDecode(resp.Body, &updated)
	cperms := updated.CallerPermission
	if !allTrue(cperms.AllowCreate, cperms.AllowDelete, cperms.AllowRead,
		cperms.AllowShare, cperms.AllowUpdate) {
		t.Errorf("Expected creator of object to have all true CallerPermission but got: %v", cperms)
	}

	t.Logf("* Add UPDATE and SHARE permission for tester10")
	tester10 := 0
	var share protocol.ObjectShare
	share.AllowUpdate = true
	share.Share = makeUserShare(fakeDN0)
	jsonBody, _ := json.Marshal(share)
	shareURI := mountPoint + "/shared/" + folder.ID
	shareReq, _ := http.NewRequest("POST", shareURI, bytes.NewBuffer(jsonBody))
	shareReq.Header.Set("Content-Type", "application/json")
	resp, err = clients[tester9].Client.Do(shareReq)
	failNowOnErr(t, err, "could not do http request to share to tester10")
	statusMustBe(t, 200, resp, "expected tester9 to be able to share to tester10")

	t.Logf("* do get object with tester10")
	req, _ = http.NewRequest("GET", uri, nil)
	resp, err = clients[tester10].Client.Do(req)
	failNowOnErr(t, err, "could not do get object request with tester10")
	var folderForTester10 protocol.Object
	util.FullDecode(resp.Body, &folderForTester10)

	cperms = folderForTester10.CallerPermission
	if cperms.AllowShare || !cperms.AllowUpdate || !cperms.AllowRead {
		t.Errorf("Expected AllowShare to be false and AllowUpdate to be true, got: %v", cperms)
	}

	for _, p := range folderForTester10.Permissions {
		t.Logf("Permission: %v", p)
	}

}

func TestUpdateObjectShareInAFolder(t *testing.T) {
	tester10 := 0
	t.Logf("* Create parent folder as tester10")
	folderParent := makeFolderViaJSON("TestUpdateObjectShareInAFolderParent", tester10, t)
	t.Logf("* Create child folder as tester10")
	folderChild := makeFolderWithParentViaJSON("TestUpdateObjectShareInAFolderChild", folderParent.ID, tester10, t)
	t.Logf("* Change ACM Share settings")

	objStringTemplate := `{
    "id": "%s",
    "parentId": "%s",
    "acm": {
        "f_missions": [],
        "fgi_open": [],
        "rel_to": [],
        "dissem_countries": [
            "USA"
        ],
        "sci_ctrls": [],
        "f_clearance": [
            "u"
        ],
        "owner_prod": [],
        "f_regions": [],
        "f_share": [
            "cnaldeaamandadcnaldadoupeopleoudiaoudodou_s_governmentcus",
            "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
            "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus"
        ],
        "portion": "U",
        "disp_only": "",
        "f_sci_ctrls": [],
        "disponly_to": [
            ""
        ],
        "banner": "UNCLASSIFIED",
        "non_ic": [],
        "f_accms": [],
        "f_sar_id": [],
        "f_oc_org": [],
        "classif": "U",
        "atom_energy": [],
        "dissem_ctrls": [],
        "sar_id": [],
        "version": "2.1.0",
        "fgi_protect": [],
        "f_macs": [],
        "f_atom_energy": [],
        "share": {
            "users": [
                "cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us",
                "cn=alleyne shennah cnalles,ou=people,ou=dia,ou=dod,o=u.s. government,c=us",
                "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
            ],
            "projects": {}
        }
    },
    "permissions": [
        {
            "grantee": "cnaldeaamandadcnaldadoupeopleoudiaoudodou_s_governmentcus",
            "userDistinguishedName": "cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us",
            "displayName": "aldea amanda d cnaldad",
            "allowCreate": false,
            "allowRead": true,
            "allowUpdate": false,
            "allowDelete": false,
            "allowShare": false,
            "share": {
                "users": [
                    "cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
                ]
            }
        },
        {
            "allowRead": true,
            "share": {
                "users": [
                    "cn=alleyne shennah cnalles,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
                ]
            }
        },
        {
            "allowCreate": true,
            "allowRead": true,
            "allowUpdate": true,
            "allowDelete": true,
            "allowShare": true,
            "users": [
                "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
            ]
        }
    ],
    "changeToken": "%s"
}`
	objString := fmt.Sprintf(objStringTemplate, folderChild.ID, folderParent.ID, folderChild.ChangeToken)
	objInt, _ := utils.UnmarshalStringToInterface(objString)
	uri := mountPoint + "/objects/" + folderChild.ID + "/properties"
	req := makeHTTPRequestFromInterface(t, "POST", uri, objInt)
	resp, err := clients[tester10].Client.Do(req)
	failNowOnErr(t, err, "unable to do request")
	statusMustBe(t, 200, resp, "expected tester10 to be able to change share")
	var updated protocol.Object
	util.FullDecode(resp.Body, &updated)

}

func TestUpdateObjectWithoutACM(t *testing.T) {
	tester10 := 0
	t.Logf("* Create object as tester10")
	folder := makeFolderViaJSON("TestUpdateObjectWithoutACM", tester10, t)
	t.Logf("* Update object, but dont provide an ACM")
	updateObj := protocol.UpdateObjectRequest{}
	updateObj.ChangeToken = folder.ChangeToken
	updateObj.ContainsUSPersonsData = folder.ContainsUSPersonsData
	updateObj.Description = folder.Description
	updateObj.ExemptFromFOIA = folder.ExemptFromFOIA
	updateObj.ID = folder.ID
	updateObj.Name = folder.Name + " updated"
	updateObj.TypeID = folder.TypeID
	updateObj.TypeName = folder.TypeName
	uri := mountPoint + "/objects/" + folder.ID + "/properties"
	req := makeHTTPRequestFromInterface(t, "POST", uri, updateObj)
	resp, err := clients[tester10].Client.Do(req)
	failNowOnErr(t, err, "unable to do request")
	statusMustBe(t, 200, resp, "expected tester10 to be able to update object")
	var updated protocol.Object
	util.FullDecode(resp.Body, &updated)
}

func TestUpdateObjectWithACMHavingEmptyValueInPart(t *testing.T) {
	tester10 := 0
	t.Logf("* Create object as tester10")
	folder, err := makeFolderWithACMViaJSON("TestUpdateObjectWithACMHavingEmptyValueInPart", ValidACMUnclassifiedEmptyDissemCountries, tester10)
	if err != nil {
		t.Logf("Error creating folder: %v", err)
		t.FailNow()
	}
	t.Logf("* Update object, with an ACM that has an empty dissem countries in the part")
	updateObj := protocol.UpdateObjectRequest{}
	updateObj.ChangeToken = folder.ChangeToken
	updateObj.ContainsUSPersonsData = folder.ContainsUSPersonsData
	updateObj.Description = folder.Description
	updateObj.ExemptFromFOIA = folder.ExemptFromFOIA
	updateObj.ID = folder.ID
	updateObj.Name = folder.Name + " updated"
	updateObj.RawAcm, _ = utils.UnmarshalStringToInterface(ValidACMUnclassifiedEmptyDissemCountriesEmptyFShare)
	updateObj.TypeID = folder.TypeID
	updateObj.TypeName = folder.TypeName
	uri := mountPoint + "/objects/" + folder.ID + "/properties"
	req := makeHTTPRequestFromInterface(t, "POST", uri, updateObj)
	resp, err := clients[tester10].Client.Do(req)
	failNowOnErr(t, err, "unable to do request")
	statusMustBe(t, 200, resp, "expected tester10 to be able to update object")
	var updated protocol.Object
	util.FullDecode(resp.Body, &updated)
}

func TestUpdateObjectHasPermissions(t *testing.T) {
	tester10 := 0
	t.Logf("* Create parent folder as tester10")
	folderParent := makeFolderViaJSON("TestUpdateObjectHasPermissionsParent", tester10, t)
	t.Logf("* Create child folder as tester10")
	folderChild := makeFolderWithParentViaJSON("TestUpdateObjectHasPermissionsChild", folderParent.ID, tester10, t)
	t.Logf("* Change ACM Share settings")

	objStringTemplate := `{
    "id": "%s",
    "parentId": "%s",
    "acm": {
        "f_missions": [],
        "fgi_open": [],
        "rel_to": [],
        "dissem_countries": [
            "USA"
        ],
        "sci_ctrls": [],
        "f_clearance": [
            "u"
        ],
        "owner_prod": [],
        "f_regions": [],
        "f_share": [
            "cnaldeaamandadcnaldadoupeopleoudiaoudodou_s_governmentcus",
            "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
            "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus"
        ],
        "portion": "U",
        "disp_only": "",
        "f_sci_ctrls": [],
        "disponly_to": [
            ""
        ],
        "banner": "UNCLASSIFIED",
        "non_ic": [],
        "f_accms": [],
        "f_sar_id": [],
        "f_oc_org": [],
        "classif": "U",
        "atom_energy": [],
        "dissem_ctrls": [],
        "sar_id": [],
        "version": "2.1.0",
        "fgi_protect": [],
        "f_macs": [],
        "f_atom_energy": [],
        "share": {
            "users": [
                "cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us",
                "cn=alleyne shennah cnalles,ou=people,ou=dia,ou=dod,o=u.s. government,c=us",
                "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
            ],
            "projects": {}
        }
    },
    "permissions": [
        {
            "grantee": "cnaldeaamandadcnaldadoupeopleoudiaoudodou_s_governmentcus",
            "userDistinguishedName": "cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us",
            "displayName": "aldea amanda d cnaldad",
            "allowCreate": false,
            "allowRead": true,
            "allowUpdate": false,
            "allowDelete": false,
            "allowShare": false,
            "share": {
                "users": [
                    "cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
                ]
            }
        },
        {
            "allowRead": true,
            "share": {
                "users": [
                    "cn=alleyne shennah cnalles,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
                ]
            }
        },
        {
            "allowCreate": true,
            "allowRead": true,
            "allowUpdate": true,
            "allowDelete": true,
            "allowShare": true,
            "users": [
                "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
            ]
        }
    ],
    "changeToken": "%s"
}`
	objString := fmt.Sprintf(objStringTemplate, folderChild.ID, folderParent.ID, folderChild.ChangeToken)
	objInt, _ := utils.UnmarshalStringToInterface(objString)
	uri := mountPoint + "/objects/" + folderChild.ID + "/properties"
	req := makeHTTPRequestFromInterface(t, "POST", uri, objInt)
	resp, err := clients[tester10].Client.Do(req)
	failNowOnErr(t, err, "unable to do request")
	statusMustBe(t, 200, resp, "expected tester10 to be able to change share")
	var updated protocol.Object
	util.FullDecode(resp.Body, &updated)

	t.Logf("* Checking caller permissions")
	if !updated.CallerPermission.AllowCreate {
		t.Logf("  missing allowCreate")
		t.Fail()
	}
	if !updated.CallerPermission.AllowRead {
		t.Logf("  missing allowRead")
		t.Fail()
	}
	if !updated.CallerPermission.AllowUpdate {
		t.Logf("  missing allowUpdate")
		t.Fail()
	}
	if !updated.CallerPermission.AllowDelete {
		t.Logf("  missing allowDelete")
		t.Fail()
	}
	if !updated.CallerPermission.AllowShare {
		t.Logf("  missing allowShare")
		t.Fail()
	}
}

func TestUpdateObjectToEveryoneReturnsOwnerCRUDS(t *testing.T) {
	tester10 := 0
	t.Logf("* Create object shared to just me (#98 steps 1-3)")
	folder, err := makeFolderWithACMViaJSON("TestUpdateObjectToEveryoneReturnsOwnerCRUDS", ValidACMUnclassifiedFOUOSharedToTester01, tester10)
	if err != nil {
		t.Logf("Error creating folder: %v", err)
		t.FailNow()
	}

	t.Logf("* Update object with share to everone (#98 steps 4-6)")
	objStringTemplate := `{
    "id": "%s",
    "acm": {"dissem_countries": ["USA"], "f_clearance": ["u"], "portion": "U", "banner": "UNCLASSIFIED", "classif": "U", "version": "2.1.0", "share": {}},
    "changeToken": "%s"
	}`
	objString := fmt.Sprintf(objStringTemplate, folder.ID, folder.ChangeToken)
	objInt, _ := utils.UnmarshalStringToInterface(objString)
	uri := mountPoint + "/objects/" + folder.ID + "/properties"
	req := makeHTTPRequestFromInterface(t, "POST", uri, objInt)
	resp, err := clients[tester10].Client.Do(req)
	failNowOnErr(t, err, "unable to do request")
	statusMustBe(t, 200, resp, "expected tester10 to be able to change share")
	var updated protocol.Object
	util.FullDecode(resp.Body, &updated)

	t.Logf("* Checking caller permissions (#98 step 7)")
	if !updated.CallerPermission.AllowCreate {
		t.Logf("  missing allowCreate")
		t.Fail()
	}
	if !updated.CallerPermission.AllowRead {
		t.Logf("  missing allowRead")
		t.Fail()
	}
	if !updated.CallerPermission.AllowUpdate {
		t.Logf("  missing allowUpdate")
		t.Fail()
	}
	if !updated.CallerPermission.AllowDelete {
		t.Logf("  missing allowDelete")
		t.Fail()
	}
	if !updated.CallerPermission.AllowShare {
		t.Logf("  missing allowShare")
		t.Fail()
	}
}

func TestUpdateObjectWithPermissions(t *testing.T) {
	tester10 := 0
	t.Logf("* Create object shared to just tester10")
	folder, err := makeFolderWithACMViaJSON("TestUpdateObjectWithPermissions", ValidACMUnclassifiedFOUOSharedToTester01, tester10)
	if err != nil {
		t.Logf("Error creating folder: %v", err)
		t.FailNow()
	}

	t.Logf("* Update object with RU for ODrive, and leave CRUDS for owner as implicit")
	objStringTemplate := `{
    "id": "%s",
    "acm": {"dissem_countries": ["USA"], "f_clearance": ["u"], "portion": "U", "banner": "UNCLASSIFIED", "classif": "U", "version": "2.1.0", "share": {"projects":{"dctc":{"disp_nm":"DCTC","groups":["ODrive"]}}}},
	"permission": {"read": {"allow":["group/dctc/DCTC/ODrive/DCTC ODrive"]}, "update":{"allow":["group/dctc/DCTC/ODrive/DCTC ODrive"]}},
    "changeToken": "%s"
	}`
	objString := fmt.Sprintf(objStringTemplate, folder.ID, folder.ChangeToken)
	objInt, _ := utils.UnmarshalStringToInterface(objString)
	uri := mountPoint + "/objects/" + folder.ID + "/properties"
	req := makeHTTPRequestFromInterface(t, "POST", uri, objInt)
	resp, err := clients[tester10].Client.Do(req)
	failNowOnErr(t, err, "unable to do request")
	statusMustBe(t, 200, resp, "expected tester10 to be able to change share")
	var updated protocol.Object
	util.FullDecode(resp.Body, &updated)

	t.Logf("* Checking caller permissions (Ensure owner CRUDS!)")
	if !updated.CallerPermission.AllowCreate {
		t.Logf("  missing allowCreate")
		t.Fail()
	}
	if !updated.CallerPermission.AllowRead {
		t.Logf("  missing allowRead")
		t.Fail()
	}
	if !updated.CallerPermission.AllowUpdate {
		t.Logf("  missing allowUpdate")
		t.Fail()
	}
	if !updated.CallerPermission.AllowDelete {
		t.Logf("  missing allowDelete")
		t.Fail()
	}
	if !updated.CallerPermission.AllowShare {
		t.Logf("  missing allowShare")
		t.Fail()
	}

	t.Logf("* Checking that ODrive group has read and update")
	groupName := "group/dctc/dctc/odrive"
	foundReader := false
	for _, allowedReader := range updated.Permission.Read.AllowedResources {
		t.Logf("allowed reader = %s", allowedReader)
		if allowedReader == groupName {
			foundReader = true
			break
		}
	}
	if !foundReader {
		t.Logf(" Group %s was not found as a reader", groupName)
		t.Fail()
	}
	foundUpdater := false
	for _, allowedUpdater := range updated.Permission.Update.AllowedResources {
		t.Logf("allowed updater = %s", allowedUpdater)
		if allowedUpdater == groupName {
			foundUpdater = true
			break
		}
	}
	if !foundUpdater {
		t.Logf(" Group %s was not found as an updater", groupName)
		t.Fail()
	}
}

func TestUpdateObjectWithPathing(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	t.Logf("* Create folder under root as tester10")
	folder1 := makeFolderViaJSON("Test Folder 1 ", tester10, t)

	t.Logf("* Attempt to rename it")
	defaultPathDelimiter := string(rune(30)) // 20161230 was a slash, 20170301 is now the record separator character code 30
	changedName := strings.Join([]string{"renamed", "with", "pathing"}, defaultPathDelimiter)
	updateuri := mountPoint + "/objects/" + folder1.ID + "/properties"
	folder1.Name = changedName

	updateFolderReq := makeHTTPRequestFromInterface(t, "POST", updateuri, folder1)
	updateFolderRes, err := clients[tester10].Client.Do(updateFolderReq)
	defer util.FinishBody(updateFolderRes.Body)
	failNowOnErr(t, err, "Unable to do request")
	statusExpected(t, 400, updateFolderRes, "Bad status when renaming folder with pathing")
}

func TestUpdateObjectProperty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	t.Logf("* Create folder under root as tester10")
	folder1 := makeFolderViaJSON("TestUpdateObjectProperty ", tester10, t)

	t.Logf("* Add a property to the object")
	folder1.Properties = append(folder1.Properties, protocol.Property{Name: "custom-property", Value: "property value 1"})

	t.Logf("* Update the object")
	updateuri := mountPoint + "/objects/" + folder1.ID + "/properties"
	req := makeHTTPRequestFromInterface(t, "POST", updateuri, folder1)
	trafficLogs[APISampleFile].Request(t, req, &TrafficLogDescription{
		OperationName:       "Update Object With New Property",
		RequestDescription:  "While updating the metadata for an object, include a dynamic property to be set",
		ResponseDescription: "Resulting object includes property",
	})
	res, err := clients[tester10].Client.Do(req)
	defer util.FinishBody(res.Body)
	failNowOnErr(t, err, "Unable to do request")
	statusExpected(t, 200, res, "Bad status when updating object with property")
	trafficLogs[APISampleFile].Response(t, res)
	var updated protocol.Object
	util.FullDecode(res.Body, &updated)

	t.Logf("* Verify updated has the property")
	if len(updated.Properties) != 1 {
		t.Logf("Expected 1 property, but got %d", len(updated.Properties))
		t.FailNow()
	}
	if updated.Properties[0].Name != "custom-property" {
		t.Logf("Expected property name to be 'custom-property' but got %s", updated.Properties[0].Name)
		t.FailNow()
	}
	if updated.Properties[0].Value != "property value 1" {
		t.Logf("Expected property value to be 'property value 1' but got %s", updated.Properties[0].Value)
		t.FailNow()
	}

	t.Logf("* Change value of property")
	updated.Properties[0].Value = "new property value"

	t.Logf("* Update the object")
	req2 := makeHTTPRequestFromInterface(t, "POST", updateuri, updated)
	trafficLogs[APISampleFile].Request(t, req2, &TrafficLogDescription{
		OperationName:       "Update Object Change Property Value",
		RequestDescription:  "While updating the metadata for an object, change value of already existing dynamic property",
		ResponseDescription: "Resulting object shows property with new value",
	})
	res2, err := clients[tester10].Client.Do(req2)
	defer util.FinishBody(res2.Body)
	failNowOnErr(t, err, "Unable to do request")
	statusExpected(t, 200, res2, "Bad status when updating object with new property value")
	trafficLogs[APISampleFile].Response(t, res2)
	var updated2 protocol.Object
	util.FullDecode(res2.Body, &updated2)

	t.Logf("* Verify updated has the property")
	if len(updated2.Properties) != 1 {
		t.Logf("Expected 1 property, but got %d", len(updated2.Properties))
		t.FailNow()
	}
	if updated2.Properties[0].Name != "custom-property" {
		t.Logf("Expected property name to be 'custom-property' but got %s", updated2.Properties[0].Name)
		t.FailNow()
	}
	if updated2.Properties[0].Value != "new property value" {
		t.Logf("Expected property value to be 'new property value' but got %s", updated2.Properties[0].Value)
		t.FailNow()
	}
}

func TestUpdateObjectContentTypeWithoutStream(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	clientid := 0

	// default for objects with content stream = "application/octet-stream"
	// no default is given for those without

	t.Logf("Create 1 folders under root")
	folder := makeFolderViaJSON("Test Folder for Update ", clientid, t)
	initialContentType := folder.ContentType // presumably empty per above since this is a folder

	t.Logf("Attempt to change content type")
	updateuri := mountPoint + "/objects/" + folder.ID + "/properties"
	folder.ContentType = "text/plain" // force it to something other then empty or application/octet-stream default
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		t.Logf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("POST", updateuri, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	t.Logf("do the request")
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)

	t.Logf("Need to parse the body and verify it didnt change")
	var updatedObject protocol.Object
	err = util.FullDecode(res.Body, &updatedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}
	if strings.Compare(updatedObject.ContentType, initialContentType) == 0 {
		t.Logf("Content Type was not changed to %s", initialContentType)
		t.FailNow()
	}
}
