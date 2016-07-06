package server_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

// TestAcmWithoutShare - User T1 creates object O1 with ACM having no share.
// Verify T1..T10, and another known DN have access to read the object
// (by virtue of it shared to everyone)
func TestAcmWithoutShare(t *testing.T) {

	// ### Create object O1 as tester1
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O1"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"]}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := host + cfg.NginxRootURL + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{TLSClientConfig: clients[tester1].Config}
	client := &http.Client{Transport: transport}
	// exec and get response
	httpCreateResponse, err := client.Do(httpCreate)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// check status of response
	if httpCreateResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating object: %s", httpCreateResponse.Status)
		t.FailNow()
	}
	// parse back to boject
	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	// ### Verify all clients can read it
	uriGetProperties := host + cfg.NginxRootURL + "/objects/" + createdObject.ID + "/properties"
	httpGet, _ := http.NewRequest("GET", uriGetProperties, nil)
	for clientIdx, ci := range clients {
		transport := &http.Transport{TLSClientConfig: ci.Config}
		client := &http.Client{Transport: transport}
		httpGetResponse, err := client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		if httpGetResponse.StatusCode != http.StatusOK {
			t.Logf("Bad status for client %d. Status was %s", clientIdx, httpGetResponse.Status)
			t.Fail()
		} else {
			t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
		}
		ioutil.ReadAll(httpGetResponse.Body)
		httpGetResponse.Body.Close()
	}

}

// TestAcmWithShareForODrive - User T1 creates object O2 with ACM having share
// for group ODrive. Verify T1..T10 but no other DNs have access since only
// T1..T10 are in that group
func TestAcmWithShareForODrive(t *testing.T) {

	// ### Create object O2 as tester1
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O2"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive"]}}}}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := host + cfg.NginxRootURL + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{TLSClientConfig: clients[tester1].Config}
	client := &http.Client{Transport: transport}
	// exec and get response
	httpCreateResponse, err := client.Do(httpCreate)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// check status of response
	if httpCreateResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating object: %s", httpCreateResponse.Status)
		t.FailNow()
	}
	// parse back to boject
	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	// ### Verify all clients can read it
	uriGetProperties := host + cfg.NginxRootURL + "/objects/" + createdObject.ID + "/properties"
	httpGet, _ := http.NewRequest("GET", uriGetProperties, nil)
	for clientIdx, ci := range clients {
		transport := &http.Transport{TLSClientConfig: ci.Config}
		client := &http.Client{Transport: transport}
		httpGetResponse, err := client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		if clientIdx < 10 {
			// Tester 1 - 10 is in the ODrive group, should be allowed
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		} else {
			// Client 11 (the server cert) isn't in the odrive group, and should be forbidden
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		ioutil.ReadAll(httpGetResponse.Body)
		httpGetResponse.Body.Close()
	}
}

// TestAcmWithShareForODriveG1Disallowed - User T1 creates object O3 with ACM
// having share for group ODrive G1. Verify that this is rejected since T1 is
// not in that group.
func TestAcmWithShareForODriveG1Disallowed(t *testing.T) {
	// ### Create object O3 as tester1
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O3"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive_G1"]}}}}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := host + cfg.NginxRootURL + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{TLSClientConfig: clients[tester1].Config}
	client := &http.Client{Transport: transport}
	// exec and get response
	httpCreateResponse, err := client.Do(httpCreate)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// check status of response
	if httpCreateResponse.StatusCode != http.StatusForbidden {
		t.Logf("Bad status when creating object: %s", httpCreateResponse.Status)
		t.FailNow()
	} else {
		t.Logf("%s is not allowed to create object %s with acm %s", clients[tester1].Name, createObjectRequest.Name, createObjectRequest.RawAcm)
	}
	ioutil.ReadAll(httpCreateResponse.Body)
	httpCreateResponse.Body.Close()
}

// TestAcmWithShareForODriveG1Allowed - User T10 creates object O4 with ACM
// having share for group ODrive G1. Verify that this is created, and
// accessible by T6..T10, but not T1..T5 or another DN
func TestAcmWithShareForODriveG1Allowed(t *testing.T) {
	// ### Create object O4 as tester10
	tester10 := 0
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O4"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive_G1"]}}}}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := host + cfg.NginxRootURL + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{TLSClientConfig: clients[tester10].Config}
	client := &http.Client{Transport: transport}
	// exec and get response
	httpCreateResponse, err := client.Do(httpCreate)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// check status of response
	if httpCreateResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating object: %s", httpCreateResponse.Status)
		t.FailNow()
	}
	// parse back to boject
	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	// ### Verify tester 6-10 can read it, but not 1-5 or other certs
	uriGetProperties := host + cfg.NginxRootURL + "/objects/" + createdObject.ID + "/properties"
	httpGet, _ := http.NewRequest("GET", uriGetProperties, nil)
	for clientIdx, ci := range clients {
		transport := &http.Transport{TLSClientConfig: ci.Config}
		client := &http.Client{Transport: transport}
		httpGetResponse, err := client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		switch clientIdx {
		case 0, 6, 7, 8, 9:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // 1-5 + twl-server-generic
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		ioutil.ReadAll(httpGetResponse.Body)
		httpGetResponse.Body.Close()
	}
}

// TestAcmWithShareForODriveG2Allowed - User T1 creates object O5 with ACM
// having share for group ODrive G2. Verify that this is created, and
// accessible by T1..T5, but not T6..T10 or another DN
func TestAcmWithShareForODriveG2Allowed(t *testing.T) {
	// ### Create object O5 as tester1
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O5"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive_G2"]}}}}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := host + cfg.NginxRootURL + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{TLSClientConfig: clients[tester1].Config}
	client := &http.Client{Transport: transport}
	// exec and get response
	httpCreateResponse, err := client.Do(httpCreate)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// check status of response
	if httpCreateResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating object: %s", httpCreateResponse.Status)
		t.FailNow()
	}
	// parse back to boject
	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	// ### Verify tester 1-5 can read it, but not 6-10 or other certs
	uriGetProperties := host + cfg.NginxRootURL + "/objects/" + createdObject.ID + "/properties"
	httpGet, _ := http.NewRequest("GET", uriGetProperties, nil)
	for clientIdx, ci := range clients {
		transport := &http.Transport{TLSClientConfig: ci.Config}
		client := &http.Client{Transport: transport}
		httpGetResponse, err := client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		switch clientIdx {
		case 1, 2, 3, 4, 5:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // tester 6-10 + twl-server-generic
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		ioutil.ReadAll(httpGetResponse.Body)
		httpGetResponse.Body.Close()
	}
}

// TestAcmWithShareForODriveG2Disallowed - User T10 creates object O6 with ACM
// having share for group ODrive G2. Verify that this is rejected since T10 is
// not in that group.
func TestAcmWithShareForODriveG2Disallowed(t *testing.T) {
	// ### Create object O6 as tester1
	tester10 := 0
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O6"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive_G2"]}}}}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := host + cfg.NginxRootURL + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{TLSClientConfig: clients[tester10].Config}
	client := &http.Client{Transport: transport}
	// exec and get response
	httpCreateResponse, err := client.Do(httpCreate)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// check status of response
	if httpCreateResponse.StatusCode != http.StatusForbidden {
		t.Logf("Bad status when creating object: %s", httpCreateResponse.Status)
		t.FailNow()
	} else {
		t.Logf("%s is not allowed to create object %s with acm %s", clients[tester10].Name, createObjectRequest.Name, createObjectRequest.RawAcm)
	}
	ioutil.ReadAll(httpCreateResponse.Body)
	httpCreateResponse.Body.Close()
}

// TestAddReadShareForUser - User T1 creates object O7 with ACM having share
// for group ODrive G2. This is created and accessible by T1..T5, but not by
// T6..T10 or other users not in the group. Then add share to T10 allowRead
// and verify that T10 is then able to read it.
func TestAddReadShareForUser(t *testing.T) {
	// ### Create object O7 as tester1
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O7"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive_G2"]}}}}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := host + cfg.NginxRootURL + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{TLSClientConfig: clients[tester1].Config}
	client := &http.Client{Transport: transport}
	// exec and get response
	httpCreateResponse, err := client.Do(httpCreate)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// check status of response
	if httpCreateResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating object: %s", httpCreateResponse.Status)
		t.FailNow()
	}
	// parse back to boject
	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	// ### Add a share for tester 10 to be able to read the object
	// prep share
	var createShareRequest protocol.ObjectShare
	createShareRequest.AllowRead = true
	createShareRequest.Share = makeUserShare("cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us")
	createShareRequest.PropagateToChildren = false
	// jsonify it
	jsonBody, _ = json.Marshal(createShareRequest)
	// prep http request
	uriShare := host + cfg.NginxRootURL + "/shared/" + createdObject.ID
	// prep http request
	httpCreateShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateShare.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateShareResponse, err := client.Do(httpCreateShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// check status of response
	if httpCreateShareResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating share: %s", httpCreateShareResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject protocol.Object
	err = util.FullDecode(httpCreateShareResponse.Body, &updatedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	// ### Verify tester 1-5 can read it, as well as 10, but not 6-9 or other certs
	uriGetProperties := host + cfg.NginxRootURL + "/objects/" + createdObject.ID + "/properties"
	httpGet, _ := http.NewRequest("GET", uriGetProperties, nil)
	for clientIdx, ci := range clients {
		transport := &http.Transport{TLSClientConfig: ci.Config}
		client := &http.Client{Transport: transport}
		httpGetResponse, err := client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		switch clientIdx {
		case 0, 1, 2, 3, 4, 5:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // 6-9 + twl-server-generic
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		ioutil.ReadAll(httpGetResponse.Body)
		httpGetResponse.Body.Close()
	}
}

// TestAddReadAndUpdateShareForUser - User T1 creates object O8 with ACM having
// share for group ODrive G2. This is created and accessible by T1..T5, but not
// by T6..T10 or other users not in the group. Then add share to T10 allowRead
// and verify that T10 is then able to read it, but T6..T9 still cannot. Next,
// add share to group G1 allowRead, allowUpdate. Verify that T1..T10 can read.
// Verify that T9 can update it by changing the name
func TestAddReadAndUpdateShareForUser(t *testing.T) {
	// ### Create object O8 as tester1
	tester1 := 1
	// prep object
	var createObjectRequest protocol.CreateObjectRequest
	createObjectRequest.Name = "TestACM O8"
	createObjectRequest.TypeName = "Folder"
	createObjectRequest.RawAcm = `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive_G2"]}}}}`
	createObjectRequest.ContentSize = 0
	// jsonify it
	jsonBody, _ := json.Marshal(createObjectRequest)
	// prep http request
	uriCreate := host + cfg.NginxRootURL + "/objects"
	httpCreate, _ := http.NewRequest("POST", uriCreate, bytes.NewBuffer(jsonBody))
	httpCreate.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{TLSClientConfig: clients[tester1].Config}
	client := &http.Client{Transport: transport}
	// exec and get response
	httpCreateResponse, err := client.Do(httpCreate)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// check status of response
	if httpCreateResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating object: %s", httpCreateResponse.Status)
		t.FailNow()
	}
	// parse back to boject
	var createdObject protocol.Object
	err = util.FullDecode(httpCreateResponse.Body, &createdObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	// ### Add a share for tester 10 to be able to read the object
	// prep share
	var createShareRequest protocol.ObjectShare
	createShareRequest.AllowRead = true
	createShareRequest.Share = makeUserShare("cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us")
	createShareRequest.PropagateToChildren = false
	// jsonify it
	jsonBody, _ = json.Marshal(createShareRequest)
	// prep http request
	uriShare := host + cfg.NginxRootURL + "/shared/" + createdObject.ID
	// prep http request
	httpCreateShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateShare.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateShareResponse, err := client.Do(httpCreateShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// check status of response
	if httpCreateShareResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating share: %s", httpCreateShareResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject protocol.Object
	err = util.FullDecode(httpCreateShareResponse.Body, &updatedObject)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	// ### Verify tester 1-5 can read it, as well as 10, but not 6-9 or other certs
	uriGetProperties := host + cfg.NginxRootURL + "/objects/" + createdObject.ID + "/properties"
	httpGet, _ := http.NewRequest("GET", uriGetProperties, nil)
	for clientIdx, ci := range clients {
		transport := &http.Transport{TLSClientConfig: ci.Config}
		client := &http.Client{Transport: transport}
		httpGetResponse, err := client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		switch clientIdx {
		case 0, 1, 2, 3, 4, 5:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // 6-9 + twl-server-generic
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		ioutil.ReadAll(httpGetResponse.Body)
		httpGetResponse.Body.Close()
	}

	// ### Add a share for G1 group to allow reading and updating
	// prep share
	var createGroupShareRequest protocol.ObjectShare
	createGroupShareRequest.AllowRead = true
	createGroupShareRequest.AllowUpdate = true
	createGroupShareRequest.Share = makeGroupShare("DCTC", "DCTC", "ODrive_G1")
	createGroupShareRequest.PropagateToChildren = false
	// jsonify it
	jsonBody, _ = json.Marshal(createGroupShareRequest)
	// prep http request
	httpCreateGroupShare, _ := http.NewRequest("POST", uriShare, bytes.NewBuffer(jsonBody))
	httpCreateGroupShare.Header.Set("Content-Type", "application/json")
	// exec and get response
	httpCreateGroupShareResponse, err := client.Do(httpCreateGroupShare)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// check status of response
	if httpCreateGroupShareResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when creating share: %s", httpCreateGroupShareResponse.Status)
		t.FailNow()
	}
	// parse back to object
	var updatedObject2 protocol.Object
	err = util.FullDecode(httpCreateGroupShareResponse.Body, &updatedObject2)
	if err != nil {
		t.Logf("Error decoding json to Object: %v", err)
		t.FailNow()
	}

	// ### Verify tester 1-10 can read it, but not others
	for clientIdx, ci := range clients {
		transport := &http.Transport{TLSClientConfig: ci.Config}
		client := &http.Client{Transport: transport}
		httpGetResponse, err := client.Do(httpGet)
		if err != nil {
			t.Logf("Error retrieving properties for client %d: %v", clientIdx, err)
			t.Fail()
		}
		switch clientIdx {
		case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9:
			if httpGetResponse.StatusCode != http.StatusOK {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is allowed to read %s", ci.Name, createdObject.Name)
			}
		default: // twl-server-generic and any others that may get added later
			if httpGetResponse.StatusCode != http.StatusForbidden {
				t.Logf("Bad status for client %d (%s). Status was %s", clientIdx, ci.Name, httpGetResponse.Status)
				t.Fail()
			} else {
				t.Logf("%s is denied access to read %s", ci.Name, createdObject.Name)
			}
		}
		ioutil.ReadAll(httpGetResponse.Body)
		httpGetResponse.Body.Close()
	}

	// ### Verify that Tester 9 can now update it
	tester9 := 9
	updatedObject2.Name += " changed by Tester09"
	uriUpdate := host + cfg.NginxRootURL + "/objects/" + updatedObject2.ID + "/properties"
	// jsonify it
	jsonBody, _ = json.Marshal(updatedObject2)
	// prep http request
	httpUpdateObject, _ := http.NewRequest("POST", uriUpdate, bytes.NewBuffer(jsonBody))
	httpUpdateObject.Header.Set("Content-Type", "application/json")
	// exec and get response
	transport9 := &http.Transport{TLSClientConfig: clients[tester9].Config}
	client9 := &http.Client{Transport: transport9}
	httpUpdateObjectResponse, err := client9.Do(httpUpdateObject)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.FailNow()
	}
	// check status of response
	if httpUpdateObjectResponse.StatusCode != http.StatusOK {
		t.Logf("Bad status when updating object: %s", httpUpdateObjectResponse.Status)
		t.FailNow()
	}

}

// TestAddReadShareForGroupRemovesEveryone - User T1 Adds Share with read
// permission for group ODrive G2 to O1. The existing share to everyone should
// be revoked. T1..T5 should have read access. T1 retains create, delete,
// update, and share access. T6..T10 should no longer see the object as its not
// shared to everyone.
// DEPENDS ON SUCCESSFUL RUN OF TestAcmWithoutShare or COPY
func TestAddReadShareForGroupRemovesEveryone(t *testing.T) {

}

// TestAddReadShareToUserWithoutEveryone - User T1 Adds Share with read
// permission for user T10 to O1. T1 retains full CRUDS, T1..T5 retains read
// access from the share established in 9 and T10 should now get read access.
// DEPENDS ON SUCCESSFUL RUN OF TestAcmWithoutShare or COPY
// DEPENDS ON SUCCESSFUL RUN OF TestAddReadShareForGroupRemovesEveryone or COPY
func TestAddReadShareToUserWithoutEveryone(t *testing.T) {

}

// TestUpdateAcmWithoutSharingToUser - User T1 Updates O1 setting an ACM that
// has a share for O1. T1 retains full CRUDS, T1..T5 retains read access from
// the share as it remains in place, and T10 should lose access as the read
// permission should be marked deleted since ACM overrides.
// DEPENDS ON SUCCESSFUL RUN OF TestAcmWithoutShare or COPY
// DEPENDS ON SUCCESSFUL RUN OF TestAddReadShareForGroupRemovesEveryone or COPY
// DEPENDS ON SUCCESSFUL RUN OF TestAddReadShareToUserWithoutEveryone or COPY
func TestUpdateAcmWithoutSharingToUser(t *testing.T) {

}

// TestUpdateAcmWithoutAnyShare - User T1 Updates O1 setting an ACM that has an
// empty share. T1 retains full CRUDS. Share to Odrive G2 in ACM is removed as
// is permission. Permission to EveryoneGroup established. T1..T10 have read
// access. Any other recognized DN also has read access
// DEPENDS ON SUCCESSFUL RUN OF TestAcmWithoutShare or COPY
// DEPENDS ON SUCCESSFUL RUN OF TestAddReadShareForGroupRemovesEveryone or COPY
// DEPENDS ON SUCCESSFUL RUN OF TestAddReadShareToUserWithoutEveryone or COPY
// DEPENDS ON SUCCESSFUL RUN OF TestUpdateAcmWithoutSharingToUser or COPY
func TestUpdateAcmWithoutAnyShare(t *testing.T) {

}
