package server_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"testing"

	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestEtag(t *testing.T) {
	clientID := 5
	b := []byte(`abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@`)

	file, cleanup := testhelpers.GenerateTempFileFromBytes(b, t)
	defer cleanup()

	req, err := testhelpers.NewCreateObjectPOSTRequest(host, "", file)
	if err != nil {
		t.Errorf("Failure from NewCreateObjectPOSTRequest: %v\n", err)
	}

	client := clients[clientID].Client
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
	}

	var responseObject protocol.Object
	err = util.FullDecode(res.Body, &responseObject)
	if err != nil {
		t.Errorf("Could not decode reponse from createObject: %v\n", err)
	}

	//Ask for it in order to get the eTag and a 200
	req2, err := testhelpers.NewGetObjectStreamRequest(responseObject.ID, "", host)
	if err != nil {
		t.Errorf("Failure from redo get object stream: %v\n", err)
	}
	client2 := client
	res2, err := client2.Do(req2)
	if err != nil {
		t.Errorf("Unable to do re request:%v\n", err)
	}
	eTag := res2.Header.Get("Etag")
	t.Logf("we got eTag:%s", eTag)
	if len(eTag) == 0 {
		//We have no situation where a stream does not return an Etag now
		t.Errorf("We did not get an Etag back")
	}

	if res2.StatusCode != http.StatusOK {
		t.Errorf("bad status on get: %d", res2.StatusCode)
	}

	//Ask again with the eTag and get a 304
	req3, err := testhelpers.NewGetObjectStreamRequest(responseObject.ID, "", host)
	if err != nil {
		t.Errorf("Failure from redo get object stream: %v\n", err)
	}
	req3.Header.Set("If-none-match", eTag)
	client3 := client
	res3, err := client3.Do(req3)
	if err != nil {
		t.Errorf("Unable to do re request:%v\n", err)
	}

	if res3.StatusCode != http.StatusNotModified {
		t.Errorf("the data was not modified, and we sent an eTag, yet did not get a 304. %d instead", res3.StatusCode)
	}
}

func TestUploadAndGetByteRange(t *testing.T) {

	b := []byte(`abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@`)
	start, end := 5, 35
	expected := b[start:end]

	file, cleanup := testhelpers.GenerateTempFileFromBytes(b, t)
	defer cleanup()

	req, err := testhelpers.NewCreateObjectPOSTRequest(host, "", file)
	if err != nil {
		t.Errorf("Failure from NewCreateObjectPOSTRequest: %v\n", err)
	}

	client := clients[5].Client
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("Unable to do request:%v\n", err)
	}
	isResponseError(res, t)

	var responseObject protocol.Object
	err = util.FullDecode(res.Body, &responseObject)
	if err != nil {
		t.Errorf("Could not decode reponse from createObject: %v\n", err)
	}

	rangeReq, err := testhelpers.NewGetObjectStreamRequest(responseObject.ID, "", host)
	if err != nil {
		t.Errorf("Could not create GetObjectStreamRequest: %v\n", err)
	}
	rangeReq.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", start, end-1))

	rangeRes, err := client.Do(rangeReq)
	if err != nil {
		t.Errorf("Could not perform range request: %v\n", err)
	}

	returned, err := ioutil.ReadAll(rangeRes.Body)
	if err != nil {
		t.Errorf("Could not read response body")
	}

	if len(returned) != len(expected) {
		t.Errorf("Response from byte range request != expected value. Expected %v Got: %v\n", len(expected), len(returned))
	}
}

func isResponseError(res *http.Response, t *testing.T) {
	if res == nil {
		t.Errorf("Response was nil")
		t.FailNow()
	}
	if res.StatusCode != 200 {
		t.Errorf("Expected 200 response")
		t.FailNow()
	}
}

func debugResponse(resp *http.Response) {
	dump, _ := httputil.DumpResponse(resp, true)
	fmt.Println("DEBUG:")
	fmt.Printf("%q", dump)
}
