package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"testing"

	"decipher.com/oduploader/protocol"
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

	// URL
	uri := host + "/service/metadataconnector/1.0/objects"

	// Body
	paging := protocol.PagingRequest{}
	paging.PageNumber = 1
	paging.PageSize = 2
	jsonBody, err := json.Marshal(paging)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		t.Fail()
	}

	// Request
	req, err := http.NewRequest("GET", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.Fail()
	}
	req.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{TLSClientConfig: clients[clientid].Config}
	client := &http.Client{Transport: transport}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.Fail()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.Fail()
	}
	decoder := json.NewDecoder(res.Body)
	var listOfObjects protocol.ObjectResultset
	err = decoder.Decode(&listOfObjects)
	if err != nil {
		log.Printf("Error decoding json to ObjectResultset: %v", err)
		t.Fail()
	}
	if verboseOutput {
		log.Printf("Total Rows: %d", listOfObjects.TotalRows)

		jsonData, err := json.MarshalIndent(listOfObjects, "", "  ")
		if err != nil {
			log.Printf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		fmt.Println("Here is the response body:")
		fmt.Println(string(jsonData))
	}
}
