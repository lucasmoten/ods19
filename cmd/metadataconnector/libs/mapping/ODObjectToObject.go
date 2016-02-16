package mapping

import (
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func mapODObjectToObject(i *models.ODObject) protocol.Object {
	var o protocol.Object
	o.ID = i.ID
	o.CreatedDate = i.CreatedDate
	o.CreatedBy = i.CreatedBy
	o.ModifiedDate = i.ModifiedDate
	o.ModifiedBy = i.ModifiedBy
	o.ChangeCount = i.ChangeCount
	o.ChangeToken = i.ChangeToken
	if i.OwnedBy.Valid {
		o.OwnedBy = i.OwnedBy.String
	} else {
		o.OwnedBy = ""
	}
	o.TypeID = i.TypeID
	if i.TypeName.Valid {
		o.TypeName = i.TypeName.String
	} else {
		o.TypeName = ""
	}
	o.Name = i.Name
	if i.Description.Valid {
		o.Description = i.Description.String
	} else {
		o.Description = ""
	}
	o.ParentID = i.ParentID
	if i.RawAcm.Valid {
		o.RawAcm = i.RawAcm.String
	} else {
		o.RawAcm = ""
	}
	if i.ContentType.Valid {
		o.ContentType = i.ContentType.String
	} else {
		o.ContentType = ""
	}
	if i.ContentSize.Valid {
		o.ContentSize = i.ContentSize.Int64
	} else {
		o.ContentSize = 0
	}
	//o.Properties = mapODPropertiesToProperties(i.Properties)
	//o.Permissions = mapODPermissionsToPermissions(i.Permissions)
	return o
}

func mapODObjectsToObjects(i *[]models.ODObject) []protocol.Object {
	var o [len(i)]protocol.Object
	return o
}
