package dao_test

import (
	"bytes"
	"fmt"
	"testing"

	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestDAOGetChildObjects(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	// Create our parent object
	var parent models.ODObject
	parent.Name = "Test GetChildObjects Parent"
	parent.CreatedBy = usernames[1]
	parent.TypeName.String = "Test Type"
	parent.TypeName.Valid = true
	parent.RawAcm.String = testhelpers.ValidACMUnclassified
	dbParent, err := d.CreateObject(&parent)
	if err != nil {
		t.Error(err)
	}
	if dbParent.ID == nil {
		t.Error("expected ID to be set")
	}
	if dbParent.ModifiedBy != parent.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if dbParent.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// Create our child object
	var child models.ODObject
	child.Name = "Test GetChildObjects Child"
	child.CreatedBy = usernames[1]
	child.ParentID = dbParent.ID
	child.TypeName.String = "Test Type"
	child.TypeName.Valid = true
	child.RawAcm.String = testhelpers.ValidACMUnclassified
	dbChild, err := d.CreateObject(&child)
	if err != nil {
		t.Error(err)
	}
	if dbChild.ID == nil {
		t.Error("expected ID to be set")
	}
	if dbChild.ModifiedBy != child.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if dbChild.TypeID == nil {
		t.Error("expected TypeID to be set")
	}
	if !bytes.Equal(child.ParentID, dbParent.ID) {
		t.Error("expected child parentID to match parent ID")
	}
	pagingRequest := dao.PagingRequest{PageNumber: 1, PageSize: 10, SortSettings: []dao.SortSetting{dao.SortSetting{SortField: "name", SortAscending: true}}}
	resultset, err := d.GetChildObjects(pagingRequest, dbParent)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != 1 {
		t.Error(fmt.Errorf("Resultset had %d totalrows", resultset.TotalRows))
		t.Error("expected 1 child")
	}

	// cleanup
	user := models.ODUser{DistinguishedName: usernames[1]}
	err = d.DeleteObject(user, dbParent, true)
	if err != nil {
		t.Error(err)
	}

}
