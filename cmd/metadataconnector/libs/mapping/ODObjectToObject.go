package mapping

import (
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func mapODObjectToObject(i models.ODObject) protocol.Object {
	o := protocol.Object{}
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
	o.Properties = mapODPropertiesToProperties(i.Properties)
	o.Permissions = mapODPermissionsToPermissions(i.Permissions)
	return o
}

func mapODObjectsToObjects(i []models.ODObject) []protocol.Object {
	o := make([]protocol.Object, len(i))
	for p, iobj := range i {
		o[p] = mapODObjectToObject(iobj)
	}
	return o
}

func mapObjectToODObject(i protocol.Object) models.ODObject {
	o := models.ODObject{}
	o.ID = i.ID
	o.CreatedDate = i.CreatedDate
	o.CreatedBy = i.CreatedBy
	o.ModifiedDate = i.ModifiedDate
	o.ModifiedBy = i.ModifiedBy
	o.ChangeCount = i.ChangeCount
	o.ChangeToken = i.ChangeToken
	o.OwnedBy.Valid = true
	o.OwnedBy.String = i.OwnedBy
	o.TypeID = i.TypeID
	o.TypeName.Valid = true
	o.TypeName.String = i.TypeName
	o.Name = i.Name
	o.Description.Valid = true
	o.Description.String = i.Description
	o.ParentID = i.ParentID
	o.RawAcm.Valid = true
	o.RawAcm.String = i.RawAcm
	o.ContentType.Valid = true
	o.ContentType.String = i.ContentType
	o.ContentSize.Valid = true
	o.ContentSize.Int64 = i.ContentSize
	o.Properties = mapPropertiesToODProperties(i.Properties)
	o.Permissions = mapPermissionsToODPermissions(i.Permissions)
	return o
}

func mapObjectsToODObjects(i []protocol.Object) []models.ODObject {
	o := make([]models.ODObject, len(i))
	for p, iobj := range i {
		o[p] = mapObjectToODObject(iobj)
	}
	return o
}
