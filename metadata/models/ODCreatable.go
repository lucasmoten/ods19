package models

import "time"

/*
ODCreatable is a nestable structure defining the attributes tracked for
Object Drive elements that may be created
*/
type ODCreatable struct {
	CreatedDate time.Time `db:"createdDate"`
	CreatedBy   string    `db:"createdBy"`
}
