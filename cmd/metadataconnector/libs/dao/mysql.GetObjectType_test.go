package dao_test

import (
	"fmt"
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func _TestGetObjectType(t *testing.T) {
	// create object type
	var objectType models.ODObjectType
	objectType.Name = "Test Object Type"
	objectType.CreatedBy = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	dao.CreateObjectType(db, &objectType)
	if objectType.ID == nil {
		t.Error("expected ID to be set")
	}
	if objectType.ModifiedBy != objectType.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}

	// get the object type by id
	objectTypeByID, err := dao.GetObjectType(db, &objectType)
	if err != nil {
		t.Error(err)
	}
	if objectTypeByID.Name != "Test Object Type" {
		t.Error(fmt.Errorf("expected objectTypeByID Name to be Test Object Type, got %s", objectTypeByID.Name))
	}
	if objectTypeByID.IsDeleted {
		t.Error("Object type was marked as deleted")
	}

	// delete the object type
	err = dao.DeleteObjectType(db, objectTypeByID)
	if err != nil {
		t.Error(err)
	}

	// Refetch by id
	objectTypeByID2, err := dao.GetObjectType(db, &objectType)
	if err != nil {
		t.Error(err)
	}
	if objectTypeByID2.Name != "Test Object Type" {
		t.Error(fmt.Errorf("expected objectTypeByID Name to be Test Object Type, got %s", objectTypeByID2.Name))
	}
	if !objectTypeByID2.IsDeleted {
		t.Error("Object type was not marked as deleted")
	}
}
