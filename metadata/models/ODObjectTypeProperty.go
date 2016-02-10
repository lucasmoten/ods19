package models

// ODObjectTypeProperty is a structure defining the associative attributes
// linking an Object Type entity to a Property entity within Object Drive.
type ODObjectTypeProperty struct {
	ODCommonMeta
	// TypeID refers to the Object Type for which this linkage between type and
	// property is associated.
	TypeID []byte `db:"typeId"`
	// PropertyID refers to the Property for which this linkage between type
	// and property is associated.
	PropertyID []byte `db:"propertyId"`
	// Property references the actual underlying property object.
	Property ODProperty
}
