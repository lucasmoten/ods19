package protocol

import "time"

// DeletedObject is a structure defining the base attributes for an Object
// in Object Drive that is deleted. It is primarily returned from GetObject
// calls for an objects property that appears in the trash.
type DeletedObject struct {
	// ID is the unique identifier for this object in Object Drive.
	ID string `json:"id"`
	// CreatedDate is the timestamp of when an item was created.
	CreatedDate time.Time `json:"createdDate"`
	// CreatedBy is the user that created this item.
	CreatedBy string `json:"createdBy"`
	// ModifiedDate is the timestamp of when an item was modified or created.
	ModifiedDate time.Time `json:"modifiedDate"`
	// ModifiedBy is the user that last modified this item
	ModifiedBy string `json:"modifiedBy"`
	// DeletedDate is the timestamp of when an item was deleted
	DeletedDate time.Time `json:"deletedDate"`
	// DeletedBy is the user that last modified this item
	DeletedBy string `json:"deletedBy"`
	// ChangeCount indicates the number of times the item has been modified.
	ChangeCount int `json:"changeCount"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeTokenStruct
	// OwnedBy indicates the individual user or group that currently owns the
	// object and has implict full permissions on the object
	OwnedBy string `json:"ownedBy"`
	// TypeID references the ODObjectType by its ID indicating the type of this
	// object
	TypeID string `json:"typeId,omitempty"`
	// TypeName reflects the name of the object type associated with TypeID
	TypeName string `json:"typeName"`
	// Name is the given name for the object. (e.g., filename)
	Name string `json:"name"`
	// Description is an abstract of the object or its contents
	Description string `json:"description"`
	// ParentID references another Object by its ID indicating which object, if
	// any, contains, or is an ancestor of this object. (e.g., folder). An object
	// without a parent is considered to be contained within the 'root' or at the
	// 'top level'.
	ParentID string `json:"parentId,omitempty"`
	// RawACM is the raw ACM string that got supplied to create this object
	RawAcm interface{} `json:"acm"`
	// ContentType indicates the mime-type, and potentially the character set
	// encoding for the object contents
	ContentType string `json:"contentType"`
	// ContentSize denotes the length of the content stream for this object, in
	// bytes
	ContentSize int64 `json:"contentSize"`
	// A sha256 hash of the plaintext as hex encoded string
	ContentHash string `json:"contentHash"`
	// IsPDFAvailable indicates if a PDF rendition is available for this object
	IsPDFAvailable bool `json:"isPDFAvailable"`
	// ContainsUSPersonsData indicates if this object contains US Persons data (Yes,No,Unknown)
	ContainsUSPersonsData string `json:"containsUSPersonsData"`
	// ExemptFromFOIA indicates if this object is exempt from Freedom of Information Act requests (Yes,No,Unknown)
	ExemptFromFOIA string `json:"exemptFromFOIA"`
	// Properties is an array of Object Properties associated with this object
	// structured as key/value with portion marking.
	Properties []Property `json:"properties,omitempty"`
	// CallerPermission is the composite permission the caller has for this object
	CallerPermission CallerPermission `json:"callerPermission,omitempty"`
	//Permission Permission `json:"permission,omitempty"`
	// Permissions is an array of Object Permissions associated with this object
	// This might be null.  It could have a large list of permission objects
	// relevant to this file (ie: shared with an organization)
	Permissions []Permission `json:"permissions,omitempty"`
}

// WithCallerPermission rolls up permissions for a caller, sets them on a copy of
// the DeletedObject, and returns that copy.
func (obj DeletedObject) WithCallerPermission(caller Caller) DeletedObject {

	var cp CallerPermission
	cp = cp.WithRolledUp(caller, obj.Permissions...)
	cp.AllowCreate = false
	cp.AllowUpdate = false
	cp.AllowShare = false
	obj.CallerPermission = cp
	return obj
}
