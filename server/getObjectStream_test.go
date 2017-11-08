package server_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"testing"

	"decipher.com/object-drive-server/util"
)

func TestEtag(t *testing.T) {
	clientID := 5
	b := []byte(`abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@`)

	file, cleanup := GenerateTempFileFromBytes(b, t)
	defer cleanup()

	req, err := NewCreateObjectPOSTRequest("", file)
	if err != nil {
		t.Errorf("Failure from NewCreateObjectPOSTRequest: %v\n", err)
	}

	responseObject := doCreateObjectRequest(t, clientID, req, 200)

	//Ask for it in order to get the eTag and a 200
	req2, err := NewGetObjectStreamRequest(responseObject.ID, "")
	if err != nil {
		t.Errorf("Failure from redo get object stream: %v\n", err)
	}
	res2 := doGetObjectRequest(t, clientID, req2, 200, nil, nil)
	eTag := res2.Header.Get("Etag")
	util.FinishBody(res2.Body)
	t.Logf("we got eTag:%s", eTag)
	if len(eTag) == 0 {
		//We have no situation where a stream does not return an Etag now
		t.Errorf("We did not get an Etag back")
	}

	if res2.StatusCode != http.StatusOK {
		t.Errorf("bad status on get: %d", res2.StatusCode)
	}

	//Ask again with the eTag and get a 304
	req3, err := NewGetObjectStreamRequest(responseObject.ID, "")
	if err != nil {
		t.Errorf("Failure from redo get object stream: %v\n", err)
	}
	req3.Header.Set("If-none-match", eTag)

	res3 := doGetObjectRequest(t, clientID, req3, 304,
		trafficLogs[APISampleFile],
		&TrafficLogDescription{
			OperationName:      "Client Caching",
			RequestDescription: "Use the Etag header sent back as If-none-match to get a 304 indicating that the content has not changed",
			ResponseDescription: `
				We get back the code rather than wastefully sending back the whole file when it has not changed.  
				304 means Not-Modified.  
				Modern web browsers do this internally to avoid re-fetching unchanged content, 
				especially with images and javascript.
				When we get an object, we get an Etag back regardless of whether it was a 200 or 304.
				`,
		},
	)
	util.FinishBody(res3.Body)

	//Ask with a wrong tag and get 200
	req4, err := NewGetObjectStreamRequest(responseObject.ID, "")
	if err != nil {
		t.Errorf("Failure from redo get object stream: %v\n", err)
	}
	//Some random tag that does not match
	eTag2 := "9a29ea29e29eac3457b"
	req4.Header.Set("If-none-match", eTag2)

	//We can use DoCreateObjectRequest because we definitely expect a 200 in this case
	res4 := doGetObjectRequest(t, clientID, req4, 200, nil, nil)
	util.FinishBody(res4.Body)
}

func TestUploadAndGetByteRange(t *testing.T) {

	b := []byte(`abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@`)
	start, end := 5, 35
	expected := b[start:end]

	file, cleanup := GenerateTempFileFromBytes(b, t)
	defer cleanup()

	req, err := NewCreateObjectPOSTRequest("", file)
	if err != nil {
		t.Errorf("Failure from NewCreateObjectPOSTRequest: %v\n", err)
	}

	clientID := 5
	responseObject := doCreateObjectRequest(t, clientID, req, 200)

	rangeReq, err := NewGetObjectStreamRequest(responseObject.ID, "")
	if err != nil {
		t.Errorf("Could not create GetObjectStreamRequest: %v\n", err)
	}
	rangeReq.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", start, end-1))
	trafficLogs[APISampleFile].Request(t, rangeReq,
		&TrafficLogDescription{
			OperationName: "Range Request a file",
			RequestDescription: `
			Get a byte range out of a file, rather than the whole file.
			This is a critical feature for using media such as video over http.
			This allows for multi-gigabyte video files to be handled very easily by
			both the web server and the browser while using a small amount of memory.
			`,
			ResponseDescription: `
			The response that comes back is truncated within the requested byte range.
			Note that the Etag applies to the whole file, and not the parts.
			`,
		},
	)
	//We can't call DoCreateObjectRequest because we read a byte body and
	//need the count
	rangeRes, err := clients[clientID].Client.Do(rangeReq)
	if err != nil {
		t.Errorf("Could not perform range request: %v\n", err)
	}
	trafficLogs[APISampleFile].Response(t, rangeRes)
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
