package mapping

import (
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
	"encoding/hex"
	"log"
)

// MapODPermissionToPermission converts an internal ODPermission model to an
// API exposable Permission
func MapODPermissionToPermission(i *models.ODObjectPermission) protocol.Permission {
	o := protocol.Permission{}
	o.ID = hex.EncodeToString(i.ID)
	o.CreatedDate = i.CreatedDate
	o.CreatedBy = i.CreatedBy
	o.ModifiedDate = i.ModifiedDate
	o.ModifiedBy = i.ModifiedBy
	o.ChangeCount = i.ChangeCount
	o.ChangeToken = i.ChangeToken
	o.ObjectID = hex.EncodeToString(i.ObjectID)
	o.Grantee = i.Grantee
	o.AllowCreate = i.AllowCreate
	o.AllowRead = i.AllowRead
	o.AllowUpdate = i.AllowUpdate
	o.AllowDelete = i.AllowDelete
	return o
}

// MapODPermissionsToPermissions converts an array of internal ODPermission
// models to an array of API exposable Permission
func MapODPermissionsToPermissions(i *[]models.ODObjectPermission) []protocol.Permission {
	o := make([]protocol.Permission, len(*i))
	for p, q := range *i {
		o[p] = MapODPermissionToPermission(&q)
	}
	return o
}

// MapPermissionToODPermission converts an API exposable Permission object to
// an internally usable ODPermission model
func MapPermissionToODPermission(i *protocol.Permission) models.ODObjectPermission {
	var err error
	o := models.ODObjectPermission{}
	o.ID, err = hex.DecodeString(i.ID)
	if err != nil {
		log.Printf("Unable to decode permission id")
	}
	o.CreatedDate = i.CreatedDate
	o.CreatedBy = i.CreatedBy
	o.ModifiedDate = i.ModifiedDate
	o.ModifiedBy = i.ModifiedBy
	o.ChangeCount = i.ChangeCount
	o.ChangeToken = i.ChangeToken
	o.ObjectID, err = hex.DecodeString(i.ObjectID)
	if err != nil {
		log.Printf("Unable to decode object id")
	}
	o.Grantee = i.Grantee
	o.AllowCreate = i.AllowCreate
	o.AllowRead = i.AllowRead
	o.AllowUpdate = i.AllowUpdate
	o.AllowDelete = i.AllowDelete
	return o
}

// MapPermissionsToODPermissions converts an array of API exposable Permission
// objects into an array of internally usable ODPermission model objects
func MapPermissionsToODPermissions(i *[]protocol.Permission) []models.ODObjectPermission {
	o := make([]models.ODObjectPermission, len(*i))
	for p, q := range *i {
		o[p] = MapPermissionToODPermission(&q)
	}
	return o
}
