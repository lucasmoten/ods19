package models

/*
ODObjectProperty is a structure defining the associative attributes linking an
Object entity to a Property entity within Object Drive.
*/
type ODObjectProperty struct {
	ODCommonMeta
	ObjectID   []byte `db:"objectId"`
	PropertyID []byte `db:"propertyId"`
	Property   ODProperty
}
