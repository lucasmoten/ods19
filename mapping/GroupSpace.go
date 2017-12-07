package mapping

import (
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/protocol"
)

// MapDAOGroupSpaceToProtocolGroupSpace converts an internal DAO GroupSpace model object into an API exposable protocol GroupSpace
func MapDAOGroupSpaceToProtocolGroupSpace(i *models.GroupSpace) protocol.GroupSpace {
	o := protocol.GroupSpace{}
	o.Grantee = i.Grantee
	o.ResourceString = i.ResourceString
	o.DisplayName = i.DisplayName
	o.Quantity = i.Quantity
	return o
}

// MapDAOGroupSpacesToProtocolGroupSpaces converts an internal DAO GroupSpace array into an API exposable protocol GroupSpace array
func MapDAOGroupSpacesToProtocolGroupSpaces(i *[]models.GroupSpace) []protocol.GroupSpace {
	o := make([]protocol.GroupSpace, len(*i))
	for p, q := range *i {
		o[p] = MapDAOGroupSpaceToProtocolGroupSpace(&q)
	}
	return o
}

// MapDAOGroupSpaceRSToProtocolGroupSpaceRS converts an internal DAO GroupSpace Resultset into an API exposable protocl GroupSpace resultset
func MapDAOGroupSpaceRSToProtocolGroupSpaceRS(i *models.GroupSpaceResultset) protocol.GroupSpaceResultset {
	o := protocol.GroupSpaceResultset{}
	o.Resultset.TotalRows = i.Resultset.TotalRows
	o.Resultset.PageCount = i.Resultset.PageCount
	o.Resultset.PageNumber = i.Resultset.PageNumber
	o.Resultset.PageSize = i.Resultset.PageSize
	o.Resultset.PageRows = i.Resultset.PageRows
	o.GroupSpaces = MapDAOGroupSpacesToProtocolGroupSpaces(&i.GroupSpaces)
	return o
}
