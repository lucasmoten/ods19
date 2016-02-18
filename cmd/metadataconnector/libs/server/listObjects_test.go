package server_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"

	"decipher.com/oduploader/protocol"
)

func TestListObjectsRoot(t *testing.T) {
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

func TestListObjectsRootPaging(t *testing.T) {
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
	if verboseOutput {
		log.Printf("Status: %s", res.Status)
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
		log.Printf("Page Count: %d", listOfObjects.PageCount)
		log.Printf("Page Size: %d", listOfObjects.PageSize)
		jsonData, err := json.MarshalIndent(listOfObjects, "", "  ")
		if err != nil {
			log.Printf("(Error in Verbose Mode) Error marshalling response back to json: %s", err.Error())
			return
		}
		fmt.Println("Here is the response body:")
		fmt.Println(string(jsonData))
	}

	for pn := 1; pn <= listOfObjects.PageCount; pn++ {
		paging.PageNumber = pn
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
		if verboseOutput {
			log.Printf("Status: %s", res.Status)
		}
		decoder := json.NewDecoder(res.Body)
		var listOfObjects protocol.ObjectResultset
		err = decoder.Decode(&listOfObjects)
		if err != nil {
			log.Printf("Error decoding json to ObjectResultset: %v", err)
			t.Fail()
		}
		if verboseOutput {
			log.Printf("Page %d: size %d, rows %d", listOfObjects.PageNumber, listOfObjects.PageSize, listOfObjects.PageRows)
			for _, obj := range listOfObjects.Objects {
				log.Printf("- object.name: %s", obj.Name)
			}
		}
	}
}

func TestListObjectsChild(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid := 0

	if verboseOutput {
		fmt.Printf("(Verbose Mode) Using client id %d", clientid)
		fmt.Println()
	}

	// URLs
	uri := host + "/service/metadataconnector/1.0/objects"

	// Body
	paging := protocol.PagingRequest{}
	paging.PageNumber = 1
	paging.PageSize = 1000
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

	level := 0
	depthstring := ""
	if level > 0 {
		depthstring = strings.Repeat("..", level)
	}
	for _, obj := range listOfObjects.Objects {
		if verboseOutput {
			fmt.Printf("%s  %s\n", depthstring, obj.Name)
		}
		childlevel := level + 1
		showChildTree(t, verboseOutput, client, childlevel, obj.ID)
	}
	for pn := 2; pn <= listOfObjects.PageCount; pn++ {
		paging.PageNumber = pn
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
		for _, obj := range listOfObjects.Objects {
			if verboseOutput {
				fmt.Printf("%s  %s\n", depthstring, obj.Name)
			}
			childlevel := level + 1
			showChildTree(t, verboseOutput, client, childlevel, obj.ID)
		}
	}
}

func showChildTree(t *testing.T, verboseOutput bool, client *http.Client, level int, childid []byte) {
	// URLs
	uri := host + "/service/metadataconnector/1.0/object/" + hex.EncodeToString(childid) + "/list"

	// Body
	paging := protocol.PagingRequest{}
	paging.PageNumber = 1
	paging.PageSize = 1000
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

	depthstring := ""
	if level > 0 {
		depthstring = strings.Repeat("..", level)
	}
	for _, obj := range listOfObjects.Objects {
		if verboseOutput {
			fmt.Printf("%s  %s\n", depthstring, obj.Name)
		}
		childlevel := level + 1
		showChildTree(t, verboseOutput, client, childlevel, obj.ID)
	}
	for pn := 2; pn <= listOfObjects.PageCount; pn++ {
		paging.PageNumber = pn
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
		for _, obj := range listOfObjects.Objects {
			if verboseOutput {
				fmt.Printf("%s  %s\n", depthstring, obj.Name)
			}
			childlevel := level + 1
			showChildTree(t, verboseOutput, client, childlevel, obj.ID)
		}
	}
}
