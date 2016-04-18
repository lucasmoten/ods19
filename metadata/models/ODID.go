package models

import "decipher.com/object-drive-server/util"

// ODID is a nestable structure defining an ID for Object Drive elements
type ODID struct {
	// ID is the unique identifier for an item in Object Drive.
	ID []byte `db:"id"`
}

// NewODID constructs a GUID and sets it on an ODID.
func NewODID() (ODID, error) {
	g, err := util.NewGUIDBytes()
	if err != nil {
		return ODID{}, err
	}
	return ODID{ID: g}, nil
}
