package protocol

import "time"

// Property is a structure defining the attributes for a property
type Property struct {
	// ID is the unique identifier for this property in Object Drive.
	ID string `json:"id"`
	// CreatedDate is the timestamp of when a property was created.
	CreatedDate time.Time `json:"createdDate"`
	// CreatedBy is the user that created this property.
	CreatedBy string `json:"createdBy"`
	// ModifiedDate is the timestamp of when a property was modified or created.
	ModifiedDate time.Time `json:"modifiedDate"`
	// ModifiedBy is the user that last modified this property
	ModifiedBy string `json:"modifiedBy"`
	// ChangeCount indicates the number of times the property has been modified.
	ChangeCount int `json:"changeCount"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken"`
	// Name is the name, key, field, or label given to a property
	Name string `json:"name"`
	// Value is the assigned value for a property.
	Value string `json:"propertyValue"`
	// ClassificationPM is the portion mark classification for the value of this
	// property
	ClassificationPM string `json:"classificationPM"`
}
