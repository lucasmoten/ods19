package mapping

import (
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/protocol"
)

// MapODObjectToExpungedObjectResponse converts an internal ODObject model
// object into an API exposable protocol response object specific to expunged
// objects
func MapODObjectToExpungedObjectResponse(i *models.ODObject) protocol.ExpungedObjectResponse {
	o := protocol.ExpungedObjectResponse{}
	o.ExpungedDate = i.ExpungedDate.Time
	return o
}
