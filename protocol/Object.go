package protocol

import "time"

// Object is a nestable structure defining the base attributes for an Object
// in Object Drive
type Object struct {
	// ID is the unique identifier for this object in Object Drive.
	ID []byte `db:"id" json:"id"`
	// CreatedDate is the timestamp of when an item was created.
	CreatedDate time.Time `db:"createdDate" json:"createdDate"`
	// CreatedBy is the user that created this item.
	CreatedBy string `db:"createdBy" json:"createdBy"`
	// ModifiedDate is the timestamp of when an item was modified or created.
	ModifiedDate time.Time `db:"modifiedDate" json:"modifiedDate"`
	// ModifiedBy is the user that last modified this item
	ModifiedBy string `db:"modifiedBy" json:"modifiedBy"`
	// ChangeCount indicates the number of times the item has been modified.
	ChangeCount int `db:"changeCount" json:"changeCount"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `db:"changeToken" json:"changeToken"`
	// OwnedBy indicates the individual user or group that currently owns the
	// object and has implict full permissions on the object
	OwnedBy string `db:"ownedBy" json:"ownedBy"`
	// TypeID references the ODObjectType by its ID indicating the type of this
	// object
	TypeID []byte `db:"typeId" json:"typeId"`
	// TypeName reflects the name of the object type associated with TypeID
	TypeName string `db:"typeName" json:"typeName"`
	// Name is the given name for the object. (e.g., filename)
	Name string `db:"name" json:"name"`
	// Description is an abstract of the object or its contents
	Description string `db:"description" json:"description"`
	// ParentID references another Object by its ID indicating which object, if
	// any, contains, or is an ancestor of this object. (e.g., folder). An object
	// without a parent is considered to be contained within the 'root' or at the
	// 'top level'.
	ParentID []byte `db:"parentId" json:"parentId"`
	// RawACM is the raw ACM string that got supplied to create this object
	RawAcm string `db:"rawAcm" json:"acm"`
	// ContentType indicates the mime-type, and potentially the character set
	// encoding for the object contents
	ContentType string `db:"contentType" json:"contentType"`
	// ContentSize denotes the length of the content stream for this object, in
	// bytes
	ContentSize int64 `db:"contentSize" json:"contentSize"`
	// Properties is an array of Object Properties associated with this object
	// structured as key/value with portion marking.
	Properties []Property `json:"properties"`
	// Permissions is an array of Object Permissions associated with this object
	Permissions []Permission `json:"permissions"`
}
