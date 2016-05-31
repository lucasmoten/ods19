package models

// ODAcmKey is a simple type holding the name of an ACM field
type ODAcmKey struct {
	ODCommonMeta
	// Name is the given name for the acm field
	Name string `db:"name"`
}

// ODAcmValue is a simple type holding the value of an ACM field
type ODAcmValue struct {
	ODCommonMeta
	// Name is the given name for the acm field
	Name string `db:"name"`
}

// ODObjectAcm is a struct holding joins between an object, an acm key and value
type ODObjectAcm struct {
	ODCommonMeta
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
