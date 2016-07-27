package server_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"testing"

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

	responseObject := doCreateObjectRequest(t, clientID, req, 200)

	//Ask for it in order to get the eTag and a 200
	req2, err := testhelpers.NewGetObjectStreamRequest(responseObject.ID, "", host)
	if err != nil {
		t.Errorf("Failure from redo get object stream: %v\n", err)
	}
	//We can't use DoCreateObjectRequest because we need to extract the Etag header
	res2, err := clients[clientID].Client.Do(req2)
	if err != nil {
		t.Errorf("Unable to do re request:%v\n", err)
	}
	defer util.FinishBody(res2.Body)
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

	doCreateObjectRequest(t, clientID, req3, 304)

	//Ask with a wrong tag and get 200
	req4, err := testhelpers.NewGetObjectStreamRequest(responseObject.ID, "", host)
	if err != nil {
		t.Errorf("Failure from redo get object stream: %v\n", err)
	}
	//Some random tag that does not match
	eTag2 := "9a29ea29e29eac3457b"
	req4.Header.Set("If-none-match", eTag2)

	//We can use DoCreateObjectRequest because we definitely expect a 200 in this case
	doCreateObjectRequest(t, clientID, req4, 200)
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

	clientID := 5
	responseObject := doCreateObjectRequest(t, clientID, req, 200)

	rangeReq, err := testhelpers.NewGetObjectStreamRequest(responseObject.ID, "", host)
	if err != nil {
		t.Errorf("Could not create GetObjectStreamRequest: %v\n", err)
	}
	rangeReq.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", start, end-1))

	//We can't call DoCreateObjectRequest because we read a byte body and
	//need the count
	rangeRes, err := clients[clientID].Client.Do(rangeReq)
	if err != nil {
		t.Errorf("Could not perform range request: %v\n", err)
	}
	defer util.FinishBody(rangeRes.Body)
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
