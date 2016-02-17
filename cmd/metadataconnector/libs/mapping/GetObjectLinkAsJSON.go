package mapping

import (
	"encoding/hex"
	"time"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

// GetObjectLinkFromObject - given an object from the database, render it back to the user as json
func GetObjectLinkFromObject(rootURL string, object *models.ODObject) protocol.ObjectLink {
	link := protocol.ObjectLink{
		URL:         rootURL + "/object/" + hex.EncodeToString(object.ID),
		Name:        object.Name,
		Type:        object.TypeName.String,
		CreateDate:  object.CreatedDate.Format(time.RFC3339),
		CreatedBy:   config.GetCommonName(object.CreatedBy),
		Size:        object.ContentSize.Int64,
		ACM:         object.RawAcm.String,
		ChangeToken: object.ChangeToken,
	}
	return link
}
