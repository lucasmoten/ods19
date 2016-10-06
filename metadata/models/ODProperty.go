package models

import "time"

// ODProperty is a nestable structure defining the base attributes for a
// property that may be associated to an Object or an Object Type in Object
// Drive
type ODProperty struct {
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
	// ChangeCount indicates the number of times the item has been modified. For
	// newly created items, this value will reflect 0
	ChangeCount int `db:"changeCount"`
	// ChangeToken is generated value which is assigned at the database as a md5
	// hash of the concatencation of the id, changeCount, and most recent
	// modifiedDate as a string delimited by colons. For API calls performing
	// updates, the changeToken must be passed which will be compared against the
	// current value on the record. If properly implemented by callers, this will
	// prevent accidental overwrites.
	ChangeToken string `db:"changeToken"`
	// Name is the name, key, field, or label given to a property
	Name string `db:"name"`
	// Value is the assigned value for a property.
	Value NullString `db:"propertyValue"`
	// ClassificationPM is the portion mark classification for the value of this
	// property
	ClassificationPM NullString `db:"classificationPM"`
}
