package models

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
