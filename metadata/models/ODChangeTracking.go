package models

// ODChangeTracking is a nestable structure defining the attributes tracked for
// Object Drive elements that record the number of changes and use tokenization
// to facilitate avoidance of blind overwrites
type ODChangeTracking struct {
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
}
