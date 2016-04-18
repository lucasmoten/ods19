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
	folder, err := makeFolderViaJSON("Test Folder for Update "+strconv.FormatInt(time.Now().Unix(), 10), clientid)
	if err != nil {
		t.FailNow()
	}

	// Attempt to rename the folder
	updateuri := host + cfg.RootURL + "/objects/" + folder.ID + "/properties"
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
	decoder := json.NewDecoder(res.Body)
	var updatedFolder protocol.Object
	err = decoder.Decode(&updatedFolder)
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
	folder, err := makeFolderViaJSON("Test Folder for Update "+strconv.FormatInt(time.Now().Unix(), 10), clientid)
	if err != nil {
		t.FailNow()
	}

	// Attempt to rename the folder
	updateObjectRequest := protocol.UpdateObjectRequest{}
	updateObjectRequest.Name = ""
	updateObjectRequest.ChangeToken = folder.ChangeToken
	updateuri := host + cfg.RootURL + "/objects/" + folder.ID + "/properties"
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
	decoder := json.NewDecoder(res.Body)
	var updatedFolder protocol.Object
	err = decoder.Decode(&updatedFolder)
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
	folder, err := makeFolderViaJSON("Test Folder for Update "+strconv.FormatInt(time.Now().Unix(), 10), clientid)
	if err != nil {
		t.FailNow()
	}

	// Attempt to change owner
	updateuri := host + cfg.RootURL + "/objects/" + folder.ID + "/properties"
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
	res, err := httpclients[clientid].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	// // process Response
	// if res.StatusCode != 428 {
	// 	log.Printf("bad status: %s", res.Status)
	// 	t.FailNow()
	// }

	// Need to parse the body and verify it didnt change
	decoder := json.NewDecoder(res.Body)
	var updatedObject protocol.Object
	err = decoder.Decode(&updatedObject)
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
