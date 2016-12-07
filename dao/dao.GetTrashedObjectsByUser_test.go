package dao_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestDAOGetTrashedObjectsByUser(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	// Create an object. Delete the object. Object should
	// show up in trash.

	user3 := setupUserWithSnippets(usernames[3])
	pagingRequest := dao.PagingRequest{PageNumber: 1, PageSize: 1000}
	// Create an object.
	objA := createTestObjectAllPermissions(user3.DistinguishedName)
	createdA, err := d.CreateObject(&objA)
	if err != nil {
		t.Fatalf("Error creating objA: %v\n", err)
	}
	// Delete the object, placing it in the trash.
	err = d.DeleteObject(user3, createdA, true)
	if err != nil {
		t.Fatalf("Error deleting object createdA: %v\n", err)
	}

	// Call listObjectsTrashed for user.
	results, err := d.GetTrashedObjectsByUser(user3, pagingRequest)

	// Ensure that the delete objects in trash.
	intrash := false
	for _, o := range results.Objects {
		if o.Name == objA.Name {
			intrash = true
		}
	}
	if !intrash {
		t.Fail()
	}
	if t.Failed() {
		t.Log("Errors finding objA in trash")
		t.FailNow()
	}
}

func TestDAOGetTrashedObjectsDeleteParent(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	user3 := setupUserWithSnippets(usernames[3])
	pagingRequest := dao.PagingRequest{PageNumber: 1, PageSize: 1000}

	// Create parent-child relationship, then delete the parent.
	// Parent should show up in trash. Child should not show up
	// in trash. Neither should show up on listObjects.
	parent1, child1, err := createParentChildObjectPair(user3.DistinguishedName)
	if err != nil {
		t.Errorf("Error creating parent-child pair: %v\n", err)
	}
	if err = d.DeleteObject(user3, parent1, true); err != nil {
		t.Errorf("Error deleting test parent1: %v\n", err)
	}
	results, err := d.GetTrashedObjectsByUser(user3, pagingRequest)
	if err != nil {
		t.Errorf("Error calling GetTrashedObjectsByUser: %v\n", err)
	}
	intrash := false
	// Assert parent is in trash.
	for _, o := range results.Objects {
		if o.Name == parent1.Name {
			intrash = true
		}
	}
	if !intrash {
		t.Logf("Object parent1 is deleted but does not appear in trash")
		t.Fail()
	}
	// Assert child is not in trash.
	intrash = false
	for _, o := range results.Objects {
		if o.Name == child1.Name {
			intrash = true
			t.Logf("Object child1 should not be in user3 trash.")
			t.Fail()
		}
	}
	// Assert neither show up in root
	results, err = d.GetRootObjectsByUser(user3, pagingRequest)
	for _, o := range results.Objects {
		if o.Name == child1.Name {
			t.Logf("Object child1 has a deleted parent, should not show in GetRootObjectsByUser. ID of child is %s", hex.EncodeToString(o.ID))
			t.Fail()
		}
		if o.Name == parent1.Name {
			t.Logf("Object parent1 is deleted, should not show in GetRootObjectsByUser. ID of parent is %s", hex.EncodeToString(o.ID))
			t.Fail()
		}
	}
	// Assert neither show up in listObjects.
	results, err = d.GetChildObjectsByUser(user3, pagingRequest, parent1)
	for _, o := range results.Objects {
		if o.Name == child1.Name {
			t.Logf("Object child1 has a deleted parent, should not show in GetChildObjectsByUser. ID of child is %s", hex.EncodeToString(o.ID))
			t.Fail()
		}
		if o.Name == parent1.Name {
			t.Logf("Object parent1 is deleted, should not show in GetChildObjectsByUser. ID of parent is %s", hex.EncodeToString(o.ID))
			t.Fail()
		}
	}

	if t.Failed() {
		t.Log("Errors in test cases for parent1, child1 object pair.")
		t.FailNow()
	}
}
func TestDAOGetTrashedObjectsDeleteChild(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	user3 := setupUserWithSnippets(usernames[3])
	pagingRequest := dao.PagingRequest{PageNumber: 1, PageSize: 1000}
	// Create parent-child relationship, then delete the child.
	// Child should show up in trash. Parent should show up in
	// listObjects.
	parent2, child2, err := createParentChildObjectPair(user3.DistinguishedName)
	if err != nil {
		t.Logf("Error creating parent-child pair: %v\n", err)
		t.Fail()
	}
	if err = d.DeleteObject(user3, child2, true); err != nil {
		t.Logf("Error deleting test parent2: %v\n", err)
		t.Fail()
	}
	// Assert child2 is in trash.
	intrash := false
	results, err := d.GetTrashedObjectsByUser(user3, pagingRequest)
	for _, o := range results.Objects {
		if o.Name == child2.Name {
			intrash = true
		}
	}
	if !intrash {
		t.Logf("child2 was not found in the trash after deleted!")
		t.Fail()
	}
	// Assert parent2 is in listObjects.
	results, err = d.GetRootObjectsByUser(user3, pagingRequest)
	inlist := false
	for _, o := range results.Objects {
		if o.Name == parent2.Name {
			inlist = true
		}
	}
	if !inlist {
		t.Logf("Object parent2 should show up in GetRootObjectsByUser.")
		t.Fail()
	}
	// Assert child2 is not in listObjects.
	results, err = d.GetChildObjectsByUser(user3, pagingRequest, parent2)
	intrash = false
	for _, o := range results.Objects {
		if o.Name == child2.Name {
			intrash = true
			t.Logf("Object child2 is deleted, should not show up in GetChildObjectsByUser.")
			t.Fail()
		}
	}
	if t.Failed() {
		t.FailNow()
	}
}

// createParentChildObjectPair creates a parent object at root and a child of that parent.
// Internally it delegates to createTestObjectAllPermissions.
func createParentChildObjectPair(username string) (parent models.ODObject, child models.ODObject, err error) {

	parent = createTestObjectAllPermissions(username)
	child = createTestObjectAllPermissions(username)
	parent, err = d.CreateObject(&parent)
	if err != nil {
		return parent, child, err
	}
	child.ParentID = parent.ID
	child, err = d.CreateObject(&child)
	return parent, child, err
}

// createTestObjectAllPermissions creates a minimal File type object for testing
// with all permissions true. The Name field is assigned a GUID for easy uniqueness verification.
func createTestObjectAllPermissions(username string) models.ODObject {

	var obj models.ODObject

	name, _ := util.NewGUID()
	obj.Name = name
	obj.CreatedBy = username
	obj.TypeName.String = "File"
	obj.TypeName.Valid = true
	obj.RawAcm.String = testhelpers.ValidACMUnclassified

	var perms models.ODObjectPermission
	perms.CreatedBy = username
	perms.Grantee = models.AACFlatten(username)
	perms.AcmShare = fmt.Sprintf(`{"users":[%s]}`, perms.CreatedBy)
	perms.AcmGrantee.Grantee = perms.Grantee
	perms.AcmGrantee.UserDistinguishedName.String = perms.CreatedBy
	perms.AcmGrantee.UserDistinguishedName.Valid = true
	perms.AllowCreate, perms.AllowDelete, perms.AllowRead, perms.AllowUpdate = true, true, true, true

	obj.Permissions = append(obj.Permissions, perms)
	return obj

}
