package models

/*
ODChangeTracking is a nestable structure defining the attributes tracked for
Object Drive elements that record the number of changes and use tokenization
to facilitate avoidance of blind overwrites
*/
type ODChangeTracking struct {
	ChangeCount int    `db:"changeCount"`
	ChangeToken string `db:"changeToken"`
}
