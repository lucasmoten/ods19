package models

import "decipher.com/object-drive-server/metadata/models/acm"

// ODUser is a structure defining the attributes of a user as referenced in
// Object Drive for caching purposes.  Unique to this element is the fact that
// its identifier is the DistinguishedName rather then ID.
type ODUser struct {
	ODID
	ODCreatable
	ODModifiable
	ODChangeTracking
	// DistinguishedName is the unique identifier of a user of the system. This
	// is generally mapped to the subject of an X509 certificate.
	DistinguishedName string `db:"distinguishedName"`
	// DisplayName is a 'nice' name to be used for rendering in user interfaces
	// when referring to a user instead of the lengthy distinguishedName
	DisplayName NullString `db:"displayName"`
	// Email is the address for sending correspondence to the user via electronic
	// mail.
	Email NullString `db:"email"`
	// Snippets holds a reference to the user snippets received from AAC
	Snippets *acm.ODriveRawSnippetFields
}

// ODUserResultset encapsulates the ODUser defined herein as an array with
// resultset metric information to expose page size, page number, total rows,
// and page count information when retrieving from the database
type ODUserResultset struct {
	Resultset
	Users []ODUser
}
