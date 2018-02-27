package server_test

import (
	"strings"
	"testing"

	"github.com/deciphernow/object-drive-server/protocol"
)

func TestCopyObject(t *testing.T) {
	tester10 := 0
	tester1 := 1

	// "share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive"]}}}							// shared to the odrive group
	// "share":{"users":["cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]}	// private to tester10

	// 1. Create object private to tester10
	create1 := protocol.CreateObjectRequest{
		Name:     "TestCopyObject",
		RawAcm:   `{"version":"2.1.0","classif":"U","share":{"users":["cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]}}`,
		TypeName: "TestObject",
	}
	res1, err := clients[tester10].C.CreateObject(create1, nil)
	if err != nil {
		t.Errorf("Error creating object %s", err.Error())
	}

	// 2. Attempt copy as tester1, who can't read this object yet
	copy2 := protocol.CopyObjectRequest{
		ID: res1.ID,
	}
	_, err = clients[tester1].C.CopyObject(copy2)
	if err == nil {
		t.Errorf("Expected error when attempting to copy object without adequate read access")
	} else {
		// make sure we got an error for the right reasons ...
		if !strings.Contains(err.Error(), "Forbidden") {
			t.Errorf("Error encountered was not expected type. %s", err.Error())
		}
	}

	// 3. Update, renaming, and sharing to everyone
	update3 := protocol.UpdateObjectRequest{
		ID:          res1.ID,
		ChangeToken: res1.ChangeToken,
		Name:        "TestCopyObject-renamed",
		RawAcm:      `{"version":"2.1.0","classif":"U"}`, // ends up shared to everyone
		TypeName:    "TestObject",
	}
	res3, err := clients[tester10].C.UpdateObject(update3)
	if err != nil {
		t.Errorf("Error updating object %s", err.Error())
	}

	// 4. Update, set a description, and sharing to everyone
	update4 := protocol.UpdateObjectRequest{
		ID:          res1.ID,
		ChangeToken: res3.ChangeToken,
		Name:        "TestCopyObject-renamed",
		Description: "Tester10 will see all three revisions. Everyone else will only have access to 2",
		RawAcm:      `{"version":"2.1.0","classif":"U"}`, // ends up shared to everyone
		TypeName:    "TestObject",
	}
	res4, err := clients[tester10].C.UpdateObject(update4)
	if err != nil {
		t.Errorf("Error updating object %s", err.Error())
	}

	// 5. Copy the object, now that we have read
	copy5 := protocol.CopyObjectRequest{
		ID: res1.ID, // could just as easily be res3.ID or res4.ID since this value does not change
	}
	res5, err := clients[tester1].C.CopyObject(copy5)
	if err != nil {
		t.Errorf("Error copying object %s", err.Error())
	}

	// Some validation checks
	if res5.ID == res1.ID {
		t.Errorf("ID of the copy is the same as the original object")
	}
	if res5.ChangeCount != res4.ChangeCount-1 {
		t.Errorf("The number of revisions on the copy (%d) is not the expected count (%d)", res5.ChangeCount, res4.ChangeCount-1)
	}
	if res5.Name != res4.Name {
		t.Errorf("The name of the copied object (%s) doesn't match the final name of the original object (%s)", res5.Name, res4.Name)
	}

}
