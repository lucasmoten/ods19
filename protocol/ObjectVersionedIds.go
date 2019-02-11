package protocol

// ObjectVersioned provides for a structure for correlating the ID of an object
// to its current change token
type ObjectVersioned struct {
	// ObjectID is the unique identifier for this object in Object Drive.
	ObjectID string `json:"objectId"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken"`
}

// ObjectVersionedIds is a simple list of object identifiers
type ObjectVersionedIds struct {
	// Objects is an array of ObjectVersioned entries
	Objects []ObjectVersioned `json:"objects"`
}
