package protocol

// Existing objects need a change token
type ObjectVersioned struct {
	ObjectID    string `json:"objectId"`
	ChangeToken string `json:"changeToken"`
}

// ObjectVersionedIds is a simple list of object identifiers
type ObjectVersionedIds struct {
	Objects []ObjectVersioned `json:"objects"`
}
