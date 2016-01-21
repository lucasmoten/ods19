package dao_test

import (
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func TestCreateObject(t *testing.T) {
	//func CreateObject(db *sqlx.DB, object *models.ODObject, acm *models.ODACM) error {
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection
	db, err := dbConfig.GetDatabaseHandle()
	if err != nil {
		t.Error("Unable to get handle to database: ", err.Error())
	}
	defer db.Close()

	var obj models.ODObject
	obj.Name = "Sample file from Go Test"
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
		t.Error("expcted ModifiedBy to match CreatedBy")
	}
	if obj.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

}

func TestGetRootObjects(t *testing.T) {
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection
	db, err := dbConfig.GetDatabaseHandle()
	if err != nil {
		t.Error("Unable to get handle to database: ", err.Error())
	}
	defer db.Close()

	resultset, err := dao.GetRootObjects(db, "", 1, 1)
	if err != nil {
		t.Failed()
	}
	if resultset.TotalRows < 1 {
		t.Error("expected more than 0, got 0")
	}
}

func TestGetRootObjectsByOwner(t *testing.T) {
	//func GetRootObjectsByOwner(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int, owner string) (models.ODObjectResultset, error) {
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection
	db, err := dbConfig.GetDatabaseHandle()
	if err != nil {
		t.Error("Unable to get handle to database: ", err.Error())
	}
	defer db.Close()

	owner := "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	objectName := "Sample file from Go Test"
	found := false

	resultset, err := dao.GetRootObjectsByOwner(db, "", 1, 1, owner)
	if err != nil {
		t.Failed()
	}
	if resultset.TotalRows < 1 {
		t.Error("expected more than 0, got 0")
	}

	for _, object := range resultset.Objects {
		if object.Name == objectName {
			found = true
		}
	}
	if !found {
		t.Error("expeted ", objectName)
	}
}
