package dao_test

import (
	"testing"

	"decipher.com/object-drive-server/util/testhelpers"
)

func TestUndeleteObjectWithChildren(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	/* Create a structure like this:

	/folder1      <-- explicit delete 3, then undelete
	  objA
	  objB
	  /folder2
	    objC      <-- explicit delete 2
		/folder3  <-- explicit delete 1
		  objD
		objE
	*/

	// Create folder1.
	folder1 := testhelpers.NewObjectWithPermissionsAndProperties(usernames[1], "Folder")
	folder1, err := d.CreateObject(&folder1)

	// Set up children of folder1.
	objA := testhelpers.NewObjectWithPermissionsAndProperties(usernames[1], "File")
	folder1, objA, err = testhelpers.CreateParentChildObjectRelationship(folder1, objA)

	objB := testhelpers.NewObjectWithPermissionsAndProperties(usernames[1], "File")
	folder1, objB, err = testhelpers.CreateParentChildObjectRelationship(folder1, objB)

	folder2 := testhelpers.NewObjectWithPermissionsAndProperties(usernames[1], "Folder")
	folder1, folder2, err = testhelpers.CreateParentChildObjectRelationship(folder1, folder2)

	objA, err = d.CreateObject(&objA)
	objB, err = d.CreateObject(&objB)
	folder2, err = d.CreateObject(&folder2)
	if err != nil {
		t.Errorf("Error creating folder1 and children: %v\n ", err)
	}

	// folder2 already created. Set up children of folder2.
	objC := testhelpers.NewObjectWithPermissionsAndProperties(usernames[1], "File")
	folder2, objC, err = testhelpers.CreateParentChildObjectRelationship(folder2, objC)

	folder3 := testhelpers.NewObjectWithPermissionsAndProperties(usernames[1], "Folder")
	folder2, folder3, err = testhelpers.CreateParentChildObjectRelationship(folder2, folder3)

	objE := testhelpers.NewObjectWithPermissionsAndProperties(usernames[1], "File")
	folder2, objE, err = testhelpers.CreateParentChildObjectRelationship(folder2, objE)

	objC, err = d.CreateObject(&objC)
	folder3, err = d.CreateObject(&folder3)
	objE, err = d.CreateObject(&objE)
	if err != nil {
		t.Errorf("Error creating folder2 children: %v\n ", err)
	}

	// folder3 already created. Set up children of folder3.
	objD := testhelpers.NewObjectWithPermissionsAndProperties(usernames[1], "File")
	folder3, objD, err = testhelpers.CreateParentChildObjectRelationship(folder3, objD)

	objD, err = d.CreateObject(&objD)
	if err != nil {
		t.Errorf("Error creating folder3 children: %v\n ", err)
	}

	// Explicit delete folder3 and objC
	user := setupUserWithSnippets(usernames[1])
	explicitDelete := true
	err = d.DeleteObject(user, folder3, explicitDelete)
	if err != nil {
		t.Errorf("Error deleting folder3: %v\n", err)
	}
	err = d.DeleteObject(user, objC, explicitDelete)
	if err != nil {
		t.Errorf("Error deleting objC: %v\n", err)
	}

	// Explicit delete folder1
	err = d.DeleteObject(user, folder1, explicitDelete)
	if err != nil {
		t.Errorf("Error deleting folder1: %v\n", err)
	}

	// Get objE and assert IsAncestorDeleted is true.
	objE, err = d.GetObject(objE, false)
	if err != nil {
		t.Errorf("Could not get objE: %v\n", err)
	}
	if !objE.IsAncestorDeleted {
		t.Errorf("Expected objE IsAncestorDeleted to be true. Got: %v\n",
			objE.IsAncestorDeleted)
	}

	// deletes.
	folder1, err = d.UndeleteObject(&folder1)
	if err != nil {
		t.Errorf("Error calling UndeleteObject on folder1: %v\n", err)
	}
	if folder1.IsDeleted {
		t.Errorf("Expected folder1 IsDeleted to be false. Got: %v\n", folder1.IsDeleted)
	}

	// Get objE and assert IsAncestorDeleted is false.
	objE, err = d.GetObject(objE, false)
	if err != nil {
		t.Errorf("Could not get objE: %v\n", err)
	}
	if objE.IsAncestorDeleted {
		t.Errorf("Expected objE IsAncestorDeleted to be false after undelete of parent. Got: %v\n",
			objE.IsAncestorDeleted)
	}

	// Get objC and assert IsDeleted is true and IsAncestorDeleted is false.
	objC, err = d.GetObject(objC, false)
	if err != nil {
		t.Errorf("Could not get objC: %v\n", err)
	}
	if objC.IsAncestorDeleted {
		t.Errorf("Expected objC IsAncestorDeleted to be false. Got: %v\n",
			objC.IsAncestorDeleted)
	}
	if !objC.IsDeleted {
		t.Errorf("Expected objC IsDeleted to be true. Got: %v\n",
			objC.IsDeleted)
	}

}
