package protocol

// MoveObjectRequest is a subset of Object for use to disallow providing certain fields.
type MoveObjectRequest struct {
	// ID is the unique identifier for this object in Object Drive.
	ID string `json:"id"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken,omitempty"`
	// ParentID is the identifier of the new parent to be assigned, or null if
	// moving to root
	ParentID string `json:"parentId,omitempty"`
}
