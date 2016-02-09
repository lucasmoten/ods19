package models

// ODProperty is a nestable structure defining the base attributes for a
// property that may be associated to an Object or an Object Type in Object
// Drive
type ODProperty struct {
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
