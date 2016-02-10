package models

// ODID is a nestable structure defining an ID for Object Drive elements
type ODID struct {
	// ID is the unique identifier for an item in Object Drive.  It is structured
	// here as a byte array, intended to be used for storing a GUID/UUID.
	ID []byte `db:"id"`
}
