package mapping

import (
	"encoding/hex"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
)

// MapODObjectToDeletedObjectResponse converts an internal ODObject model object
// into an API exposable protocol response object specific to deleted objects
func MapODObjectToDeletedObjectResponse(i *models.ODObject) protocol.DeletedObjectResponse {
	var o protocol.DeletedObjectResponse
	o.DeletedDate = i.DeletedDate.Time
	o.ID = hex.EncodeToString(i.ID)
	return o
}
