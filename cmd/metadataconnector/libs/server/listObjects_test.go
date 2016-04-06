package server_test

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"testing"

	cfg "decipher.com/oduploader/config"

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
	uri := host + cfg.RootURL + "/objects"
	uri1 := uri + "?PageNumber=1&PageSize=2"

	// Request
	req, err := http.NewRequest("GET", uri1, nil)
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpclients[clientid].Do(req)
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
	var listOfObjects protocol.ObjectResultset
	err = decoder.Decode(&listOfObjects)
	if err != nil {
		log.Printf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
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
	uri := host + cfg.RootURL + "/objects"
	uri1 := uri + "?PageNumber=1&PageSize=1"

	// Request
	req, err := http.NewRequest("GET", uri1, nil)
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpclients[clientid].Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	if verboseOutput {
		log.Printf("Status: %s", res.Status)
	}
	decoder := json.NewDecoder(res.Body)
	var listOfObjects protocol.ObjectResultset
	err = decoder.Decode(&listOfObjects)
	if err != nil {
		log.Printf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
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
		uriPaged := uri + "?PageNumber=" + strconv.Itoa(pn) + "&PageSize=2"
		// Request
		req, err := http.NewRequest("GET", uriPaged, nil)
		if err != nil {
			log.Printf("Error setting up HTTP Request: %v", err)
			t.FailNow()
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := httpclients[clientid].Do(req)
		if err != nil {
			log.Printf("Unable to do request:%v", err)
			t.FailNow()
		}
		// Response validation
		if res.StatusCode != http.StatusOK {
			log.Printf("bad status: %s", res.Status)
			t.FailNow()
		}
		if verboseOutput {
			log.Printf("Status: %s", res.Status)
		}
		decoder := json.NewDecoder(res.Body)
		var listOfObjects protocol.ObjectResultset
		err = decoder.Decode(&listOfObjects)
		if err != nil {
			log.Printf("Error decoding json to ObjectResultset: %v", err)
			t.FailNow()
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
	uri := host + cfg.RootURL + "/objects?PageSize="
	if testing.Short() {
		uri += "20"
	} else {
		uri += "1000"
	}
	uri += "&PageNumber="
	uri1 := uri + "1"

	// Request
	req, err := http.NewRequest("GET", uri1, nil)
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpclients[clientid].Do(req)
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
	var listOfObjects protocol.ObjectResultset
	err = decoder.Decode(&listOfObjects)
	if err != nil {
		log.Printf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}

	level := 0
	depthstring := "+-"
	for _, obj := range listOfObjects.Objects {
		if verboseOutput {
			fmt.Printf(depthstring)
			fmt.Printf(obj.Name)
			fmt.Println()
		}
		childlevel := level + 1
		showChildTree(t, verboseOutput, httpclients[clientid], childlevel, obj.ID)
		if t.Failed() {
			return
		}
	}
	for pn := 2; pn <= listOfObjects.PageCount; pn++ {
		if testing.Short() && pn >= 3 {
			return
		}
		uriPaged := uri + strconv.Itoa(pn)
		// Request
		req, err := http.NewRequest("GET", uriPaged, nil)
		if err != nil {
			log.Printf("Error setting up HTTP Request: %v", err)
			t.FailNow()
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := httpclients[clientid].Do(req)
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
		var listOfObjects protocol.ObjectResultset
		err = decoder.Decode(&listOfObjects)
		if err != nil {
			log.Printf("Error decoding json to ObjectResultset: %v", err)
			t.FailNow()
		}
		for _, obj := range listOfObjects.Objects {
			if verboseOutput {
				fmt.Printf(depthstring)
				fmt.Printf(obj.Name)
				fmt.Println()
			}
			childlevel := level + 1
			showChildTree(t, verboseOutput, httpclients[clientid], childlevel, obj.ID)
			if t.Failed() {
				return
			}
		}
	}
}

func showChildTree(t *testing.T, verboseOutput bool, client *http.Client, level int, childid string) {
	// URLs
	uri := host + cfg.RootURL + "/objects/" + childid + "?PageSize="
	if testing.Short() {
		uri += "20"
	} else {
		uri += "1000"
	}
	uri += "&PageNumber="
	uri1 := uri + "1"

	depthstring := ""
	if level > 0 {
		for l := 0; l < level; l++ {
			depthstring += "| "
		}
	}

	// Request
	req, err := http.NewRequest("GET", uri1, nil)
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}

	// Response validation
	if res.StatusCode != http.StatusOK {
		//log.Printf("bad status: %s %s", res.Status, hex.EncodeToString(childid))
		fmt.Printf(depthstring)
		fmt.Printf(" >>> 403 Unauthorized to read this object, so cannot list children")
		fmt.Println()
		t.FailNow()
	}
	decoder := json.NewDecoder(res.Body)
	var listOfObjects protocol.ObjectResultset
	err = decoder.Decode(&listOfObjects)
	if err != nil {
		log.Printf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}

	depthstring += "+-"
	for _, obj := range listOfObjects.Objects {
		if verboseOutput {
			fmt.Printf(depthstring)
			fmt.Printf(obj.Name)
			fmt.Println()
		}
		childlevel := level + 1
		showChildTree(t, verboseOutput, client, childlevel, obj.ID)
		if t.Failed() {
			return
		}
	}
	for pn := 2; pn <= listOfObjects.PageCount; pn++ {
		if testing.Short() && pn >= 3 {
			return
		}
		uriPaged := uri + strconv.Itoa(pn)
		// Request
		req, err := http.NewRequest("GET", uriPaged, nil)
		if err != nil {
			log.Printf("Error setting up HTTP Request: %v", err)
			t.FailNow()
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			log.Printf("Unable to do request:%v", err)
			t.FailNow()
		}
		// Response validation
		if res.StatusCode != http.StatusOK {
			log.Printf("bad status: %s", res.Status)
			t.Fail()
			return
		}
		decoder := json.NewDecoder(res.Body)
		var listOfObjects protocol.ObjectResultset
		err = decoder.Decode(&listOfObjects)
		if err != nil {
			log.Printf("Error decoding json to ObjectResultset: %v", err)
			t.FailNow()
		}
		for _, obj := range listOfObjects.Objects {
			if verboseOutput {
				fmt.Printf(depthstring)
				fmt.Printf(obj.Name)
				fmt.Println()
			}
			childlevel := level + 1
			showChildTree(t, verboseOutput, client, childlevel, obj.ID)
			if t.Failed() {
				return
			}
		}
	}
}
