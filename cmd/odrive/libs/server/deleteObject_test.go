package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"testing"
	"time"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/util"

	"decipher.com/object-drive-server/protocol"
)

func TestDeleteObject(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d", clientid)
		fmt.Println()
	}

	// Create folder under root
	folder1, err := makeFolderViaJSON("Test Folder for Deletion "+strconv.FormatInt(time.Now().Unix(), 10), clientid)
	if err != nil {
		log.Printf("Error making folder 1: %v", err)
		t.FailNow()
	}

	// Now delete the first folder
	deleteuri := host + cfg.NginxRootURL + "/objects/" + folder1.ID + "/trash"
	objChangeToken := protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = folder1.ChangeToken
	jsonBody, err := json.Marshal(objChangeToken)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("POST", deleteuri, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	res, err := httpclients[clientid].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	// process Response
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	var deletedFolder protocol.DeletedObjectResponse
	err = util.FullDecode(res.Body, deletedFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	// if verboseOutput {
	// 	jsonData, err := json.MarshalIndent(deletedFolder, "", "  ")
	// 	if err != nil {
	// 		log.Printf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
	// 		return
	// 	}
	// 	fmt.Println("Here is the json object:")
	// 	fmt.Println(string(jsonData))
	// }

	// now make sure the item is marked as deleted when calling for properties
	geturi := host + cfg.NginxRootURL + "/objects/" + folder1.ID + "/properties"
	req, err = http.NewRequest("GET", geturi, nil)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	res, err = httpclients[clientid].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	// process Response
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	var getResponse protocol.DeletedObject
	err = util.FullDecode(res.Body, &getResponse)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	// Verify that it has deletedDate and deletedBy
	if len(getResponse.DeletedBy) == 0 {
		log.Printf("Deleted by is not set")
		t.FailNow()
	}

}

func TestDeleteWithChildObject(t *testing.T) {
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
	folder1, err := makeFolderViaJSON("Test Folder 1 "+strconv.FormatInt(time.Now().Unix(), 10), clientid)
	if err != nil {
		log.Printf("Error making folder 1: %v", err)
		t.FailNow()
	}
	folder2, err := makeFolderViaJSON("Test Folder 2 "+strconv.FormatInt(time.Now().Unix(), 10), clientid)
	if err != nil {
		log.Printf("Error making folder 2: %v", err)
		t.FailNow()
	}

	// Attempt to move folder 2 under folder 1
	moveuri := host + cfg.NginxRootURL + "/objects/" + folder2.ID + "/move/" + folder1.ID
	objChangeToken := protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = folder2.ChangeToken
	jsonBody, err := json.Marshal(objChangeToken)
	if err != nil {
		log.Printf("moving folder Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("POST", moveuri, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("moving folderError setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	res, err := httpclients[clientid].Do(req)
	if err != nil {
		log.Printf("moving folder Unable to do request:%v", err)
		t.FailNow()
	}
	// process Response
	if res.StatusCode != http.StatusOK {
		log.Printf("moving folder bad status: %s", res.Status)
		t.FailNow()
	}
	var updatedFolder protocol.Object
	err = util.FullDecode(res.Body, &updatedFolder)
	if err != nil {
		log.Printf("moving folder Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	// if verboseOutput {
	// 	jsonData, err := json.MarshalIndent(updatedFolder, "", "  ")
	// 	if err != nil {
	// 		log.Printf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
	// 		return
	// 	}
	// 	fmt.Println("Here is the json object:")
	// 	fmt.Println(string(jsonData))
	// }

	// Now delete the first folder
	deleteuri := host + cfg.NginxRootURL + "/objects/" + folder1.ID + "/trash"
	objChangeToken = protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = folder1.ChangeToken
	jsonBody, err = json.Marshal(objChangeToken)
	if err != nil {
		log.Printf("deleting folder Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err = http.NewRequest("POST", deleteuri, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("deleting folder Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	res, err = httpclients[clientid].Do(req)
	if err != nil {
		log.Printf("deleting folder Unable to do request:%v", err)
		t.FailNow()
	}
	// process Response
	if res.StatusCode != http.StatusOK {
		log.Printf("deleting folder bad status: %s", res.Status)
		t.FailNow()
	}
	var deletedFolder protocol.Object
	err = util.FullDecode(res.Body, &deletedFolder)
	if err != nil {
		log.Printf("deleting folder Error decoding json to Object: %v", err)
		log.Println()
		t.FailNow()
	}
	// if verboseOutput {
	// 	jsonData, err := json.MarshalIndent(deletedFolder, "", "  ")
	// 	if err != nil {
	// 		log.Printf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
	// 		return
	// 	}
	// 	fmt.Println("Here is the response body:")
	// 	fmt.Println(string(jsonData))
	// }

	// Make sure we can't get folder2 anymore (because its a child of a deleted item)
	geturi := host + cfg.NginxRootURL + "/objects/" + folder2.ID + "/properties"
	req, err = http.NewRequest("GET", geturi, nil)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	// do the request
	res, err = httpclients[clientid].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	// process Response
	if res.StatusCode == http.StatusOK {
		log.Printf("able to get folder2 when its parent is deleted")
		t.FailNow()
	}

}
