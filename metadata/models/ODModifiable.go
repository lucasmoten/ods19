package models

import "time"

/*
ODModifiable is a nestable structure defining the attributes tracked for
Object Drive elements that may be modifiedBy
*/
type ODModifiable struct {
	ModifiedDate time.Time `db:"modifiedDate"`
	ModifiedBy   string    `db:"modifiedBy"`
}
