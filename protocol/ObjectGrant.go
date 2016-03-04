package protocol

// ObjectGrant is the grant of an object to a user - possibly the owner
// Granter and URL are implicit in the form of the POST
type ObjectGrant struct {
	// Grantee indicates the user for which this grant applies
	Grantee string `json:"grantee"`
	// Create indicates whether the grantee has permission to create child
	// objects beneath the target of this grant
	Create bool `json:"create"`
	// Read indicates whether the grantee has permission to read/view the
	// target of this grant
	Read bool `json:"read"`
	// Update indicates whether the grantee has permission to make changes
	// to the metadata or stream that is the target of this grant
	Update bool `json:"update"`
	// Delete indicates whether the grantee has permission to delete the
	// target of this grant
	Delete bool `json:"delete"`
	// Share indicates whether the grantee has permission to delegate the
	// same permissions established in this grant to others
	Share bool `json:"share"`
	// PropagateToChildren indicates whether the characteristics of this
	// grant will be recursively applied to existing children of the
	// target of this grant.  New children created always inherit the same
	// permissions of their parent.
	PropagateToChildren bool `json:"propagateToChildren"`
}
