package models

// ODObjectPropertyEx is a structure defining the attributes for a property
type ODObjectPropertyEx struct {
	ODCommonMeta
	ODChangeTracking
	// Name is the name, key, field, or label given to a property
	Name string `db:"name"`
	// Value is the assigned value for a property.
	Value NullString `db:"propertyValue"`
	// ClassificationPM is the portion mark classification for the value of this
	// property
	ClassificationPM NullString `db:"classificationPM"`
}
