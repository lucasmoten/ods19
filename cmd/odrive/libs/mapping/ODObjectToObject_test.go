package mapping_test

import (
	"testing"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

func TestMapCreateObjectRequestToODObject(t *testing.T) {

	input := protocol.CreateObjectRequest{
		Name:        "Test",
		ParentID:    "",
		RawAcm:      "{}",
		ContentType: "text/plain",
		ContentSize: 1024,
	}
	result, err := mapping.MapCreateObjectRequestToODObject(&input)

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
