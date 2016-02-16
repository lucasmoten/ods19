package mapping

import (
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func mapODPermissionToPermission(i models.ODObjectPermission) protocol.Permission {
	o := protocol.Permission{}
	o.ID = i.ID
	o.CreatedDate = i.CreatedDate
	o.CreatedBy = i.CreatedBy
	o.ModifiedDate = i.ModifiedDate
	o.ModifiedBy = i.ModifiedBy
	o.ChangeCount = i.ChangeCount
	o.ChangeToken = i.ChangeToken
	o.ObjectID = i.ObjectID
	o.Grantee = i.Grantee
	o.AllowCreate = i.AllowCreate
	o.AllowRead = i.AllowRead
	o.AllowUpdate = i.AllowUpdate
	o.AllowDelete = i.AllowDelete
	return o
}

func mapODPermissionsToPermissions(i []models.ODObjectPermission) []protocol.Permission {
	o := make([]protocol.Permission, len(i))
	for p, q := range i {
		o[p] = mapODPermissionToPermission(q)
	}
	return o
}

func mapPermissionToODPermission(i protocol.Permission) models.ODObjectPermission {
	o := models.ODObjectPermission{}
	o.ID = i.ID
	o.CreatedDate = i.CreatedDate
	o.CreatedBy = i.CreatedBy
	o.ModifiedDate = i.ModifiedDate
	o.ModifiedBy = i.ModifiedBy
	o.ChangeCount = i.ChangeCount
	o.ChangeToken = i.ChangeToken
	o.ObjectID = i.ObjectID
	o.Grantee = i.Grantee
	o.AllowCreate = i.AllowCreate
	o.AllowRead = i.AllowRead
	o.AllowUpdate = i.AllowUpdate
	o.AllowDelete = i.AllowDelete
	return o
}

func mapPermissionsToODPermissions(i []protocol.Permission) []models.ODObjectPermission {
	o := make([]models.ODObjectPermission, len(i))
	for p, q := range i {
		o[p] = mapPermissionToODPermission(q)
	}
	return o
}
