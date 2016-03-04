package protocol

// RemoveObjectShareRequest is the response information provided when an
// object share is deleted from Object Drive
type RemoveObjectShareRequest struct {
	// ObjectID is the unique identifier for this object in Object Drive.
	ObjectID string `json:"objecId"`
	// ShareID is the unique identifier of the share to be removed.
	ShareID string `json:"shareId"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeTokenStruct
}
