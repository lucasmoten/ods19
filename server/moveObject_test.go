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
	"decipher.com/object-drive-server/util"

	"decipher.com/object-drive-server/protocol"
)

func TestMoveObject(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d", clientid)
		fmt.Println()
	}

	// Create 2 folders under root
	folder1 := makeFolderViaJSON("Test Folder 1 ", clientid, t)
	folder2 := makeFolderViaJSON("Test Folder 2 ", clientid, t)

	// Attempt to move folder 2 under folder 1
	moveuri := host + cfg.NginxRootURL + "/objects/" + folder2.ID + "/move/" + folder1.ID
	objChangeToken := protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = folder2.ChangeToken
	jsonBody, err := json.Marshal(objChangeToken)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("POST", moveuri, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName:       "Move an object",
			RequestDescription:  "Request that an object be moved to a new location",
			ResponseDescription: "The object in its new location",
		},
	)
	// do the request
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	trafficLogs[APISampleFile].Response(t, res)
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

func TestMoveObjectToRoot(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0

	t.Logf("* Create 2 folders under root as tester10")
	folder1 := makeFolderViaJSON("Test Folder 1 ", tester10, t)
	t.Logf("  Folder 1 ID: %s", folder1.ID)
	folder2 := makeFolderViaJSON("Test Folder 2 ", tester10, t)
	t.Logf("  Folder 2 ID: %s", folder2.ID)

	t.Logf("* Move folder 2 under folder 1")
	moveuri := host + cfg.NginxRootURL + "/objects/" + folder2.ID + "/move/" + folder1.ID
	objChangeToken := protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = folder2.ChangeToken
	moveReq1 := makeHTTPRequestFromInterface(t, "POST", moveuri, objChangeToken)
	moveRes1, err := clients[tester10].Client.Do(moveReq1)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, moveRes1, "Bad status when moving folder 2 under folder 1")
	var updatedFolder2a protocol.Object
	err = util.FullDecode(moveRes1.Body, &updatedFolder2a)
	failNowOnErr(t, err, "Error decoding json to Object")
	t.Logf("  Folder 2 Parent ID: %s", updatedFolder2a.ParentID)
	if strings.Compare(updatedFolder2a.ParentID, folder1.ID) != 0 {
		t.Logf("  FAIL: Parent of folder 2 is not folder 1")
		t.FailNow()
	} else {
		t.Logf("  Folder2 is now under Folder1")
	}

	t.Logf("* Move folder 2 back to root")
	moveuriroot := host + cfg.NginxRootURL + "/objects/" + folder2.ID + "/move/"
	objChangeToken.ChangeToken = updatedFolder2a.ChangeToken
	moveReq2 := makeHTTPRequestFromInterface(t, "POST", moveuriroot, objChangeToken)
	moveRes2, err := clients[tester10].Client.Do(moveReq2)
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, moveRes2, "Bad status when moving folder 2 back to root")
	var updatedFolder2b protocol.Object
	err = util.FullDecode(moveRes2.Body, &updatedFolder2b)
	failNowOnErr(t, err, "Error decoding json to Object")
	t.Logf("  Folder 2 Parent ID: %s", updatedFolder2b.ParentID)
	if len(updatedFolder2b.ParentID) != 0 {
		t.Logf("  FAIL: Parent of folder 2 is not root")
		t.FailNow()
	} else {
		t.Logf("  Folder2 is back under root")
	}
}