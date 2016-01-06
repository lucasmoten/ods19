package models

/*
ODID is a nestable structure defining an ID for Object Drive elements
*/
type ODID struct {
	ID []byte `db:"id"`
}
