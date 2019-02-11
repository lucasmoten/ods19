package models

import "time"

// ODRelationship is a structure defining the associative attributes linking
// two Objects to each other within Object Drive.
type ODRelationship struct {
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
	// hash of the concatenation of the id, changeCount, and most recent
	// modifiedDate as a string delimited by colons. For API calls performing
	// updates, the changeToken must be passed which will be compared against the
	// current value on the record. If properly implemented by callers, this will
	// prevent accidental overwrites.
	ChangeToken string `db:"changeToken"`
	// SourceID is the identifier of the source object for this relationship.
	// In a hierarchial context, the source refers to the 'parent' item
	SourceID []byte `db:"sourceId"`
	// TargetID is the identifier of the target object for this relationship.
	// In a hierarchial context, the target refers to the 'child' item
	TargetID []byte `db:"targetId"`
	// Description is an indicator of the purpose of the relationship.
	Description NullString `db:"description"`
	// ClassificationPM is the portion mark classification for the description of
	// this relationship
	ClassificationPM NullString `db:"classificationPM"`
}
