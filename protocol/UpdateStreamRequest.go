package protocol

// UpdateStreamRequest is for updating the object stream
type UpdateStreamRequest struct {
	ChangeToken string `json:"changeToken,omitempty"`
	// RawACM is the raw ACM string that got supplied to create this object
	RawAcm string `json:"acm"`
}
