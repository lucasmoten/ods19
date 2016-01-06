package models

/*
ODObjectPropertyEx is a structure defining the attributes for a property
*/
type ODObjectPropertyEx struct {
	ODCommonMeta
	ODChangeTracking
	Name  string     `db:"name"`
	Value NullString `db:"propertyValue"`
	/*
		ClassificationPM is the portion mark classification for the value of this
		property
	*/
	ClassificationPM NullString `db:"classificationPM"`
}
