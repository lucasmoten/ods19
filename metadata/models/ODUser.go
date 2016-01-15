package models

/*
ODUser is a structure defining the attributes of a user as referenced in
Object Drive for caching purposes.  Unique to this element is the fact that
its identifier is the DistinguishedName rather then ID.
*/
type ODUser struct {
	ODID
	ODCreatable
	ODModifiable
	ODChangeTracking
	DistinguishedName string     `db:"distinguishedName"`
	DisplayName       NullString `db:"displayName"`
	Email             NullString `db:"email"`
}

/*
ODUserResultset encapsulates the ODUser defined herein as an array with
resultset metric information to expose page size, page number, total rows, and
page count information when retrieving from the database
*/
type ODUserResultset struct {
	Resultset
	Users []ODUser
}
