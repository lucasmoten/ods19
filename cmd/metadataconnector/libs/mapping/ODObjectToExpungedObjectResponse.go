package mapping

import (
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func MapODObjectToExpungedObjectResponse(i *models.ODObject) protocol.ExpungedObjectResponse {
	o := protocol.ExpungedObjectResponse{}
	o.ExpungedDate = i.ExpungedDate.Time
	return o
}
