package mapping

import (
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func MapODObjectToDeletedObjectResponse(i *models.ODObject) protocol.DeletedObjectResponse {
	o := protocol.DeletedObjectResponse{}
	o.DeletedDate = i.DeletedDate.Time
	return o
}
