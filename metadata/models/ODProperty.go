package models

/*
ODProperty is a nestable structure defining the base attributes for a property
that may be associated to an Object or an Object Type in Object Drive
*/
type ODProperty struct {
	ODCommonMeta
	ODChangeTracking
	Name             string     `db:"name"`
	Value            NullString `db:"propertyValue"`
	ClassificationPM NullString `db:"classificationPM"`
}
