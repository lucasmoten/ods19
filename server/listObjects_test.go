package server_test

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"testing"

	"github.com/deciphernow/object-drive-server/util"

	"github.com/deciphernow/object-drive-server/protocol"
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
	uri := mountPoint + "/objects"
	uri1 := uri + "?PageNumber=1&PageSize=2"

	// Request
	req, err := http.NewRequest("GET", uri1, nil)
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	trafficLogs[APISampleFile].Request(t, req,
		&TrafficLogDescription{
			OperationName:       "Get a root object listing",
			RequestDescription:  "Send a response for a paged listing",
			ResponseDescription: "We get back a page from the listing",
			ResponseBodyHide:    true, //this might be kind of big due to test re-runs
		},
	)

	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	var listOfObjects protocol.ObjectResultset
	err = util.FullDecode(res.Body, &listOfObjects)
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
	uri := mountPoint + "/objects"
	uri1 := uri + "?PageNumber=1&PageSize=1"

	// Request
	req, err := http.NewRequest("GET", uri1, nil)
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
	if verboseOutput {
		log.Printf("Status: %s", res.Status)
	}
	var listOfObjects protocol.ObjectResultset
	err = util.FullDecode(res.Body, &listOfObjects)
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
		if pn >= 3 {
			return
		}
		uriPaged := uri + "?PageNumber=" + strconv.Itoa(pn) + "&PageSize=20"
		// Request
		req, err := http.NewRequest("GET", uriPaged, nil)
		if err != nil {
			log.Printf("Error setting up HTTP Request: %v", err)
			t.FailNow()
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := clients[clientid].Client.Do(req)
		if err != nil {
			log.Printf("Unable to do request:%v", err)
			t.FailNow()
		}
		defer util.FinishBody(res.Body)
		// Response validation
		if res.StatusCode != http.StatusOK {
			log.Printf("bad status: %s", res.Status)
			t.FailNow()
		}
		if verboseOutput {
			log.Printf("Status: %s", res.Status)
		}
		var listOfObjects protocol.ObjectResultset
		err = util.FullDecode(res.Body, &listOfObjects)
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
		t.Logf("(Verbose Mode) Using client id %d", clientid)
	}

	// URLs
	uri := mountPoint + "/objects?PageSize="
	if testing.Short() {
		uri += "20"
	} else {
		uri += "1000"
	}
	uri += "&PageNumber="
	uri1 := uri + "1"

	t.Logf("Request")
	req, err := http.NewRequest("GET", uri1, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	t.Logf("Response validation")
	if res.StatusCode != http.StatusOK {
		t.Logf("bad status: %s", res.Status)
		t.FailNow()
	}
	t.Logf("Decoding body to resultset")
	var listOfObjects protocol.ObjectResultset
	err = util.FullDecode(res.Body, &listOfObjects)
	if err != nil {
		t.Logf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}
	t.Logf("Depth display in child tree")
	level := 0
	depthstring := "+-"
	for _, obj := range listOfObjects.Objects {
		if verboseOutput {
			t.Logf("%s%s", depthstring, obj.Name)
		}
		childlevel := level + 1
		showChildTree(t, verboseOutput, clients[clientid].Client, childlevel, obj.ID)
		if t.Failed() {
			return
		}
	}
	t.Logf("Paging")
	for pn := 2; pn <= listOfObjects.PageCount; pn++ {
		if pn >= 3 {
			return
		}
		uriPaged := uri + strconv.Itoa(pn)
		// Request
		req, err := http.NewRequest("GET", uriPaged, nil)
		if err != nil {
			t.Logf("Error setting up HTTP Request: %v", err)
			t.FailNow()
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := clients[clientid].Client.Do(req)
		if err != nil {
			t.Logf("Unable to do request:%v", err)
			t.FailNow()
		}
		defer util.FinishBody(res.Body)
		// Response validation
		if res.StatusCode != http.StatusOK {
			t.Logf("bad status: %s", res.Status)
			t.FailNow()
		}
		var listOfObjects protocol.ObjectResultset
		err = util.FullDecode(res.Body, &listOfObjects)
		if err != nil {
			t.Logf("Error decoding json to ObjectResultset: %v", err)
			t.FailNow()
		}
		for _, obj := range listOfObjects.Objects {
			if verboseOutput {
				t.Logf("%s%s", depthstring, obj.Name)
			}
			childlevel := level + 1
			showChildTree(t, verboseOutput, clients[clientid].Client, childlevel, obj.ID)
			if t.Failed() {
				return
			}
		}
	}
}

func showChildTree(t *testing.T, verboseOutput bool, client *http.Client, level int, childid string) {
	// URLs
	uri := mountPoint + "/objects/" + childid + "?PageSize="
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
	if level > 3 {
		return
	}

	// Request
	req, err := http.NewRequest("GET", uri1, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// Response validation
	if res.StatusCode != http.StatusOK {
		t.Logf("Status code expected was 200 but received %d %s %s", res.StatusCode, res.Status, childid)
		t.Logf(depthstring)
		t.Logf(" >>> 403 Unauthorized to read this object, so cannot list children")
		return
	}
	var listOfObjects protocol.ObjectResultset
	err = util.FullDecode(res.Body, &listOfObjects)
	if err != nil {
		t.Logf("Error decoding json to ObjectResultset: %v", err)
		t.FailNow()
	}

	depthstring += "+-"
	for _, obj := range listOfObjects.Objects {
		if verboseOutput {
			t.Logf("%s%s", depthstring, obj.Name)
		}
		childlevel := level + 1
		showChildTree(t, verboseOutput, client, childlevel, obj.ID)
		if t.Failed() {
			return
		}
	}
	for pn := 2; pn <= listOfObjects.PageCount; pn++ {
		if pn >= 3 {
			return
		}
		uriPaged := uri + strconv.Itoa(pn)
		// Request
		req, err := http.NewRequest("GET", uriPaged, nil)
		if err != nil {
			t.Logf("Error setting up HTTP Request: %v", err)
			t.FailNow()
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			t.Logf("Unable to do request:%v", err)
			t.FailNow()
		}
		defer util.FinishBody(res.Body)
		// Response validation
		if res.StatusCode != http.StatusOK {
			t.Logf("bad status: %s", res.Status)
			t.Fail()
			return
		}
		var listOfObjects protocol.ObjectResultset
		err = util.FullDecode(res.Body, &listOfObjects)
		if err != nil {
			t.Logf("Error decoding json to ObjectResultset: %v", err)
			t.FailNow()
		}
		for _, obj := range listOfObjects.Objects {
			if verboseOutput {
				t.Logf("%s,%s", depthstring, obj.Name)
			}
			childlevel := level + 1
			showChildTree(t, verboseOutput, client, childlevel, obj.ID)
			if t.Failed() {
				return
			}
		}
	}
}

func TestListObjectsWithInvalidFilterField(t *testing.T) {
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
	uri := mountPoint + "/objects"
	uri1 := uri + "?PageNumber=1&PageSize=1&filterField=NON_EXISTENT_FIELD&condition=equals&expression=x"

	// Request
	req, err := http.NewRequest("GET", uri1, nil)
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
}

func TestListObjectsWithInvalidFilterCondition(t *testing.T) {
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
	uri := mountPoint + "/objects"
	uri1 := uri + "?PageNumber=1&PageSize=1&filterField=NON_EXISTENT_FIELD&condition=INVALID_CONDITION&expression=x"

	// Request
	req, err := http.NewRequest("GET", uri1, nil)
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// Response validation
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
}

func TestListObjectsForNonExistentUser(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	clientid := 10
	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	nonexistentuser := "cn=bob smith,ou=fake,ou=dia,o=u.s. government,c=us"

	// URL
	uri := mountPoint + "/objects"

	// Request
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	req.Header.Add("USER_DN", nonexistentuser)
	req.Header.Add("SSL_CLIENT_S_DN", whitelistedDN)
	req.Header.Add("EXTERNAL_SYS_DN", whitelistedDN)

	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		t.FailNow()
	}
	defer util.FinishBody(res.Body)
	// Response validation
	if res.StatusCode != http.StatusForbidden {
		log.Printf("bad status: %s", res.Status)
		t.FailNow()
	}
}
