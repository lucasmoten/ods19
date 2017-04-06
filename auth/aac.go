package auth

import (
	"errors"
	"fmt"
	"strings"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
	"decipher.com/object-drive-server/services/aac"
	"github.com/uber-go/zap"
)

var (
	// ErrServiceNotSet is an authorization error that is returned if the AAC service is not setup
	ErrServiceNotSet = errors.New("auth: service is not set on auth module")
	// ErrServiceNoResponse is an authorization error that is returned if the AAC service does not respond properly
	ErrServiceNoResponse = errors.New("auth: service returned a nil response")
	// ErrServiceNotSuccessful is an authorization error that is returned if the AAC service response indicates Success = false
	ErrServiceNotSuccessful = errors.New("auth: service returned an unsuccessful response")
)

const (
	snippetType     = "odrive-raw"
	tokenType       = "pki_dias"
	acmShareKey     = "share"
	snippetShareKey = "f_share"
)

// AACAuth is an Authorization implementation backed by the AAC Service.
type AACAuth struct {
	Logger  zap.Logger
	Service aac.AacService
}

// NewAACAuth is a helper that builds an AACAuth from a provided logger and service connection
func NewAACAuth(logger zap.Logger, service aac.AacService) *AACAuth {
	a := &AACAuth{Logger: logger, Service: service}
	return a
}

// GetFlattenedACM for AACAuth
func (aac *AACAuth) GetFlattenedACM(acm string) (string, error) {
	// Checks that dont depend on service availability
	// No ACM
	if acm == "" {
		return acm, ErrACMNotSpecified
	}
	// Service state
	if aac.Service == nil {
		return acm, ErrServiceNotSet
	}

	// Do request
	acmResponse, acmResponseError := aac.Service.PopulateAndValidateAcm(acm)

	// Process response
	if acmResponseError != nil {
		aac.Logger.Error("Error calling AAC.PopulateAndValidateAcm", zap.String("err", acmResponseError.Error()))
		return acm, ErrFailToFlattenACM
	}
	if acmResponse == nil {
		aac.Logger.Error("Error calling AAC.PopulateAndValidateAcm", zap.String("acmResponse", "nil"))
		return acm, ErrServiceNoResponse
	}
	for _, msg := range acmResponse.Messages {
		aac.Logger.Info("Message in AAC.PopulateAndValidateAcm", zap.String("msg", msg))
	}
	msgsString := strings.Join(acmResponse.Messages, "/")
	if !acmResponse.Success {
		aac.Logger.Error("AAC.PopulateAndValidateAcm failed", zap.Bool("success", acmResponse.Success), zap.String("acm", acm))

		return acm, fmt.Errorf("%s %s", ErrServiceNotSuccessful.Error(), msgsString)
	}
	if !acmResponse.AcmValid {
		aac.Logger.Error("AAC.PopulateAndValidateAcm failed", zap.Bool("valid", acmResponse.AcmValid))
		return acm, fmt.Errorf("%s %s", ErrACMNotValid.Error(), msgsString)
	}
	if acmResponse.AcmInfo == nil {
		aac.Logger.Error("AAC.PopulateAndValidateAcm failed", zap.String("acmInfo", "nil"))
		return acm, ErrServiceNotSuccessful
	}

	// If passed all conditions, acm is flattened
	aac.Logger.Debug("AAC.PopulateAndValidateACM success", zap.String("before-acm", acm), zap.String("after-acm", acmResponse.AcmInfo.Acm))
	return acmResponse.AcmInfo.Acm, nil
}

// GetGroupsForUser for AACAuth
func (aac *AACAuth) GetGroupsForUser(userIdentity string) ([]string, error) {
	snippets, err := aac.GetSnippetsForUser(userIdentity)
	if err != nil {
		return nil, err
	}
	return aacGetGroupsFromSnippets(aac.Logger, snippets), nil
}

// GetGroupsFromSnippets for AACAuth
func (aac *AACAuth) GetGroupsFromSnippets(snippets *acm.ODriveRawSnippetFields) []string {
	return aacGetGroupsFromSnippets(aac.Logger, snippets)
}

// GetSnippetsForUser for AACAuth
func (aac *AACAuth) GetSnippetsForUser(userIdentity string) (*acm.ODriveRawSnippetFields, error) {
	// No User (Anonymous)
	if userIdentity == "" {
		return nil, ErrUserNotSpecified
	}

	// TODO: Support injecting user profiles for server identities? Only if AAC wont handle it

	// Service state
	if aac.Service == nil {
		return nil, ErrServiceNotSet
	}

	// Do request
	getSnippetsResponse, getSnippetsError := aac.Service.GetSnippets(userIdentity, tokenType, snippetType)

	// Process response
	if getSnippetsError != nil {
		aac.Logger.Error("Error calling AAC.GetSnippets", zap.String("err", getSnippetsError.Error()))
		return nil, ErrFailToRetrieveSnippets
	}
	if getSnippetsResponse == nil {
		aac.Logger.Error("Error calling AAC.GetSnippets", zap.String("getSnippetsResponse", "nil"))
		return nil, ErrServiceNoResponse
	}
	for _, msg := range getSnippetsResponse.Messages {
		aac.Logger.Info("AAC.GetSnippets response", zap.String("msg", msg))
	}
	msgsString := strings.Join(getSnippetsResponse.Messages, "/")
	if !getSnippetsResponse.Success {
		aac.Logger.Error("AAC.GetSnippets failed", zap.Bool("success", getSnippetsResponse.Success))
		return nil, fmt.Errorf("%s %s", ErrServiceNotSuccessful.Error(), msgsString)
	}

	// Convert to Snippet Fields
	convertedSnippets, convertedSnippetsError := acm.NewODriveRawSnippetFieldsFromSnippetResponse(getSnippetsResponse.Snippets)
	if convertedSnippetsError != nil {
		aac.Logger.Error("Convert snippets to fields failed", zap.String("err", convertedSnippetsError.Error()))
		return nil, convertedSnippetsError
	}

	return &convertedSnippets, nil
}

// InjectPermissionsIntoACM for AACAuth
func (aac *AACAuth) InjectPermissionsIntoACM(permissions []models.ODObjectPermission, acm string) (string, error) {
	return aacInjectPermissionsIntoACM(aac.Logger, permissions, acm)
}

// IsUserAuthorizedForACM for AACAuth
func (aac *AACAuth) IsUserAuthorizedForACM(userIdentity string, acm string) (bool, error) {
	// Checks that dont depend on service availability
	// No ACM
	if acm == "" {
		return false, ErrACMNotSpecified
	}
	// No User (Anonymous) but ACM is present
	if userIdentity == "" {
		return false, ErrUserNotSpecified
	}
	// Service state
	if aac.Service == nil {
		return false, ErrServiceNotSet
	}

	// Preflatten
	flattenedACM, flattenedACMErr := aac.GetFlattenedACM(acm)
	if flattenedACMErr != nil {
		return false, flattenedACMErr
	}

	// Do request
	checkAccessResponse, checkAccessError := aac.Service.CheckAccess(userIdentity, tokenType, flattenedACM)

	// Process response
	if checkAccessError != nil {
		aac.Logger.Error("Error calling AAC.CheckAccess", zap.String("err", checkAccessError.Error()))
		return false, ErrFailToCheckUserAccess
	}
	if checkAccessResponse == nil {
		aac.Logger.Error("Error calling AAC.CheckAccess", zap.String("checkAccessResponse", "nil"))
		return false, ErrServiceNoResponse
	}
	for _, msg := range checkAccessResponse.Messages {
		aac.Logger.Info("Message in AAC.CheckAccess Response", zap.String("msg", msg))
	}
	msgsString := strings.Join(checkAccessResponse.Messages, "/")
	if !checkAccessResponse.Success {
		aac.Logger.Error("AAC.CheckAccess failed", zap.Bool("success", checkAccessResponse.Success))
		return false, fmt.Errorf("%s %s", ErrServiceNotSuccessful.Error(), msgsString)
	}
	if !checkAccessResponse.HasAccess {
		aac.Logger.Error("AAC.CheckAccess failed", zap.Bool("hasAccess", checkAccessResponse.HasAccess))
		return false, fmt.Errorf("%s %s", ErrUserNotAuthorized.Error(), msgsString)
	}

	// If passed all conditions, user is authorized
	return true, nil
}

// IsUserOwner for AACAuth
func (aac *AACAuth) IsUserOwner(userIdentity string, resourceStrings []string, objectOwner string) bool {
	return aacIsUserOwner(aac.Logger, userIdentity, resourceStrings, objectOwner)
}

// NormalizePermissionsFromACM for AACAuth
func (aac *AACAuth) NormalizePermissionsFromACM(objectOwner string, permissions []models.ODObjectPermission, acm string, isCreating bool) ([]models.ODObjectPermission, string, error) {
	return aacNormalizePermissionsFromACM(aac.Logger, objectOwner, permissions, acm, isCreating)
}

// RebuildACMFromPermissions for AACAuth
func (aac *AACAuth) RebuildACMFromPermissions(permissions []models.ODObjectPermission, acm string) (string, error) {
	return aacRebuildACMFromPermissions(aac.Logger, permissions, acm)
}

// aacCompileCheck ensures that AACAuth implements Authorization
func aacCompileCheck() Authorization {
	return &AACAuth{}
}
