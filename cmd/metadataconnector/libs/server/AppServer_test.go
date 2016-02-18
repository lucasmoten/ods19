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

	// set up request
	s := httptest.NewServer(fakeServer)

	s.Start()

	_ = s

	//
}
