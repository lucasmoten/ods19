package dao_test

import (
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func TestCreateObject(t *testing.T) {

	var obj models.ODObject
	obj.Name = "Test CreateObject"
	obj.CreatedBy = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	obj.TypeName.String = "File"
	obj.TypeName.Valid = true
	var acm models.ODACM
	acm.CreatedBy = obj.CreatedBy
	acm.Classification.String = "UNCLASSIFIED"
	acm.Classification.Valid = true

	dao.CreateObject(db, &obj, &acm)
	if obj.ID == nil {
		t.Error("expected ID to be set")
	}
	if obj.ModifiedBy != obj.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if obj.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	err := dao.DeleteObject(db, &obj, true)
	if err != nil {
		t.Error(err)
	}

}
