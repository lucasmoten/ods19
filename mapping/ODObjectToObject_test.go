package mapping_test

import (
	"testing"

	"github.com/deciphernow/object-drive-server/mapping"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/protocol"
	"github.com/deciphernow/object-drive-server/util"
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
