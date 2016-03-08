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

	"decipher.com/oduploader/protocol"
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
	updateuri := host + "/service/metadataconnector/1.0/object/" + folder.ID + "/properties"
	folder.Name = "Test Folder Updated " + strconv.FormatInt(time.Now().Unix(), 10)
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("PUT", updateuri, bytes.NewBuffer(jsonBody))
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
	updateuri := host + "/service/metadataconnector/1.0/object/" + folder.ID + "/properties"
	folder.Name = ""
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("PUT", updateuri, bytes.NewBuffer(jsonBody))
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
	if !strings.HasPrefix(updatedFolder.Name, "Unnamed ") {
		log.Printf("Folder name is %s, expected it to be Unnamed Folder", updatedFolder.Name)
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
	updateuri := host + "/service/metadataconnector/1.0/object/" + folder.ID + "/properties"
	folder.OwnedBy = fakeDN2
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.FailNow()
	}
	req, err := http.NewRequest("PUT", updateuri, bytes.NewBuffer(jsonBody))
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
	if res.StatusCode != 428 {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}

}
