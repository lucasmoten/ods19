package mapping

import (
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
)

// MapODUserToUser ...
func MapODUserToUser(i *models.ODUser) protocol.User {
	o := protocol.User{}
	o.DistinguishedName = i.DistinguishedName
	o.DisplayName = i.DisplayName.String
	o.Email = i.Email.String
	return o
}

// MapODUsersToUsers converts an array of internal ODUsers model Users
// into an array of API exposable protocol Objects
func MapODUsersToUsers(i *[]models.ODUser) []protocol.User {
	o := make([]protocol.User, len(*i))
	for p, q := range *i {
		o[p] = MapODUserToUser(&q)
	}
	return o
}
