package dao_test

import "testing"

func TestDAOUndeleteObjectWithChildren(t *testing.T) {

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
	folder1 := NewObjectWithPermissionsAndProperties(usernames[1], "Folder")
	folderType, err := d.GetObjectTypeByName(folder1.TypeName.String, true, folder1.CreatedBy)
	if err != nil {
		t.Error(err)
	} else {
		folder1.TypeID = folderType.ID
	}
	folder1, err = d.CreateObject(&folder1)

	// Set up children of folder1.
	objA := NewObjectWithPermissionsAndProperties(usernames[1], "File")
	fileType, err := d.GetObjectTypeByName(objA.TypeName.String, true, objA.CreatedBy)
	if err != nil {
		t.Error(err)
	} else {
		objA.TypeID = fileType.ID
	}
	folder1, objA, err = CreateParentChildObjectRelationship(folder1, objA)

	objB := NewObjectWithPermissionsAndProperties(usernames[1], "File")
	objB.TypeID = fileType.ID
	folder1, objB, err = CreateParentChildObjectRelationship(folder1, objB)

	folder2 := NewObjectWithPermissionsAndProperties(usernames[1], "Folder")
	folder2.TypeID = folderType.ID
	folder1, folder2, err = CreateParentChildObjectRelationship(folder1, folder2)

	objA, err = d.CreateObject(&objA)
	objB, err = d.CreateObject(&objB)
	folder2, err = d.CreateObject(&folder2)
	if err != nil {
		t.Errorf("Error creating folder1 and children: %v\n ", err)
	}

	// folder2 already created. Set up children of folder2.
	objC := NewObjectWithPermissionsAndProperties(usernames[1], "File")
	objC.TypeID = fileType.ID
	folder2, objC, err = CreateParentChildObjectRelationship(folder2, objC)

	folder3 := NewObjectWithPermissionsAndProperties(usernames[1], "Folder")
	folder3.TypeID = folderType.ID
	folder2, folder3, err = CreateParentChildObjectRelationship(folder2, folder3)

	objE := NewObjectWithPermissionsAndProperties(usernames[1], "File")
	objE.TypeID = fileType.ID
	folder2, objE, err = CreateParentChildObjectRelationship(folder2, objE)

	objC, err = d.CreateObject(&objC)
	folder3, err = d.CreateObject(&folder3)
	objE, err = d.CreateObject(&objE)
	if err != nil {
		t.Errorf("Error creating folder2 children: %v\n ", err)
	}

	// folder3 already created. Set up children of folder3.
	objD := NewObjectWithPermissionsAndProperties(usernames[1], "File")
	objD.TypeID = fileType.ID
	folder3, objD, err = CreateParentChildObjectRelationship(folder3, objD)

	objD, err = d.CreateObject(&objD)
	if err != nil {
		t.Errorf("Error creating folder3 children: %v\n ", err)
	}

	// Explicit delete folder3 and objC
	explicitDelete := true
	err = d.DeleteObject(users[1], folder3, explicitDelete)
	if err != nil {
		t.Errorf("Error deleting folder3: %v\n", err)
	}
	err = d.DeleteObject(users[1], objC, explicitDelete)
	if err != nil {
		t.Errorf("Error deleting objC: %v\n", err)
	}

	// Explicit delete folder1
	err = d.DeleteObject(users[1], folder1, explicitDelete)
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
