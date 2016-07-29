package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"

	"decipher.com/object-drive-server/protocol"
)

func TestUpdateObject(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d", clientid)
		fmt.Println()
	}

	// Create 1 folders under root
	folder := makeFolderViaJSON("Test Folder for Update ", clientid, t)

	// Attempt to rename the folder
	updateuri := host + cfg.NginxRootURL + "/objects/" + folder.ID + "/properties"
	folder.Name = "Test Folder Updated " + strconv.FormatInt(time.Now().Unix(), 10)
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("POST", updateuri, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
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
	var updatedFolder protocol.Object
	err = util.FullDecode(res.Body, &updatedFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	if verboseOutput {
		jsonData, err := json.MarshalIndent(updatedFolder, "", "  ")
		if err != nil {
			log.Printf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		fmt.Println("Here is the response body:")
		fmt.Println(string(jsonData))
	}

}

func TestUpdateObjectToHaveNoName(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d", clientid)
		fmt.Println()
	}

	// Create 1 folders under root
	folder := makeFolderViaJSON("Test Folder for Update ", clientid, t)

	// Attempt to rename the folder
	updateObjectRequest := protocol.UpdateObjectRequest{}
	updateObjectRequest.ID = folder.ID
	updateObjectRequest.Name = ""
	updateObjectRequest.ChangeToken = folder.ChangeToken
	updateuri := host + cfg.NginxRootURL + "/objects/" + folder.ID + "/properties"
	jsonBody, err := json.Marshal(updateObjectRequest)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("POST", updateuri, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
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
	var updatedFolder protocol.Object
	err = util.FullDecode(res.Body, &updatedFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	if strings.Compare(updatedFolder.Name, folder.Name) != 0 {
		log.Printf("Folder name is %s, expected it to be %s", updatedFolder.Name, folder.Name)
		t.FailNow()
	}
	if verboseOutput {
		jsonData, err := json.MarshalIndent(updatedFolder, "", "  ")
		if err != nil {
			log.Printf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		fmt.Println("Here is the response body:")
		fmt.Println(string(jsonData))
	}
}

func TestUpdateObjectToChangeOwnedBy(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d", clientid)
		fmt.Println()
	}

	// Create 1 folders under root
	folder := makeFolderViaJSON("Test Folder for Update ", clientid, t)

	// Attempt to change owner
	updateuri := host + cfg.NginxRootURL + "/objects/" + folder.ID + "/properties"
	folder.OwnedBy = fakeDN2
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("POST", updateuri, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// // process Response
	// if res.StatusCode != 428 {
	// 	log.Printf("bad status: %s", res.Status)
	// 	t.FailNow()
	// }

	// Need to parse the body and verify it didnt change
	var updatedObject protocol.Object
	err = util.FullDecode(res.Body, &updatedObject)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	if strings.Compare(updatedObject.OwnedBy, folder.OwnedBy) == 0 {
		log.Printf("Owner was changed to %s", updatedObject.OwnedBy)
		t.FailNow()
	}
	if strings.Compare(updatedObject.OwnedBy, folder.CreatedBy) != 0 {
		log.Printf("Owner is not %s", folder.CreatedBy)
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

	updateuri := host + cfg.NginxRootURL + "/objects/" + folder.ID + "/properties"

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
	updatedFolder.RawAcm = testhelpers.ValidACMUnclassifiedFOUO
	updateReq2 := makeHTTPRequestFromInterface(t, "POST", updateuri, updatedFolder)
	updateRes2, err := clients[tester2].Client.Do(updateReq2)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, updateRes2, "Bad status when updating object")
	err = util.FullDecode(updateRes2.Body, &updatedFolder)
	failNowOnErr(t, err, "Error decoding json to Object")

	t.Logf("* Tester02 update name + acm with a different share. Expect error")
	updatedFolder.Name += " and share"
	updatedFolder.RawAcm = testhelpers.ValidACMUnclassifiedFOUOSharedToTester01And02
	updateReq3 := makeHTTPRequestFromInterface(t, "POST", updateuri, updatedFolder)
	updateRes3, err := clients[tester2].Client.Do(updateReq3)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 403, updateRes3, "Bad status when updating object")
	ioutil.ReadAll(updateRes3.Body)
	updateRes3.Body.Close()

	t.Logf("* Tester01 Add a share allowing Tester02 to share")
	shareSetting2 := protocol.ObjectShare{}
	shareSetting2.Share = makeUserShare(fakeDN2)
	shareSetting2.AllowShare = true
	updatedFolder = doAddObjectShare(t, updatedFolder, &shareSetting2, tester1)

	t.Logf("* Tester02 update name + acm with a different share. Expect success")
	updatedFolder.Name += " and share"
	updatedFolder.RawAcm = testhelpers.ValidACMUnclassifiedFOUOSharedToTester01And02
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
	updateuri := host + cfg.NginxRootURL + "/objects/" + folder1.ID + "/properties"
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

		geturi := host + cfg.NginxRootURL + "/objects/" + folder2.ID + "/properties"
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
	updateuri := host + cfg.NginxRootURL + "/objects/" + folder1.ID + "/properties"
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
