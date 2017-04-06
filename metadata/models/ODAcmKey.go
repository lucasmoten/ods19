package models

import "time"

// ODAcmKey2 is a simple type holding the name of an ACM field
type ODAcmKey2 struct {
	// ID is the unique identifier for this acm field in the metadata store
	ID int64 `db:"id"`
	// Name is the given name for the acm value
	Name string `db:"name"`
}

// ODAcmValue2 is a simple type holding the value of an ACM field
type ODAcmValue2 struct {
	// ID is the unique identifier for this acm value in the metadata store
	ID int64 `db:"id"`
	// Name is the given name for the acm field
	Name string `db:"name"`
}

// ODAcm2 is a simple type holding the name (flattened/normalized acm) and corresponding hash of an ACM
type ODAcm2 struct {
	// ID is the unique identifier for this acm in the metadata store
	ID int64 `db:"id"`
	// SHA256Hash is the SHA-256 bit hash of the flattenedACM value
	SHA256Hash string `db:"sha256hash"`
	// FlattenedACM is the name given for the acm
	FlattenedACM string `db:"flattenedacm"`
}

// ODAcmPart2 is a struct holding joins between an acm definition, key, and value
type ODAcmPart2 struct {
	// ID is the unique identifier for this acmpart in the metadata store
	ID int64 `db:"id"`
	// ACMID is the unique identifier of the acm for which this part associates to
	ACMID int64 `db:"acmid"`
	// ACMKeyID is the unique identifier for the acm field of this part
	ACMKeyID int64 `db:"acmkeyid"`
	// ACMValueID is the unique identifier for a value of the acm field for this part
	ACMValueID int64 `db:"acmvalueid"`
}

// ODAcmKey is a simple type holding the name of an ACM field
type ODAcmKey struct {
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
	// IsDeleted indicates whether the item is currently marked as deleted and
	// subsequently filtered from certain API results
	IsDeleted bool `db:"isDeleted" json:"-"`
	// DeletedDate is the timestamp of when an item was deleted, or null if it
	// currently is not deleted.
	DeletedDate NullTime `db:"deletedDate" json:"-"`
	// DeletedBy is the user, identified by distinguished name, that marked the
	// item as deleted, or null if the item is currently not deleted.
	DeletedBy NullString `db:"deletedBy" json:"-"`
	// Name is the given name for the acm field
	Name string `db:"name"`
}

// ODAcmValue is a simple type holding the value of an ACM field
type ODAcmValue struct {
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
	// IsDeleted indicates whether the item is currently marked as deleted and
	// subsequently filtered from certain API results
	IsDeleted bool `db:"isDeleted" json:"-"`
	// DeletedDate is the timestamp of when an item was deleted, or null if it
	// currently is not deleted.
	DeletedDate NullTime `db:"deletedDate" json:"-"`
	// DeletedBy is the user, identified by distinguished name, that marked the
	// item as deleted, or null if the item is currently not deleted.
	DeletedBy NullString `db:"deletedBy" json:"-"`
	// Name is the given name for the acm field
	Name string `db:"name"`
}

// ODAcm is a simple type holding the name (flattened/normalized acm, or hash thereof) of an ACM
type ODAcm struct {
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
	// IsDeleted indicates whether the item is currently marked as deleted and
	// subsequently filtered from certain API results
	IsDeleted bool `db:"isDeleted" json:"-"`
	// DeletedDate is the timestamp of when an item was deleted, or null if it
	// currently is not deleted.
	DeletedDate NullTime `db:"deletedDate" json:"-"`
	// DeletedBy is the user, identified by distinguished name, that marked the
	// item as deleted, or null if the item is currently not deleted.
	DeletedBy NullString `db:"deletedBy" json:"-"`
	// Name is the given name for the acm
	Name string `db:"name"`
}

// ODAcmPart is a struct holding joins between an acm definition, key, and value
type ODAcmPart struct {
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
	// IsDeleted indicates whether the item is currently marked as deleted and
	// subsequently filtered from certain API results
	IsDeleted bool `db:"isDeleted" json:"-"`
	// DeletedDate is the timestamp of when an item was deleted, or null if it
	// currently is not deleted.
	DeletedDate NullTime `db:"deletedDate" json:"-"`
	// DeletedBy is the user, identified by distinguished name, that marked the
	// item as deleted, or null if the item is currently not deleted.
	DeletedBy NullString `db:"deletedBy" json:"-"`
	// AcmID is the unique identifier for an ACM definition in Object Drive.
	AcmID []byte `db:"acmId"`
	// AcmKeyID is the unique identifier for an acm key
	AcmKeyID []byte `db:"acmKeyId"`
	// AcmKeyName is the name of an acm key
	AcmKeyName string `db:"acmKeyName"`
	// AcmValueID is the unique identifier for an acm value
	AcmValueID []byte `db:"acmValueId"`
	// AcmValueName is the name of an acm value
	AcmValueName string `db:"acmValueName"`
}

// ODObjectAcm is a struct holding joins between an object, an acm key, and value
type ODObjectAcm struct {
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
	// IsDeleted indicates whether the item is currently marked as deleted and
	// subsequently filtered from certain API results
	IsDeleted bool `db:"isDeleted" json:"-"`
	// DeletedDate is the timestamp of when an item was deleted, or null if it
	// currently is not deleted.
	DeletedDate NullTime `db:"deletedDate" json:"-"`
	// DeletedBy is the user, identified by distinguished name, that marked the
	// item as deleted, or null if the item is currently not deleted.
	DeletedBy NullString `db:"deletedBy" json:"-"`
	// ObjectID is the unique identifier for an item in Object Drive.
	ObjectID []byte `db:"objectId"`
	// AcmKeyID is the unique identifier for an acm key
	AcmKeyID []byte `db:"acmKeyId"`
	// AcmKeyName is the name of an acm key
	AcmKeyName string `db:"acmKeyName"`
	// AcmValueID is the unique identifier for an acm value
	AcmValueID []byte `db:"acmValueId"`
	// AcmValueName is the name of an acm value
	AcmValueName string `db:"acmValueName"`
}
