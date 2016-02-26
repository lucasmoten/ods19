package models

// ODObject is a nestable structure defining the base attributes for an Object
// in Object Drive
type ODObject struct {
	ODCommonMeta
	ODChangeTracking
	// IsAncestorDeleted is flagged as true if a parent in the tree has their
	// ODDeletable.IsDeleted flag marked as true
	IsAncestorDeleted bool `db:"isAncestorDeleted" json:"-"`
	// IsExpunged denotes whether this object is considered permanently deleted
	// and will be excluded from all API calls and reports
	IsExpunged bool `db:"isExpunged" json:"-"`
	// ExpungedDate reflects the datetime for which the object was marked as
	// expunged if IsExpunged is set to true
	ExpungedDate NullTime `db:"expungedDate" json:"-"`
	//ExpungedBy contains the	distinguishedName of the user that marked the object
	// as expunged if IsExpunged is set to true
	ExpungedBy NullString `db:"expungedBy" json:"-"`
	// OwnedBy indicates the individual user or group that currently owns the
	// object and has implict full permissions on the object
	OwnedBy NullString `db:"ownedBy"`
	// TypeID references the ODObjectType by its ID indicating the type of this
	// object
	TypeID []byte `db:"typeId"`
	// Name is the given name for the object. (e.g., filename)
	Name string `db:"name"`
	// Description is an abstract of the object or its contents
	Description NullString `db:"description"`
	// ParentID references another Object by its ID indicating which object, if
	// any, contains, or is an ancestor of this object. (e.g., folder). An object
	// without a parent is considered to be contained within the 'root' or at the
	// 'top level'.
	ParentID []byte `db:"parentId"`
	// ContentConnector contains connection information for the storage of the
	// content of this object (e.g., S3 connection settings for bucket)
	ContentConnector NullString `db:"contentConnector" json:"-"`
	// RawACM is the raw ACM string that got supplied to create this object
	RawAcm NullString `db:"rawAcm"`
	// ContentType indicates the mime-type, and potentially the character set
	// encoding for the object contents
	ContentType NullString `db:"contentType"`
	// ContentSize denotes the length of the content stream for this object, in
	// bytes
	ContentSize NullInt64 `db:"contentSize"`
	// ContentHash represents a hash (MD5? SHA1? SHA256?) of the contents of the
	// object stream and can be used for deduplication with other objects stored
	// in a backend repository.
	ContentHash []byte `db:"contentHash"`
	// EncryptIV contains the initialization vector information for encrypting the
	// content stream for this object at result
	EncryptIV []byte `db:"encryptIV" json:"-"`
	// TypeName reflects the name of the object type associated with TypeID
	TypeName NullString `db:"typeName"`
	// Properties is an array of Object Properties associated with this object
	// structured as key/value with portion marking.
	Properties []ODObjectPropertyEx `json:"properties"`
	// Permissions is an array of Object Permissions associated with this object
	Permissions []ODObjectPermission `json:"permissions"`
}

// ODObjectResultset encapsulates the ODObject defined herein as an array with
// resultset metric information to expose page size, page number, total rows,
// and page count information when retrieving from the database
type ODObjectResultset struct {
	Resultset
	Objects []ODObject
}
