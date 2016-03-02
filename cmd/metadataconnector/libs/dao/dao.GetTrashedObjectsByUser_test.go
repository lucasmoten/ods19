package dao_test

import (
	"testing"

	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/util"
)

func TestDAOGetTrashedObjectsByUser(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	// Create an object. Delete the object. Object should
	// show up in trash.

	user3 := usernames[3]
	// Create an object.
	objA := createTestObjectAllPermissions(user3)
	createdA, err := d.CreateObject(&objA, nil)
	if err != nil {
		t.Fatalf("Error creating objA: %v\n", err)
	}
	// Delete the object, placing it in the trash.
	err = d.DeleteObject(createdA, true)
	if err != nil {
		t.Fatalf("Error deleting object createdA: %v\n", err)
	}

	// Call listObjectsTrashed for user.
	results, err := d.GetTrashedObjectsByUser("", 1, 1000, user3)

	// Ensure that the delete objects in trash.
	success := false
	for _, o := range results.Objects {
		if o.Name == objA.Name {
			success = true
		}
	}
	if !success {
		t.Fail()
	}

	// Create parent-child relationship, then delete the parent.
	// Parent should show up in trash. Child should not show up
	// in trash. Neither should show up on listObjects.
	parent1, child1, err := createParentChildObjectPair(user3)
	if err != nil {
		t.Errorf("Error creating parent-child pair: %v\n", err)
	}
	if err = d.DeleteObject(parent1, true); err != nil {
		t.Errorf("Error deleting test parent1: %v\n", err)
	}
	results, err = d.GetTrashedObjectsByUser("", 1, 1000, user3)
	if err != nil {
		t.Errorf("Error calling GetTrashedObjectsByUser: %v\n", err)
	}
	success = false
	// Assert parent is in trash.
	for _, o := range results.Objects {
		if o.Name == parent1.Name {
			success = true
		}
	}
	if !success {
		t.Error("Object parent1 is deleted but does not appear in trash")
	}
	// Assert child is not in trash.
	for _, o := range results.Objects {
		if o.Name == child1.Name {
			success = false
			t.Error("Object child1 should not be in user3 trash.")
		}
	}
	// Assert neither show up in listObjects.
	results, err = d.GetChildObjectsByUser("", 1, 1000, parent1, user3)

	for _, o := range results.Objects {
		if o.Name == child1.Name {
			success = false
			t.Error("Object child1 has a deleted parent, should not show in GetChildObjectsByUser.")
		}
		if o.Name == parent1.Name {
			success = false
			t.Error("Object parent1 is deleted, should not show in GetChildObjectsByUser.")
		}
	}

	if !success {
		t.Log("Errors in test cases for parent1, child1 object pair.")
		t.Fail()
	}

	// Create parent-child relationship, then delete the child.
	// Child should show up in trash. Parent should show up in
	// listObjects.
	success = false
	parent2, child2, err := createParentChildObjectPair(user3)
	if err != nil {
		t.Errorf("Error creating parent-child pair: %v\n", err)
	}
	if err = d.DeleteObject(child2, true); err != nil {
		t.Errorf("Error deleting test parent2: %v\n", err)
	}
	// Assert child2 is in trash.
	results, err = d.GetTrashedObjectsByUser("", 1, 1000, user3)
	for _, o := range results.Objects {
		if o.Name == child2.Name {
			success = true
		}
	}
	// Assert parent2 is in listObjects.
	results, err = d.GetChildObjectsByUser("", 1, 1000, parent1, user3)
	for _, o := range results.Objects {
		if o.Name == parent2.Name {
			success = true
		}
	}
	if !success {
		t.Error("Object parent2 should show up in GetChildObjectsByUser.")
	}
	// Assert child2 is not in listObjects.
	for _, o := range results.Objects {
		if o.Name == child2.Name {
			t.Error("Object child2 is deleted, should not show up in GetChildObjectsByUser.")
		}
	}

}

// createParentChildObjectPair creates a parent object at root and a child of that parent.
// Internally it delegates to createTestObjectAllPermissions.
func createParentChildObjectPair(username string) (parent models.ODObject, child models.ODObject, err error) {

	parent = createTestObjectAllPermissions(username)
	child = createTestObjectAllPermissions(username)
	parent, err = d.CreateObject(&parent, nil)
	if err != nil {
		return parent, child, err
	}
	child.ParentID = parent.ID
	child, err = d.CreateObject(&child, nil)
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

	var perms models.ODObjectPermission
	perms.CreatedBy = username
	perms.Grantee = username
	perms.AllowCreate, perms.AllowDelete, perms.AllowRead, perms.AllowUpdate = true, true, true, true

	obj.Permissions = append(obj.Permissions, perms)
	return obj

}
