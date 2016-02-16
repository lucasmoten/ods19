package mapping

import (
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func mapODPropertyToProperty(i models.ODObjectPropertyEx) protocol.Property {
	o := protocol.Property{}
	o.ID = i.ID
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

func mapODPropertiesToProperties(i []models.ODObjectPropertyEx) []protocol.Property {
	o := make([]protocol.Property, len(i))
	for p, q := range i {
		o[p] = mapODPropertyToProperty(q)
	}
	return o
}

func mapPropertyToODProperty(i protocol.Property) models.ODObjectPropertyEx {
	o := models.ODObjectPropertyEx{}
	o.ID = i.ID
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
	return o
}

func mapPropertiesToODProperties(i []protocol.Property) []models.ODObjectPropertyEx {
	o := make([]models.ODObjectPropertyEx, len(i))
	for p, q := range i {
		o[p] = mapPropertyToODProperty(q)
	}
	return o
}
