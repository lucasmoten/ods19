package server_test

import (
	"net/http/httptest"
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/cmd/metadataconnector/libs/server"
	"decipher.com/oduploader/metadata/models"
)

func TestAppServerGetObject(t *testing.T) {

	// expected return type
	obj := models.ODObject{}

	// set up fake DAO
	fakeDAO := dao.FakeDAO{Object: &obj}

	// set up server
	fakeServer := server.AppServer{DAO: &fakeDAO}
	s := httptest.NewServer(fakeServer)

	defer s.Close()
	s.URL = ""

	// set up request
	// req, err := http.NewRequest("GET", )

	s.Start()

	_ = s

	//
}
