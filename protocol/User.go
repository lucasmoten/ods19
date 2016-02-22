package protocol

// User is the serialized version of user information
type User struct {
	// DistinguishedName is the unique identifier of a user of the system. This
	// is generally mapped to the subject of an X509 certificate.
	DistinguishedName string `json:"distinguishedName"`
	// DisplayName is a 'nice' name to be used for rendering in user interfaces
	// when referring to a user instead of the lengthy distinguishedName
	DisplayName string `json:"displayName"`
	// Email is the address for sending correspondence to the user via electronic
	// mail.
	Email string `json:"email"`
}
