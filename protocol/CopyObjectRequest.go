package protocol

// CopyObjectRequest is a subset of Object for use to disallow providing certain fields.
type CopyObjectRequest struct {
	// ID is the unique identifier for this object in Object Drive.
	ID string `json:"id"`
}
