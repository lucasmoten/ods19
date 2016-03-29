package dao_test

import (
	"bytes"
	"fmt"
	"testing"

	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
	"decipher.com/oduploader/util/testhelpers"
)

func TestDAOGetChildObjects(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	// Create our parent object
	var parent models.ODObject
	parent.Name = "Test GetChildObjects Parent"
	parent.CreatedBy = usernames[1] // "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
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
	child.CreatedBy = usernames[1] // "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
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
	pagingRequest := protocol.PagingRequest{PageNumber: 1, PageSize: 10, SortSettings: []protocol.SortSetting{protocol.SortSetting{SortField: "name", SortAscending: true}}}
	resultset, err := d.GetChildObjects(pagingRequest, dbParent)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != 1 {
		t.Error(fmt.Errorf("Resultset had %d totalrows", resultset.TotalRows))
		t.Error("expected 1 child")
	}

	// cleanup
	err = d.DeleteObject(dbParent, true)
	if err != nil {
		t.Error(err)
	}

}
