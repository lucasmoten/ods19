package dao_test

import (
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/dao"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
)

func TestDAOGetRootObjects(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	pagingRequest := dao.PagingRequest{PageNumber: 1, PageSize: 1}
	// Get root Objects
	resultset, err := d.GetRootObjects(pagingRequest)
	if err != nil {
		t.Error(err)
	}
	// capture how many objects are rooted before changes
	originalTotalRows := resultset.TotalRows

	// Create an object with no parent
	var object1 models.ODObject
	object1.Name = "Test GetRootObjects"
	object1.CreatedBy = usernames[1]
	object1.TypeName = models.ToNullString("Test Type")
	object1.RawAcm.String = ValidACMUnclassified
	objectType, err := d.GetObjectTypeByName(object1.TypeName.String, true, object1.CreatedBy)
	if err != nil {
		t.Error(err)
	} else {
		object1.TypeID = objectType.ID
	}
	dbObject1, err := d.CreateObject(&object1)
	if err != nil {
		t.Error(err)
	}
	if dbObject1.ID == nil {
		t.Error("expected ID to be set")
	}
	if dbObject1.ModifiedBy != object1.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if dbObject1.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// Get root Objects
	resultset, err = d.GetRootObjects(pagingRequest)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows <= originalTotalRows {
		t.Error("expected an increase in objects at root")
	}
}
