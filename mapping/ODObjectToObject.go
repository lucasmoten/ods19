package mapping

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"decipher.com/object-drive-server/utils"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
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
	o.DeletedDate = i.DeletedDate.Time
	o.DeletedBy = i.DeletedBy.String
	o.ChangeCount = i.ChangeCount
	o.ChangeToken = i.ChangeToken

	o.OwnedBy = i.OwnedBy.String
	o.TypeID = hex.EncodeToString(i.TypeID)
	o.TypeName = i.TypeName.String
	o.Name = i.Name
	o.Description = i.Description.String

	o.ParentID = hex.EncodeToString(i.ParentID)
	if i.RawAcm.Valid {
		returnableACM, err := CreateReturnableACM(i.RawAcm.String)
		if err != nil {
			log.Printf("Couldnt convert rawacm string to object %v", err)
			o.RawAcm = ""
		} else {
			o.RawAcm = returnableACM
		}
	} else {
		o.RawAcm = ""
	}

	o.ContentType = i.ContentType.String

	if i.ContentSize.Valid {
		o.ContentSize = i.ContentSize.Int64
		o.ContentHash = hex.EncodeToString(i.ContentHash)
	}

	o.Properties = MapODPropertiesToProperties(&i.Properties)
	o.Permissions = MapODPermissionsToPermissions(&i.Permissions)
	o.IsPDFAvailable = i.IsPDFAvailable
	o.ContainsUSPersonsData = i.ContainsUSPersonsData
	o.ExemptFromFOIA = i.ExemptFromFOIA
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
		returnableACM, err := CreateReturnableACM(i.RawAcm.String)
		if err != nil {
			log.Printf("Couldnt convert rawacm string to object %v", err)
			o.RawAcm = ""
		} else {
			o.RawAcm = returnableACM
		}
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
		o.ContentHash = hex.EncodeToString(i.ContentHash)
	} else {
		o.ContentSize = 0
	}
	o.Properties = MapODPropertiesToProperties(&i.Properties)
	o.Permissions = MapODPermissionsToPermissions(&i.Permissions)
	o.IsPDFAvailable = i.IsPDFAvailable
	o.ContainsUSPersonsData = i.ContainsUSPersonsData
	o.ExemptFromFOIA = i.ExemptFromFOIA
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

// MapObjectToODObject converts an external protocol.Object to a models.ODObject.
func MapObjectToODObject(i *protocol.Object) (models.ODObject, error) {

	var err error
	o := models.ODObject{}
	o.ID, err = hex.DecodeString(i.ID)
	if err != nil {
		return o, fmt.Errorf("Unable to decode id from %s", i.ID)
	}
	if len(o.ID) == 0 {
		o.ID = nil
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
		return o, fmt.Errorf("Unable to decode type id from %s", i.TypeID)
	}
	if len(o.TypeID) == 0 {
		o.TypeID = nil
	}
	o.TypeName.Valid = true
	o.TypeName.String = i.TypeName
	o.Name = i.Name
	o.Description.Valid = true
	o.Description.String = i.Description

	if len(i.ParentID) > 0 {
		o.ParentID, err = hex.DecodeString(i.ParentID)
		if err != nil {
			return o, fmt.Errorf("Unable to decode parent id from %s", i.ParentID)
		}
		if len(o.ParentID) == 0 {
			o.ParentID = nil
		}
	} else {
		o.ParentID = nil
	}

	o.RawAcm.Valid = true
	o.RawAcm.String, err = utils.MarshalInterfaceToString(i.RawAcm)
	if err != nil {
		return o, fmt.Errorf("Unable to convert ACM to string %v", err)
	}
	o.ContentType.Valid = true
	o.ContentType.String = i.ContentType
	o.ContentSize.Valid = true
	o.ContentSize.Int64 = i.ContentSize
	// TODO: Is this really needed to be either exposed to user or received from user via Protocol?
	if len(i.ContentHash) > 0 {
		o.ContentHash, err = hex.DecodeString(i.ContentHash)
	}
	o.Properties, err = MapPropertiesToODProperties(&i.Properties)
	if err != nil {
		return o, err
	}
	o.Permissions, err = MapPermissionsToODPermissions(&i.Permissions)
	if err != nil {
		return o, err
	}
	o.ContainsUSPersonsData = i.ContainsUSPersonsData
	o.ExemptFromFOIA = i.ExemptFromFOIA
	return o, nil
}

// MapCreateObjectRequestToODObject converts an API exposable protocol object used for
// create requests into an internally usable model object
func MapCreateObjectRequestToODObject(i *protocol.CreateObjectRequest) (models.ODObject, error) {

	var err error
	o := models.ODObject{}
	o.TypeName = models.ToNullString(i.TypeName)
	o.Name = i.Name
	o.Description = models.ToNullString(i.Description)
	o.ParentID, err = hex.DecodeString(i.ParentID)
	if err != nil {
		return o, fmt.Errorf("Unable to decode parent id from %s", i.ParentID)
	}
	if len(o.ParentID) == 0 {
		o.ParentID = nil
	}
	o.RawAcm.Valid = true
	o.RawAcm.String, err = utils.MarshalInterfaceToString(i.RawAcm)
	if err != nil {
		return o, fmt.Errorf("Unable to convert ACM to string %v", err)
	}
	o.ContentType = models.ToNullString(i.ContentType)
	o.ContentSize.Valid = true
	o.ContentSize.Int64 = i.ContentSize
	o.Properties, err = MapPropertiesToODProperties(&i.Properties)
	if err != nil {
		return o, err
	}
	o.Permissions, err = MapObjectSharesToODPermissions(&i.Permissions)
	if err != nil {
		return o, err
	}
	o.ContainsUSPersonsData = i.ContainsUSPersonsData
	o.ExemptFromFOIA = i.ExemptFromFOIA

	return o, nil
}

// MapObjectsToODObjects converts an array of API exposable protocol Objects
// into an array of internally usable model Objects
func MapObjectsToODObjects(i *[]protocol.Object) ([]models.ODObject, error) {
	o := make([]models.ODObject, len(*i))
	for p, q := range *i {
		mappedObject, err := MapObjectToODObject(&q)
		if err != nil {
			return o, err
		}
		o[p] = mappedObject
	}
	return o, nil
}

// OverwriteODObjectWithCreateObjectRequest takes a CreateObjectRequest as input
// and maps its fields values over an existing ODObject. Only fields defined in
// the CreateObjectRequest are valid for mapping during creating objects
func OverwriteODObjectWithCreateObjectRequest(o *models.ODObject, i *protocol.CreateObjectRequest) error {
	o.TypeName = models.ToNullString(i.TypeName)
	o.Name = i.Name
	o.Description = models.ToNullString(i.Description)

	// Parent ID convert string to byte, reassign to nil if empty
	parentID, err := hex.DecodeString(i.ParentID)
	switch {
	case err != nil:
		if len(i.ParentID) > 0 {
			log.Printf("Unable to decode parent id")
			return err
		}
	case len(parentID) == 0:
		////If the i.id being sent in is blank, that's a signal to NOT use it
	default:
		o.ParentID = parentID
	}

	parsedACM, err := utils.MarshalInterfaceToString(i.RawAcm)
	if err != nil {
		return fmt.Errorf("Unable to convert ACM: %v", err)
	}
	o.RawAcm = models.ToNullString(parsedACM)

	o.ContentType = models.ToNullString(i.ContentType)
	o.ContentSize.Int64 = i.ContentSize

	o.Properties, err = MapPropertiesToODProperties(&i.Properties)
	if err != nil {
		return err
	}

	o.Permissions, err = MapObjectSharesToODPermissions(&i.Permissions)
	if err != nil {
		return err
	}

	o.ContainsUSPersonsData = i.ContainsUSPersonsData
	o.ExemptFromFOIA = i.ExemptFromFOIA
	return nil
}

// OverwriteODObjectWithUpdateObjectAndStreamRequest takes a
// UpdateObjectAndStreamRequest as input and maps its fields values over an
// existing ODObject. Only fields defined in the UpdateObjectAndStreamRequest
// are valid for mapping during updating objects
func OverwriteODObjectWithUpdateObjectAndStreamRequest(o *models.ODObject, i *protocol.UpdateObjectAndStreamRequest) error {

	// ID convert string to byte, reassign to nil if empty
	id, err := hex.DecodeString(i.ID)
	switch {
	case err != nil:
		log.Printf("Unable to decode id")
		return err
	case len(id) == 0:
		////If the i.id being sent in is blank, that's a signal to NOT use it
	default:
		o.ID = id
	}

	// Type ID convert string to byte, reassign to nil if empty
	typeID, err := hex.DecodeString(i.TypeID)
	switch {
	case err != nil:
		if len(i.TypeID) > 0 {
			log.Printf("Unable to decode type id")
			return err
		}
	case len(typeID) == 0:
		o.TypeID = nil
	default:
		o.TypeID = typeID
	}

	if len(i.TypeName) > 0 {
		o.TypeName = models.ToNullString(i.TypeName)
	}

	if len(i.Name) > 0 {
		o.Name = i.Name
	}

	if len(i.Description) > 0 {
		o.Description = models.ToNullString(i.Description)
	}

	parsedACM, err := utils.MarshalInterfaceToString(i.RawAcm)
	if err != nil {
		return fmt.Errorf("Unable to convert ACM: %v", err)
	}
	if len(parsedACM) > 0 {
		o.RawAcm = models.ToNullString(parsedACM)
	}

	if len(i.ContentType) > 0 {
		o.ContentType = models.ToNullString(i.ContentType)
	}

	o.ContentSize.Int64 = i.ContentSize

	o.Properties, err = MapPropertiesToODProperties(&i.Properties)
	if err != nil {
		return err
	}

	o.ContainsUSPersonsData = i.ContainsUSPersonsData
	o.ExemptFromFOIA = i.ExemptFromFOIA

	return nil
}

// CreateReturnableACM is a wrapper for the return type.
// The original API worked with ACMs as serialized strings.
// This allows for switching between the string or object representation on the response
func CreateReturnableACM(acm string) (interface{}, error) {
	//return acm, nil
	return utils.UnmarshalStringToInterface(acm)
}