package auth

import (
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/metadata/models/acm"
)

// FakeAuth is suitable for tests. Add fields to this struct to hold fake
// reponses for each of the methods that FakeAuth will implement. These fake
// response fields can be explicitly set, or setup functions can be defined.
type FakeAuth struct {
	Err                 error
	FlattenedACM        string
	Groups              []string
	IsAuthorized        bool
	IsOwner             bool
	ModifiedACM         string
	ModifiedPermissions []models.ODObjectPermission
	Snippets            *acm.ODriveRawSnippetFields
	UserAttributes      *acm.ODriveUserAttributes
}

// GetAttributesForUser for FakeAuth
func (fake *FakeAuth) GetAttributesForUser(userIdentity string) (attributes *acm.ODriveUserAttributes, err error) {
	return fake.UserAttributes, fake.Err
}

// GetFlattenedACM for FakeAuth
func (fake *FakeAuth) GetFlattenedACM(acm string) (flattenedACM string, msgs []string, err error) {
	return fake.FlattenedACM, nil, fake.Err
}

// GetGroupsForUser for FakeAuth
func (fake *FakeAuth) GetGroupsForUser(userIdentity string) (groups []string, err error) {
	return fake.Groups, fake.Err
}

// GetGroupsFromSnippets for FakeAuth
func (fake *FakeAuth) GetGroupsFromSnippets(snippets *acm.ODriveRawSnippetFields) (groups []string) {
	return fake.Groups
}

// GetSnippetsForUser for FakeAuth
func (fake *FakeAuth) GetSnippetsForUser(userIdentity string) (snippets *acm.ODriveRawSnippetFields, err error) {
	return fake.Snippets, fake.Err
}

// InjectPermissionsIntoACM for FakeAuth
func (fake *FakeAuth) InjectPermissionsIntoACM(permissions []models.ODObjectPermission, acm string) (modifiedACM string, err error) {
	return fake.ModifiedACM, fake.Err
}

// IsUserAuthorizedForACM for FakeAuth
func (fake *FakeAuth) IsUserAuthorizedForACM(userIdentity string, acm string) (isAuthorized bool, err error) {
	return fake.IsAuthorized, fake.Err
}

// IsUserOwner for FakeAuth
func (fake *FakeAuth) IsUserOwner(userIdentity string, resourceStrings []string, objectOwner string) (isOwner bool) {
	return fake.IsOwner
}

// NormalizePermissionsFromACM for FakeAuth
func (fake *FakeAuth) NormalizePermissionsFromACM(objectOwner string, permissions []models.ODObjectPermission, acm string, isCreating bool) (modifiedPermissions []models.ODObjectPermission, modifiedACM string, err error) {
	return fake.ModifiedPermissions, fake.ModifiedACM, fake.Err
}

// RebuildACMFromPermissions for FakeAuth
func (fake *FakeAuth) RebuildACMFromPermissions(permissions []models.ODObjectPermission, acm string) (modifiedACM string, err error) {
	return fake.ModifiedACM, fake.Err
}

// fakeCompileCheck ensures that FakeAuth implements Authorization.
func fakeCompileCheck() Authorization {
	return &FakeAuth{}
}
