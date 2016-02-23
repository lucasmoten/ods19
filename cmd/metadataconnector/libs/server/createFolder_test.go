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

	"decipher.com/oduploader/protocol"
)

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
	uri := host + "/service/metadataconnector/1.0/folder"
	if verboseOutput {
		fmt.Printf("(Verbose Mode) uri: %s\n", uri)
	}

	// Body
	folder := protocol.Object{}
	folder.Name = "Test Folder At Root " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.TypeName = "Folder"
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
	transport := &http.Transport{TLSClientConfig: clients[clientid].Config}
	client := &http.Client{Transport: transport}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	decoder := json.NewDecoder(res.Body)
	var createdFolder protocol.Object
	err = decoder.Decode(&createdFolder)
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
	uri := host + "/service/metadataconnector/1.0/folder"
	if verboseOutput {
		fmt.Printf("(Verbose Mode) uri: %s", uri)
		fmt.Println()
	}

	// Body
	folder := protocol.Object{}
	folder.Name = "Test Folder At Root " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.TypeName = "Folder"
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
	transport := &http.Transport{TLSClientConfig: clients[clientid].Config}
	client := &http.Client{Transport: transport}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	decoder := json.NewDecoder(res.Body)
	var createdFolder protocol.Object
	err = decoder.Decode(&createdFolder)
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
	res, err = client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	decoder = json.NewDecoder(res.Body)
	var createdSubFolder protocol.Object
	err = decoder.Decode(&createdSubFolder)
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
	uri := host + "/service/metadataconnector/1.0/folder"
	if verboseOutput {
		fmt.Printf("(Verbose Mode) uri: %s", uri)
		fmt.Println()
	}

	// Body
	folder := protocol.Object{}
	folder.Name = "Test Folder At Root " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.TypeName = "Folder"
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
	transport1 := &http.Transport{TLSClientConfig: clients[clientid1].Config}
	client1 := &http.Client{Transport: transport1}
	res, err := client1.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	decoder := json.NewDecoder(res.Body)
	var createdFolder protocol.Object
	err = decoder.Decode(&createdFolder)
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
	transport2 := &http.Transport{TLSClientConfig: clients[clientid2].Config}
	client2 := &http.Client{Transport: transport2}
	res, err = client2.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

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
	clientid1 := 0
	transport1 := &http.Transport{TLSClientConfig: clients[clientid1].Config}
	client1 := &http.Client{Transport: transport1}
	clientid2 := 1
	transport2 := &http.Transport{TLSClientConfig: clients[clientid2].Config}
	client2 := &http.Client{Transport: transport2}

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d to create folder in root", clientid1)
		fmt.Printf("(Verbose Mode) Using client id %d to create subfolder", clientid2)
		fmt.Println()
	}

	// URL
	uri := host + "/service/metadataconnector/1.0/folder"
	if verboseOutput {
		fmt.Printf("(Verbose Mode) uri: %s", uri)
		fmt.Println()
	}

	// Body
	folder := protocol.Object{}
	folder.Name = "Test Folder At Root " + strconv.FormatInt(time.Now().Unix(), 10)
	folder.TypeName = "Folder"
	folder.ParentID = ""
	grant2client2 := protocol.Permission{}
	grant2client2.Grantee = "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US"
	grant2client2.AllowRead = true
	grant2client2.AllowCreate = true
	folder.Permissions = append(folder.Permissions, grant2client2)
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
	res, err := client1.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	decoder := json.NewDecoder(res.Body)
	var createdFolder protocol.Object
	err = decoder.Decode(&createdFolder)
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
	res, err = client2.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("The second user was not allowed to create a folder but they should have permissions granted")
		t.FailNow()
	}
}
