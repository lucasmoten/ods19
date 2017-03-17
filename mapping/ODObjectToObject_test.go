package mapping_test

import (
	"testing"

	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

func TestOverwriteODObjectWithCreateObjectRequest(t *testing.T) {

	input := protocol.CreateObjectRequest{
		Name:        "Test",
		ParentID:    "",
		RawAcm:      "{}",
		ContentType: "text/plain",
		ContentSize: 1024,
	}
	var result models.ODObject
	err := mapping.OverwriteODObjectWithCreateObjectRequest(&result, &input)

	if err != nil {
		t.Fail()
	}

	if result.Name != input.Name {
		t.Fail()
	}
}

func randomString() string {
	s, _ := util.NewGUID()
	return s
}
