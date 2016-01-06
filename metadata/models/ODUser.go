package models

/*
ODUser is a structure defining the attributes of a user as referenced in
Object Drive for caching purposes.  Unique to this element is the fact that
its identifier is the DistinguishedName rather then ID.
*/
type ODUser struct {
	ODCreatable
	ODModifiable
	ODChangeTracking
	DistinguishedName string     `db:"distinguishedName"`
	DisplayName       NullString `db:"displayName"`
	Email             NullString `db:"email"`
}
