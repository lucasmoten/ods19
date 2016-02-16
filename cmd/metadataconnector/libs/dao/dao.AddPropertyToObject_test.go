package dao_test

import (
	"log"
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
)

func TestDAOAddPropertyToObject(t *testing.T) {
	if db == nil {
		log.Fatal("db is nil")
	}

	daoConn := dao.DataAccessLayer{db}
	_ = daoConn
}
