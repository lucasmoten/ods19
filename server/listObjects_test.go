package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"testing"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/client"
	"bitbucket.di2e.net/dime/object-drive-server/util"

	"bitbucket.di2e.net/dime/object-drive-server/protocol"
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
			OperationName:       "List Root Objects for User",
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
	maxtest := 50
	for ix, obj := range listOfObjects.Objects {
		if maxtest > 0 && ix > maxtest {
			break
		}
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

func TestListObjectsWithOCUSGOV(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tester10 := 0
	DN4TP := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	theFilename := "jira-DIMEODS-1183-" + DN4TP + ".png"
	method := "POST"
	uri := mountPoint + "/objects"
	jiraDIMEODS1183 := `
------WebKitFormBoundaryJcb70Da1bmhPiYzE
Content-Disposition: form-data; name="ObjectMetadata"

{"content":{"ext":"png"},"type":"image/png","file":{},"acm":{"fgi_open":[],"rel_to":[],"dissem_countries":["USA"],"sci_ctrls":["HCS-P","SI-G"],"owner_prod":["USA"],"portion":"TS//HCS-P/SI-G//OC-USGOV/NF","disp_only":"","disponly_to":[],"banner":"TOP SECRET//HCS-P/SI-G//ORCON-USGOV/NOFORN","non_ic":[],"classif":"TS","atom_energy":[],"dissem_ctrls":["OC-USGOV","NF"],"sar_id":[],"version":"2.1.0","fgi_protect":[],"share":{"users":["cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]}},"user_dn":"cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us","permission":{"create":{"allow":["user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"delete":{"allow":["user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"read":{"allow":["user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"share":{"allow":["user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"update":{"allow":["user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]}},"name":"` + theFilename + `"}
------WebKitFormBoundaryJcb70Da1bmhPiYzE
Content-Disposition: form-data; name="filestream"; filename="` + theFilename + `"
Content-Type: image/png


------WebKitFormBoundaryJcb70Da1bmhPiYzE--
	
	`
	t.Logf(`* Attempt to upload file with OC-USGOV having name ` + theFilename)
	var requestBuffer *bytes.Buffer
	requestBuffer = bytes.NewBufferString(jiraDIMEODS1183)
	req, err := http.NewRequest(method, uri, requestBuffer)
	if err != nil {
		t.Logf("Error setting up HTTP request: %v", err)
		t.FailNow()
	}
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----WebKitFormBoundaryJcb70Da1bmhPiYzE")
	createObjectRes, err := clients[tester10].Client.Do(req)
	defer util.FinishBody(createObjectRes.Body)
	t.Logf("* Processing Response")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, createObjectRes, "Bad status when creating object")
	data, _ := ioutil.ReadAll(createObjectRes.Body)
	t.Logf("* Length of data in response is %d", len(data))

	t.Logf("* Listing objects with the name " + theFilename)
	method = "GET"
	uri = uri + "?filterMatchType=and&filterField=name&condition=equals&expression=" + theFilename
	reqList, err := http.NewRequest(method, uri, nil)
	if err != nil {
		t.Logf("Error setting up HTTP request to list objects: %v", err)
		t.FailNow()
	}
	listObjectRes, err := clients[tester10].Client.Do(reqList)
	defer util.FinishBody(listObjectRes.Body)
	t.Logf("* Processing Response from List")
	failNowOnErr(t, err, "Unable to do request")
	statusMustBe(t, 200, listObjectRes, "Bad status when listing objects")
	dataList, _ := ioutil.ReadAll(listObjectRes.Body)
	//t.Logf("%s", dataList)
	var ret protocol.ObjectResultset
	jsonErr := json.Unmarshal(dataList, &ret)
	if jsonErr != nil {
		t.Logf("Error unmarshalling body to json object")
		t.FailNow()
	}
	if ret.TotalRows != 1 {
		t.Logf("Unexpected number of results returned. Got %d expected 1", ret.TotalRows)
		t.FailNow()
	}
	if ret.Objects[0].Name == theFilename {
		t.Logf("Object found")
	} else {
		t.Logf("Object returned in list didn't match expected name. Got '%s', expected '%s'", ret.Objects[0].Name, theFilename)
	}
}

func TestListObjectsOfFolder1000(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	tester10 := 0
	acm := `{"version":"2.1.0","classif":"u","dissem_countries":["USA"]}`
	maxobjects := 1000

	// 1. create a folder
	folderResponse, err := clients[tester10].C.CreateObject(client.CreateObjectRequest{Name: "TestListObjectsOfFolder1000", TypeName: "Folder", RawAcm: acm}, nil)
	if err != nil {
		t.Logf("Unable to create folder")
		t.FailNow()
	}

	// 2. create 1000 objects in the folder
	for i := 0; i < maxobjects; i++ {
		_, err := clients[tester10].C.CreateObject(client.CreateObjectRequest{Name: "ChildItem" + strconv.Itoa(i), TypeName: "File", RawAcm: acm, ParentID: folderResponse.ID}, nil)
		if err != nil {
			t.Logf("Unable to create child %d %v", i, err)
			t.FailNow()
		}
	}

	// 3. list the contents of the folder
	pagingRequest := client.PagingRequest{ObjectID: folderResponse.ID, PageNumber: 1, PageSize: maxobjects}
	folderResults, err := clients[tester10].C.Search(pagingRequest, false)
	if err != nil {
		t.Logf("Unable to list folder, %v", err)
		t.FailNow()
	}
	if folderResults.PageRows != maxobjects {
		// we are constrained by service settings. Verify that total is ok
		if folderResults.TotalRows != maxobjects {
			t.Logf("folderResults.TotalRows = %d, expected %d", folderResults.TotalRows, maxobjects)
			t.FailNow()
		}
	}

}
