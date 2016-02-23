package server_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"testing"
	"time"

	"decipher.com/oduploader/protocol"
)

func makeFolderViaJSON(folderName string, clientid int) (*protocol.Object, error) {
	folderuri := host + "/service/metadataconnector/1.0/folder"
	folder := protocol.Object{}
	folder.Name = folderName
	folder.TypeName = "Folder"
	folder.ParentID = ""
	// marshall request
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		return nil, err
	}
	req, err := http.NewRequest("POST", folderuri, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		return nil, err
	}
	// do the request
	req.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{TLSClientConfig: clients[clientid].Config}
	client := &http.Client{Transport: transport}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		return nil, err
	}
	// process Response
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		return nil, errors.New("Status was " + res.Status)
	}
	decoder := json.NewDecoder(res.Body)
	var createdFolder protocol.Object
	err = decoder.Decode(&createdFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		return nil, err
	}
	return &createdFolder, nil
}

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
	folder1, err := makeFolderViaJSON("Test Folder 1 "+strconv.FormatInt(time.Now().Unix(), 10), clientid)
	if err != nil {
		t.Fail()
		return
	}
	folder2, err := makeFolderViaJSON("Test Folder 2 "+strconv.FormatInt(time.Now().Unix(), 10), clientid)
	if err != nil {
		t.Fail()
		return
	}

	// Attempt to move folder 2 under folder 1
	moveuri := host + "/service/metadataconnector/1.0/object/" + folder2.ID + "/move/" + folder1.ID
	objChangeToken := protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = folder2.ChangeToken
	jsonBody, err := json.Marshal(objChangeToken)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.Fail()
		return
	}
	req, err := http.NewRequest("PUT", moveuri, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.Fail()
		return
	}
	// do the request
	transport := &http.Transport{TLSClientConfig: clients[clientid].Config}
	client := &http.Client{Transport: transport}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.Fail()
		return
	}
	// process Response
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.Fail()
		return
	}
	decoder := json.NewDecoder(res.Body)
	var updatedFolder protocol.Object
	err = decoder.Decode(&updatedFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		t.Fail()
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
