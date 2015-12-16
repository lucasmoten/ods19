package integration

import (
	"bytes"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"golang.org/x/net/context"

	"decipher.com/oduploader/services/transfer"

	httptransport "github.com/go-kit/kit/transport/http"
)

// TODO extract config port
var serviceIP = "127.0.0.1:6060"

func TestUploadFile(t *testing.T) {

	filename := "./testfiles/plaintext.txt"

	svc := transfer.TransferServiceImpl{}
	uploadHandler := httptransport.NewServer(
		context.Background(),
		transfer.MakeUploadEndpoint(svc),
		transfer.DecodeUploadRequest,
		transfer.EncodeResponse,
	)
	ts := httptest.NewServer(uploadHandler)
	defer ts.Close()

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("theFile", filename)
	if err != nil {
		log.Fatal(err)
	}
	_ = fileWriter // satistfy compiler

	fh, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	_ = fh // satisfy compiler

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()
	var targetUrl string
	resp, err := http.Post(targetUrl, contentType, bodyBuf)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

}
