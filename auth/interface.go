package auth

import (
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
)

// Error is our error type.
type Error string

func (e Error) Error() string { return string(e) }

const (
	// ErrACMResponseFailed is a sentinal error for the case when Success == false in an *AcmResponse
	// This error type can be returned when err == nil from the service API call itself,
	// but the internal Success field shows false.
	ErrACMResponseFailed = Error("auth: acm response type marked as failed")
	// ErrACMNotSpecified is an authorization error that is returned if an ACM is required but not specified.
	ErrACMNotSpecified = Error("auth: acm not specified")
	// ErrACMNotValid is an authorization error that is returned if an ACM provided is not in valid format.
	ErrACMNotValid = Error("auth: acm not valid")
	// ErrFailToCheckUserAccess is an authorization error returned if service error occurs while checking user access to ACM.
	ErrFailToCheckUserAccess = Error("auth: unable to check user access to acm")
	// ErrFailToFlattenACM is an authorization error returned if service error occurs while flattening an ACM.
	ErrFailToFlattenACM = Error("auth: unable to flatten acm")
	// ErrFailToInjectPermissions is an authorization error returned if not able to inject the permissions into the ACM.
	ErrFailToInjectPermissions = Error("auth: unable to inject permissions into acm")
	// ErrFailToNormalizePermissions is an authorization error returned if not able to normalize permissions and ACM.
	ErrFailToNormalizePermissions = Error("auth: unable to normalize permissions")
	// ErrFailToRebuildACMFromPermissions is an authorization error returned if not able to rebuild the ACM.
	ErrFailToRebuildACMFromPermissions = Error("auth: unable to rebuild acm from permissions")
	// ErrFailToRetrieveAttributes is an authorization error returned if service error occurs while retrieving user attributes.
	ErrFailToRetrieveAttributes = Error("auth: unable to retrieve attributes")
	// ErrFailToRetrieveSnippets is an authorization error returned if service error occurs while retrieving snippets.
	ErrFailToRetrieveSnippets = Error("auth: unable to retrieve snippets")
	// ErrUserNotAuthorized is an authorization error that is returned if a user identity does not have authorization for ACM.
	ErrUserNotAuthorized = Error("auth: user not authorized")
	// ErrUserNotSpecified is an authorization error that is returned if a user identity is required but not specified.
	ErrUserNotSpecified = Error("auth: user not specified")
)

// Authorization represents a common interface for which any auth implementation is expected to support
type Authorization interface {
	GetAttributesForUser(userIdentity string) (attributes *acm.ODriveUserAttributes, err error)
	GetFlattenedACM(acm string) (flattenedACM string, msgs []string, err error)
	GetGroupsForUser(userIdentity string) (groups []string, err error)
	GetGroupsFromSnippets(snippets *acm.ODriveRawSnippetFields) (groups []string)
	GetSnippetsForUser(userIdentity string) (snippets *acm.ODriveRawSnippetFields, err error)
	InjectPermissionsIntoACM(permissions []models.ODObjectPermission, acm string) (modifiedACM string, err error)
	IsUserAuthorizedForACM(userIdentity string, acm string) (isAuthorized bool, err error)
	IsUserOwner(userIdentity string, resourceStrings []string, objectOwner string) (isOwner bool)
	NormalizePermissionsFromACM(objectOwner string, permissions []models.ODObjectPermission, acm string, isCreating bool) (modifiedPermissions []models.ODObjectPermission, modifiedACM string, err error)
	RebuildACMFromPermissions(permissions []models.ODObjectPermission, acm string) (modifiedACM string, err error)
}
