package dao_test

import (
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func TestCreateObject(t *testing.T) {
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection
	db, err := dbConfig.GetDatabaseHandle()
	if err != nil {
		t.Error("Unable to get handle to database: ", err.Error())
	}
	defer db.Close()

	obj, acm := setupObject("Test CreateObject", "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US", "File", "UNCLASSIFIED")
	dao.CreateObject(db, &obj, &acm)
	if obj.ID == nil {
		t.Error("expected ID to be set")
	}
	if obj.ModifiedBy != obj.CreatedBy {
		t.Error("expcted ModifiedBy to match CreatedBy")
	}
	if obj.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	err = dao.DeleteObject(db, &obj, true)
	if err != nil {
		t.Error(err)
	}
}

func setupObject(name string, createdBy string, typeName string, classification string) (models.ODObject, models.ODACM) {
	var obj models.ODObject
	obj.Name = name
	obj.CreatedBy = createdBy
	obj.TypeName.String = typeName
	obj.TypeName.Valid = true
	var acm models.ODACM
	acm.CreatedBy = obj.CreatedBy
	acm.Classification.String = classification
	acm.Classification.Valid = true

	return obj, acm
}
