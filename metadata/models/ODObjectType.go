package models

// ODObjectType is a nestable structure defining the base attributes for an
// Object Type in Object Drive
type ODObjectType struct {
	ODCommonMeta
	ODChangeTracking
	// OwnedBy indicates the individual user or group that currently owns the type
	// and has implict full permissions on it
	OwnedBy NullString `db:"ownedBy"`
	// Name is the given name for the object type. (e.g., Document, Image)
	Name string `db:"name"`
	// Description is an abstract of the type such as its purpose
	Description NullString `db:"description"`
	// ContentConnector contains default connection information for the storage of
	// the content of new objects created of this type
	ContentConnector NullString `db:"contentConnector"`
	// Properties is an array of Object Properties associated with this Object
	// Type structured as key/value with portion marking.  When new objects are
	// created of this type, the properties defined on the type act as default
	// initializer
	Properties []ODObjectPropertyEx
}
