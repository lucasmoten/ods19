package mapping

import (
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

// MapODObjectToDeletedObjectResponse converts an internal ODObject model object
// into an API exposable protocol response object specific to deleted objects
func MapODObjectToDeletedObjectResponse(i *models.ODObject) protocol.DeletedObjectResponse {
	o := protocol.DeletedObjectResponse{}
	o.DeletedDate = i.DeletedDate.Time
	return o
}
