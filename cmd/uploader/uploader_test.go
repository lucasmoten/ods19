package main

import (
	"decipher.com/oduploader/cmd/uploader/libs"
	"decipher.com/oduploader/config"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	//"strings"
	"io"
	"testing"
)

var (
	theEnv      *config.Environment
	theServer   *http.Server
	theUploader *libs.Uploader
)

func doSmallURLTest(t *testing.T, method, uri string, body io.Reader) {
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(method, "https://"+theServer.Addr+"/"+uri, body)
	//XXX hack
	req.RequestURI = uri
	if err != nil {
		log.Printf("Unable to fetch:%s\n%v", uri, err)
		t.Fail()
	}

	//Consume the request and put it into resp
	theUploader.ServeHTTP(resp, req)
	if _, err := ioutil.ReadAll(resp.Body); err != nil {
		t.Fail()
	} else {
		if resp.Code != 200 {
			t.Errorf("fail (%d) to GET %s", resp.Code, uri)
		}
	}
}

func TestHttpServer(t *testing.T) {
	//XXX The URL shows up blank in the uploader at the moment
	doSmallURLTest(t, "GET", "/stats", nil)
}

func init() {
	//Configure a server to be run
	//This is a kind of test of its own, but it builds the web server objects
	var err error
	theEnv, theUploader, theServer, err = BuildServer()
	if err != nil {
		log.Printf("Could not build server:%v", err)
		return
	}
}
