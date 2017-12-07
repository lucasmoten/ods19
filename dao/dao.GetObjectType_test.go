package dao_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/deciphernow/object-drive-server/metadata/models"
)

func TestDAOGetObjectType(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	// create object type
	var objectType models.ODObjectType
	objectType.Name = "Test Object Type" + strconv.Itoa(time.Now().UTC().Nanosecond())
	objectType.CreatedBy = usernames[1]
	dbObjectType, err := d.CreateObjectType(&objectType)
	if err != nil {
		t.Error(err)
	}
	if dbObjectType.ID == nil {
		t.Error("expected ID to be set")
	}
	if dbObjectType.ModifiedBy != objectType.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}

	// get the object type by id
	objectTypeByID, err := d.GetObjectType(dbObjectType)
	if err != nil {
		t.Error(err)
	}
	if objectTypeByID.Name != objectType.Name {
		t.Error(fmt.Errorf("expected objectTypeByID Name to be Test Object Type, got %s", objectTypeByID.Name))
	}
	if objectTypeByID.IsDeleted {
		t.Error("Object type was marked as deleted")
	}

	// delete the object type
	err = d.DeleteObjectType(*objectTypeByID)
	if err != nil {
		t.Error(err)
	}

	// Refetch by id
	objectTypeByID2, err := d.GetObjectType(dbObjectType)
	if err != nil {
		t.Error(err)
	}
	if objectTypeByID2.Name != objectType.Name {
		t.Error(fmt.Errorf("expected objectTypeByID Name to be Test Object Type, got %s", objectTypeByID2.Name))
	}
	if !objectTypeByID2.IsDeleted {
		t.Error("Object type was not marked as deleted")
	}
}
