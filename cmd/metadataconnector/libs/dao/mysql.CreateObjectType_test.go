package dao_test

import (
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func TestCreateObjectType(t *testing.T) {
	var objectType models.ODObjectType
	objectType.Name = "Test Type"
	objectType.CreatedBy = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"

	dbObjectType, err := dao.GetObjectTypeByName(db, objectType.Name, false, objectType.CreatedBy)
	if err != nil {
		t.Error(err)
	}
	if dbObjectType.ID == nil {
		dao.CreateObjectType(db, &objectType)
	} else {
		objectType = dbObjectType
	}

	if objectType.ID == nil {
		t.Error("expected ID to be set")
	}
	if objectType.ModifiedBy != objectType.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}

	err = dao.DeleteObjectType(db, &objectType)
	if err != nil {
		t.Error(err)
	}
}