package models

// ODObjectProperty is a structure defining the associative attributes linking
// an Object entity to a Property entity within Object Drive.
type ODObjectProperty struct {
	ODCommonMeta
	// ObjectID refers to the Object for which this linkage between object and
	// property is associated.
	ObjectID []byte `db:"objectId"`
	// PropertyID refers to the Property for which this linkage between object
	// and property is associated.
	PropertyID []byte `db:"propertyId"`
	// Property references the actual underlying property object.
	Property ODProperty
}
