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

	client := httpclients[5]
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
