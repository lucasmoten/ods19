package mapping_test

import (
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/mapping"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/util"
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
