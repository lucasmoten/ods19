package mapping_test

import (
	"encoding/hex"
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
	"decipher.com/oduploader/util"
)

func TestOverwriteODObjectWithProtocolObject(t *testing.T) {

	someID, _ := util.NewGUID()
	po := &protocol.Object{ID: someID}
	mo := &models.ODObject{}

	// Convert the object.
	mapping.OverwriteODObjectWithProtocolObject(mo, po)

	// Assert that the values are the same.
	stringRepr := hex.EncodeToString(mo.ID)

	if stringRepr != someID {
		t.Errorf("IDs not the same\n\t %v \n\t %v", stringRepr, someID)
		t.Fail()
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
	result := mapping.MapCreateObjectRequestToODObject(&input)

	if result.Name != input.Name {
		t.Fail()
	}
}
