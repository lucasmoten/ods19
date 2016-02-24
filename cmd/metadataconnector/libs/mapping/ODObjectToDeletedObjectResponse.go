package mapping

import (
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

// MapODObjectToDeletedObjectResponse converts an internal ODObject model object
// into an API exposable protocol response object specific to deleted objects
func MapODObjectToDeletedObjectResponse(i *models.ODObject) protocol.DeletedObjectResponse {
	o := protocol.DeletedObjectResponse{}
	o.DeletedDate = i.DeletedDate.Time
	return o
}
