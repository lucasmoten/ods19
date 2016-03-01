package mapping

import (
	"encoding/hex"
	"fmt"

	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

// MapODPropertyToProperty converts an ODObjectPropertyEx from internal model
// format to exposable API protocol format
func MapODPropertyToProperty(i *models.ODObjectPropertyEx) protocol.Property {
	o := protocol.Property{}
	o.ID = hex.EncodeToString(i.ID)
	o.CreatedDate = i.CreatedDate
	o.CreatedBy = i.CreatedBy
	o.ModifiedDate = i.ModifiedDate
	o.ModifiedBy = i.ModifiedBy
	o.ChangeCount = i.ChangeCount
	o.ChangeToken = i.ChangeToken
	o.Name = i.Name
	if i.Value.Valid {
		o.Value = i.Value.String
	} else {
		o.Value = ""
	}
	if i.ClassificationPM.Valid {
		o.ClassificationPM = i.ClassificationPM.String
	} else {
		o.ClassificationPM = ""
	}
	return o
}

// MapODPropertiesToProperties converts an array of ODObjectPropertyEx struct
// from internal model format to exposable API protocol format
func MapODPropertiesToProperties(i *[]models.ODObjectPropertyEx) []protocol.Property {
	o := make([]protocol.Property, len(*i))
	for p, q := range *i {
		o[p] = MapODPropertyToProperty(&q)
	}
	return o
}

// MapPropertyToODProperty converts an exposable API protocol format of a
// Property to an internal ODObjectPropertyEx model
func MapPropertyToODProperty(i *protocol.Property) (models.ODObjectPropertyEx, error) {
	var err error
	o := models.ODObjectPropertyEx{}

	// ID convert string to byte, reassign to nil if empty
	ID, err := hex.DecodeString(i.ID)
	if err != nil {
		return o, fmt.Errorf("Unable to decode property id from %s", i.ID)
	}
	if len(o.ID) == 0 {
		o.ID = nil
	} else {
		o.ID = ID
	}

	o.CreatedDate = i.CreatedDate
	o.CreatedBy = i.CreatedBy
	o.ModifiedDate = i.ModifiedDate
	o.ModifiedBy = i.ModifiedBy
	o.ChangeCount = i.ChangeCount
	o.ChangeToken = i.ChangeToken
	o.Name = i.Name
	o.Value.Valid = true
	o.Value.String = i.Value
	o.ClassificationPM.Valid = true
	o.ClassificationPM.String = i.ClassificationPM
	return o, nil
}

// MapPropertiesToODProperties converts an array of exposable API protocol
// format of properties into an array of internally usable ODObjectPropertyEx
// models
func MapPropertiesToODProperties(i *[]protocol.Property) ([]models.ODObjectPropertyEx, error) {
	o := make([]models.ODObjectPropertyEx, len(*i))
	for p, q := range *i {
		mappedProperty, err := MapPropertyToODProperty(&q)
		if err != nil {
			return o, err
		}
		o[p] = mappedProperty
	}
	return o, nil
}
