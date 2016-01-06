package models

/*
ODDeletable is a nestable structure defining the attributes tracked for
Object Drive elements that may be deletedBy
*/
type ODDeletable struct {
	IsDeleted   bool       `db:"isDeleted"`
	DeletedDate NullTime   `db:"deletedDate"`
	DeletedBy   NullString `db:"deletedBy"`
}
