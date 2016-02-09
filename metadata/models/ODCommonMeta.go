package models

// ODCommonMeta is a nestable structure defining the attributes most common for
// Object Drive elements
type ODCommonMeta struct {
	ODID
	ODCreatable
	ODModifiable
	ODDeletable
}
