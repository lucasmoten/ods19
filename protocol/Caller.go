package protocol

import (
	"github.com/deciphernow/object-drive-server/metadata/models"
)

// Caller provides the distinguished names obtained from specific request
// headers and peer certificate if called directly. Note that this type
// mirrors the type in package server, and we should aim to unify them.
type Caller struct {
	// DistinguishedName is the unique identity of a user
	DistinguishedName string
	// UserDistinguishedName holds the value passed in header USER_DN
	UserDistinguishedName string
	// ExternalSystemDistinguishedName holds the value passed in header EXTERNAL_SYS_DN
	ExternalSystemDistinguishedName string
	// SSLClientSDistinguishedName holds the value passed in header SSL_CLIENT_S_DN
	SSLClientSDistinguishedName string
	// CommonName is the CN value part of the DistinguishedName
	CommonName string
	// TransactionType can be either NORMAL, IMPERSONATION, or UNKNOWN
	TransactionType string
	// Groups are extracted from the f_share fields for a Caller. Groups should be flattened
	// before comparing strings.
	Groups []string
}

// InGroup tests a group name against the Caller's embedded slice of groups to determine
// if the Caller is a member. Group names are flattened for comparison.
func (caller Caller) InGroup(group string) bool {

	for _, grp := range caller.Groups {
		if models.AACFlatten(group) == models.AACFlatten(grp) {
			return true
		}
	}
	return false
}
