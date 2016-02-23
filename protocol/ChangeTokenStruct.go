package protocol

// ChangeTokenStruct is a nestable structure defining the ChangeToken attribute
// for items in Object Drive
type ChangeTokenStruct struct {
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken,omitempty"`
}
