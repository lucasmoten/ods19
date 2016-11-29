package protocol

// ChangeOwnerRequest is a subset of Object for use to disallow providing certain fields.
type ChangeOwnerRequest struct {
	// ID is the unique identifier for this object in Object Drive.
	ID string `json:"id"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken,omitempty"`
	// NewOwner indicates the individual user or group that will become the new
	// owner of the object
	NewOwner string `json:"newOwner"`
}
