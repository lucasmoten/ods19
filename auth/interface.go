package auth

import (
	"errors"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
)

var (
	// ErrACMNotSpecified is an authorization error that is returned if an ACM is required but not specified
	ErrACMNotSpecified = errors.New("auth: acm not specified")
	// ErrACMNotValid is an authorization error that is returned if an ACM provided is not in valid format
	ErrACMNotValid = errors.New("auth: acm not valid")
	// ErrFailToCheckUserAccess is an authorization error returned if service error occurs while checking user access to ACM
	ErrFailToCheckUserAccess = errors.New("auth: unable to check user access to acm")
	// ErrFailToFlattenACM is an authorization error returned if service error occurs while flattening an ACM
	ErrFailToFlattenACM = errors.New("auth: unable to flatten acm")
	// ErrFailToInjectPermissions is an authorization error returned if not able to inject the permissions into the ACM
	ErrFailToInjectPermissions = errors.New("auth: unable to inject permissions into acm")
	// ErrFailToNormalizePermissions is an authorization error returned if not able to normalize permissions and ACM
	ErrFailToNormalizePermissions = errors.New("auth: unable to normalize permissions")
	// ErrFailToRebuildACMFromPermissions is an authorization error returned if not able to rebuild the ACM
	ErrFailToRebuildACMFromPermissions = errors.New("auth: unable to rebuild acm from permissions")
	// ErrFailToRetrieveSnippets is an authorization error returned if service error occurs while retrieving snippets
	ErrFailToRetrieveSnippets = errors.New("auth: unable to retrieve snippets")
	// ErrUserNotAuthorized is an authorization error that is returned if a user identity does not have authorization for ACM
	ErrUserNotAuthorized = errors.New("auth: user not authorized")
	// ErrUserNotSpecified is an authorization error that is returned if a user identity is required but not specified
	ErrUserNotSpecified = errors.New("auth: user not specified")
)

// Authorization represents a common interface for which any auth implementation is expected to support
type Authorization interface {
	GetFlattenedACM(acm string) (flattenedACM string, err error)
	GetGroupsForUser(userIdentity string) (groups []string, err error)
	GetGroupsFromSnippets(snippets *acm.ODriveRawSnippetFields) (groups []string)
	GetSnippetsForUser(userIdentity string) (snippets *acm.ODriveRawSnippetFields, err error)
	InjectPermissionsIntoACM(permissions []models.ODObjectPermission, acm string) (modifiedACM string, err error)
	IsUserAuthorizedForACM(userIdentity string, acm string) (isAuthorized bool, err error)
	IsUserOwner(userIdentity string, resourceStrings []string, objectOwner string) (isOwner bool)
	NormalizePermissionsFromACM(objectOwner string, permissions []models.ODObjectPermission, acm string, isCreating bool) (modifiedPermissions []models.ODObjectPermission, modifiedACM string, err error)
	RebuildACMFromPermissions(permissions []models.ODObjectPermission, acm string) (modifiedACM string, err error)
}
