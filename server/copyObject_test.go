package server_test

import (
	"testing"

	"github.com/deciphernow/object-drive-server/protocol"
)

func TestCopyObject(t *testing.T) {
	tester10 := 0

	create1 := protocol.CreateObjectRequest{
		Name:     "TestCopyObject",
		RawAcm:   `{"version":"2.1.0","classif":"U"}`,
		TypeName: "TestObject",
	}
	res1, _ := clients[tester10].C.CreateObject(create1, nil)

	update2 := protocol.UpdateObjectRequest{
		ID:          res1.ID,
		ChangeToken: res1.ChangeToken,
		Name:        "TestCopyObject",
		RawAcm:      `{"version":"2.1.0","classif":"U"}`,
		TypeName:    "TestObject",
	}
	res2, _ := clients[tester10].C.UpdateObject(update2)

	copy3 := protocol.CopyObjectRequest{
		ID: res1.ID,
	}
	res3, _ := clients[tester10].C.CopyObject(copy3)

	if res3.ID == res1.ID {
		t.Errorf("ID of the copy is the same as the original object")
	}
	if res3.ChangeCount != res2.ChangeCount {
		t.Errorf("The number of revisions on the copy (%d) is not the expected count (%d)", res3.ChangeCount, res2.ChangeCount)
	}

}
