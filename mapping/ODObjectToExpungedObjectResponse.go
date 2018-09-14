package mapping

import (
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
)

// MapODObjectToExpungedObjectResponse converts an internal ODObject model
// object into an API exposable protocol response object specific to expunged
// objects
func MapODObjectToExpungedObjectResponse(i *models.ODObject) protocol.ExpungedObjectResponse {
	o := protocol.ExpungedObjectResponse{}
	o.ExpungedDate = i.ExpungedDate.Time
	return o
}
