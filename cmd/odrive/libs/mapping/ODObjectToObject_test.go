package mapping_test

import (
	"encoding/hex"
	"testing"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

func TestOverwriteODObjectWithProtocolObject(t *testing.T) {

	var mo models.ODObject

	po := &protocol.Object{
		ID:         randomString(),
		Name:       randomString(),
		ModifiedBy: randomString(),
	}

	// Convert the object.
	mapping.OverwriteODObjectWithProtocolObject(&mo, po)

	// Assert that the values are the same.
	if hex.EncodeToString(mo.ID) != po.ID {
		t.Errorf("IDs not the same\n\t %v \n\t %v", hex.EncodeToString(mo.ID), po.ID)
	}

	if po.Name != mo.Name {
		t.Errorf("field Name not mapped")
	}

	if mo.ModifiedBy == po.ModifiedBy {
		t.Errorf("field modifiedBy was mapped from protocol object")
	}

}

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
