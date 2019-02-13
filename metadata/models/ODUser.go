package models

import (
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models/acm"
)

// ODUser is a structure defining the attributes of a user as referenced in
// Object Drive for caching purposes.  Unique to this element is the fact that
// its identifier is the DistinguishedName rather then ID.
type ODUser struct {
	// ID is the unique identifier for an item in Object Drive.
	ID []byte `db:"id"`
	// CreatedDate is the timestamp of when an item was created.
	CreatedDate time.Time `db:"createdDate"`
	// CreatedBy is the user, identified by distinguished name, that created this
	// item.
	CreatedBy string `db:"createdBy"`
	// ModifiedDate is the timestamp of when an item was modified. If an item
	// has only been created and not subsequently modified, its ModifiedDate
	// shall equate to the CreatedDate once stored in the repository.
	ModifiedDate time.Time `db:"modifiedDate"`
	// ModifiedBy is the user, identified by distinguished name, that last
	// modified this item
	ModifiedBy string `db:"modifiedBy"`
	// ChangeCount indicates the number of times the item has been modified. For
	// newly created items, this value will reflect 0
	ChangeCount int `db:"changeCount"`
	// ChangeToken is generated value which is assigned at the database as a md5
	// hash of the concatenation of the id, changeCount, and most recent
	// modifiedDate as a string delimited by colons. For API calls performing
	// updates, the changeToken must be passed which will be compared against the
	// current value on the record. If properly implemented by callers, this will
	// prevent accidental overwrites.
	ChangeToken string `db:"changeToken"`
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

// ODUserAOCache captures the attributes for metadata about the state of a
// User's Authorization Object Cache
type ODUserAOCache struct {
	// ID is the unique identifier for an item in Object Drive.
	ID int64 `db:"id"`
	// UserID is the unique identifier associating to the user
	UserID []byte `db:"userid"`
	// IsCaching indicates whether a separate routine, perhaps from this same
	// process, or a peer is currently rebuilding the user's cache
	IsCaching bool `db:"iscaching"`
	// CacheDate denotes how long ago this cache was rebuilt
	CacheDate NullTime `db:"cachedate"`
	// SHA256Hash is the SHA-256 bit hash of the user's snippets
	SHA256Hash string `db:"sha256hash"`
}
