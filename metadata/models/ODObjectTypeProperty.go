package models

/*
ODObjectTypeProperty is a structure defining the associative attributes linking
an Object Type entity to a Property entity within Object Drive.
*/
type ODObjectTypeProperty struct {
	ODCommonMeta
	TypeID     []byte `db:"typeId"`
	PropertyID []byte `db:"propertyId"`
	Property   ODProperty
}
