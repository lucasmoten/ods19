package models

import "time"

// ODCreatable is a nestable structure defining the attributes tracked for
// Object Drive elements that may be created
type ODCreatable struct {
	// CreatedDate is the timestamp of when an item was created.
	CreatedDate time.Time `db:"createdDate"`
	// CreatedBy is the user, identified by distinguished name, that created this
	// item.
	CreatedBy string `db:"createdBy"`
}
