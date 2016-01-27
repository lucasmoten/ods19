package dao_test

import (
	"fmt"
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func TestGetObjectTypeByName(t *testing.T) {
	var objectTypeName = "Test Object Type By Name"

	// create object type
	var objectType models.ODObjectType
	objectType.Name = objectTypeName
	objectType.CreatedBy = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	dao.CreateObjectType(db, &objectType)
	if objectType.ID == nil {
		t.Error("expected ID to be set")
	}
	if objectType.ModifiedBy != objectType.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}

	// get the object type by name
	objectTypeByName, err := dao.GetObjectTypeByName(db, objectTypeName, false, "")
	if err != nil {
		t.Error(err)
	}
	if objectTypeByName.Name != objectTypeName {
		t.Error(fmt.Errorf("expected objectTypeByName Name to be %s, got %s", objectTypeName, objectTypeByName.Name))
	}
	if objectTypeByName.IsDeleted {
		t.Error("Object type was marked as deleted")
	}

	// delete the object type
	err = dao.DeleteObjectType(db, &objectTypeByName)
	if err != nil {
		t.Error(err)
	}

	// Refetch by id
	objectTypeByName2, err := dao.GetObjectTypeByName(db, objectTypeName, false, "")
	if err != nil {
		t.Error(err)
	}
	if objectTypeByName2.Name != objectTypeName {
		t.Error(fmt.Errorf("expected objectTypeByName Name to be %s, got %s", objectTypeName, objectTypeByName2.Name))
	}
	if !objectTypeByName2.IsDeleted {
		t.Error("Object type was not marked as deleted")
	}
}