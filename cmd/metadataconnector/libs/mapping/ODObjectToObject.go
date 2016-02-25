package mapping

import (
	"encoding/hex"
	"encoding/json"
	"log"

	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

// MapODObjectToObject converts an internal ODObject model object into an API
// exposable protocol Object
func MapODObjectToObject(i *models.ODObject) protocol.Object {
	o := protocol.Object{}
	o.ID = hex.EncodeToString(i.ID)
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
	o.TypeID = hex.EncodeToString(i.TypeID)
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
	o.ParentID = hex.EncodeToString(i.ParentID)
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

// MapODObjectToDeletedObject converts an internal ODObject model object into an
// API exposable protocol DeletedObject
func MapODObjectToDeletedObject(i *models.ODObject) protocol.DeletedObject {
	o := protocol.DeletedObject{}
	o.ID = hex.EncodeToString(i.ID)
	o.CreatedDate = i.CreatedDate
	o.CreatedBy = i.CreatedBy
	o.ModifiedDate = i.ModifiedDate
	o.ModifiedBy = i.ModifiedBy
	o.DeletedDate = i.DeletedDate.Time
	o.DeletedBy = i.DeletedBy.String
	o.ChangeCount = i.ChangeCount
	o.ChangeToken = i.ChangeToken
	if i.OwnedBy.Valid {
		o.OwnedBy = i.OwnedBy.String
	} else {
		o.OwnedBy = ""
	}
	o.TypeID = hex.EncodeToString(i.TypeID)
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
	o.ParentID = hex.EncodeToString(i.ParentID)
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

// MapODObjectToJSON writes a database object to a json representation,
// which is mostly useful in error messages
func MapODObjectToJSON(i *models.ODObject) string {
	o := MapODObjectToObject(i)
	jsonobj, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		log.Printf("Unable to marshal object to json:%v", err)
	}
	return string(jsonobj)
}

// MapObjectToODObject converts an API exposable protocol Object into an
// internally usable model object.
func MapObjectToODObject(i *protocol.Object) models.ODObject {
	var err error
	o := models.ODObject{}
	o.ID, err = hex.DecodeString(i.ID)
	if err != nil {
		log.Printf("Unable to decode id")
	}
	o.CreatedDate = i.CreatedDate
	o.CreatedBy = i.CreatedBy
	o.ModifiedDate = i.ModifiedDate
	o.ModifiedBy = i.ModifiedBy
	o.ChangeCount = i.ChangeCount
	o.ChangeToken = i.ChangeToken
	o.OwnedBy.Valid = true
	o.OwnedBy.String = i.OwnedBy
	o.TypeID, err = hex.DecodeString(i.TypeID)
	if err != nil {
		log.Printf("Unable to decode type id")
	}
	o.TypeName.Valid = true
	o.TypeName.String = i.TypeName
	o.Name = i.Name
	o.Description.Valid = true
	o.Description.String = i.Description
	o.ParentID, err = hex.DecodeString(i.ParentID)
	if err != nil {
		log.Printf("Unable to decode parent id")
	}
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

// TypeName string `json:"typeName"`
// Name     string `json:"name"`
// ParentID string `json:"parentId,omitempty"`
// RawAcm string `json:"acm"`
// ContentType string `json:"contentType"`
// ContentSize int64 `json:"contentSize"`

// MapCreateObjectRequestToODObject ...
func MapCreateObjectRequestToODObject(i *protocol.CreateObjectRequest) models.ODObject {

	var err error
	o := models.ODObject{}
	o.TypeName.Valid = true
	o.TypeName.String = i.TypeName
	o.Name = i.Name
	o.ParentID, err = hex.DecodeString(i.ParentID)
	if err != nil {
		log.Printf("Unable to decode parent id")
	}
	o.RawAcm.Valid = true
	o.RawAcm.String = i.RawAcm
	o.ContentType.Valid = true
	o.ContentType.String = i.ContentType
	o.ContentSize.Valid = true
	o.ContentSize.Int64 = i.ContentSize
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

// OverwriteODObjectWithProtocolObject ...
// When we get a decoded json object, for uploads, we have specific items that
// we should extract and write over the object that we have
func OverwriteODObjectWithProtocolObject(o *models.ODObject, i *protocol.Object) error {
	id, err := hex.DecodeString(i.ID)
	if err != nil {
		log.Printf("Count not decode id")
		return err
	}
	if len(id) > 0 {
		o.ID = id
	}

	pid, err := hex.DecodeString(i.ParentID)
	if err != nil {
		log.Printf("Count not decode parent id")
		return err
	}
	if len(pid) > 0 {
		o.ParentID = pid
	}
	if len(o.ParentID) == 0 {
		o.ParentID = nil
	}

	o.ContentSize.Int64 = i.ContentSize
	if i.Name != "" {
		o.Name = i.Name
	}

	o.ContentType.String = i.ContentType
	o.RawAcm.String = i.RawAcm
	o.TypeName.String = i.TypeName

	return nil
}
