package mapping

import (
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

// MapODObjectToObject converts an internal ODObject model object into an API
// exposable protocol Object
func MapODObjectToObject(i *models.ODObject) protocol.Object {
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
	o.Properties = MapODPropertiesToProperties(&i.Properties)
	o.Permissions = MapODPermissionsToPermissions(&i.Permissions)
	return o
}

// MapODObjectsToObjects converts an array of internal ODObject model Objects
// into an array of API exposable protocol Objects
func MapODObjectsToObjects(i *[]models.ODObject) []protocol.Object {
	o := make([]protocol.Object, len(*i))
	for p, q := range *i {
		o[p] = MapODObjectToObject(&q)
	}
	return o
}

// MapODObjectResultsetToObjectResultset converts an internal resultset of
// ODObjects into a corresponding protocol resultset of Objects
func MapODObjectResultsetToObjectResultset(i *models.ODObjectResultset) protocol.ObjectResultset {
	o := protocol.ObjectResultset{}
	o.Resultset.TotalRows = i.Resultset.TotalRows
	o.Resultset.PageCount = i.Resultset.PageCount
	o.Resultset.PageNumber = i.Resultset.PageNumber
	o.Resultset.PageSize = i.Resultset.PageSize
	o.Resultset.PageRows = i.Resultset.PageRows
	o.Objects = MapODObjectsToObjects(&i.Objects)
	return o
}

// MapObjectToODObject converts an API exposable protocol Object into an
// internally usable model object.
func MapObjectToODObject(i *protocol.Object) models.ODObject {
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
	o.Properties = MapPropertiesToODProperties(&i.Properties)
	o.Permissions = MapPermissionsToODPermissions(&i.Permissions)
	return o
}

// MapObjectsToODObjects converts an array of API exposable protocol Objects
// into an array of internally usable model Objects
func MapObjectsToODObjects(i *[]protocol.Object) []models.ODObject {
	o := make([]models.ODObject, len(*i))
	for p, q := range *i {
		o[p] = MapObjectToODObject(&q)
	}
	return o
}
