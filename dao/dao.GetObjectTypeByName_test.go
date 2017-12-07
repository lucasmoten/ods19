package dao_test

import (
	"fmt"
	"testing"

	"github.com/deciphernow/object-drive-server/metadata/models"
)

func TestDAOGetObjectTypeByName(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	var objectTypeName = "Test Object Type By Name"

	// create object type
	var objectType models.ODObjectType
	objectType.Name = objectTypeName
	objectType.CreatedBy = usernames[1]
	dbObjectType, err := d.CreateObjectType(&objectType)
	if dbObjectType.ID == nil {
		t.Error("expected ID to be set")
	}
	if dbObjectType.ModifiedBy != objectType.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}

	// get the object type by name
	objectTypeByName, err := d.GetObjectTypeByName(objectTypeName, false, "")
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
	err = d.DeleteObjectType(objectTypeByName)
	if err != nil {
		t.Error(err)
	}

	// Refetch by id
	objectTypeByName2, err := d.GetObjectTypeByName(objectTypeName, false, "")
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
