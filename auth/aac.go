package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/deciphernow/object-drive-server/config"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/metadata/models/acm"
	"github.com/deciphernow/object-drive-server/services/aac"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/deciphernow/object-drive-server/utils"
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
	Version string
}

// NewAACAuth is a helper that builds an AACAuth from a provided logger and service connection
func NewAACAuth(logger zap.Logger, service aac.AacService) *AACAuth {
	a := &AACAuth{Logger: logger, Service: service, Version: "1.1"}
	// lm - Set AAC version based upon announcement point info, otherwise continue to assume 1.1
	AACAnnouncementPoint := os.Getenv(config.OD_ZK_AAC)
	if len(AACAnnouncementPoint) > 0 && strings.Contains(AACAnnouncementPoint, "/1.0/") {
		a.Version = "1.0"
	}
	return a
}

// GetAttributesForUser for AACAuth
func (aac *AACAuth) GetAttributesForUser(userIdentity string) (*acm.ODriveUserAttributes, error) {
	defer util.Time("GetAttributesForUser")()
	// No User (Anonymous)
	if userIdentity == "" {
		return nil, ErrUserNotSpecified
	}

	// Service state
	if aac.Service == nil {
		return nil, ErrServiceNotSet
	}

	// Do request
	getUserAttributesResponse, getUserAttributesError := aac.Service.GetUserAttributes(userIdentity, tokenType, snippetType)

	// Process response
	if getUserAttributesError != nil {
		aac.Logger.Error("Error calling AAC.GetUserAttributes", zap.String("err", getUserAttributesError.Error()))
		return nil, ErrFailToRetrieveAttributes
	}
	if getUserAttributesResponse == nil {
		aac.Logger.Error("Error calling AAC.GetUserAttributes", zap.String("getUserAttributesResponse", "nil"))
		return nil, ErrServiceNoResponse
	}
	for _, msg := range getUserAttributesResponse.Messages {
		aac.Logger.Info("AAC.GetUserAttributes response", zap.String("message", msg))
	}
	msgsString := strings.Join(getUserAttributesResponse.Messages, "/")
	if !getUserAttributesResponse.Success {
		aac.Logger.Error("AAC.GetUserAttributes failed", zap.Bool("success", getUserAttributesResponse.Success))
		return nil, fmt.Errorf("%s %s", ErrServiceNotSuccessful.Error(), msgsString)
	}

	// Convert to ODrive User Attributes
	convertedAttributes, convertedAttributesError := acm.NewODriveAttributesFromAttributeResponse(getUserAttributesResponse.UserAttributes)
	if convertedAttributesError != nil {
		aac.Logger.Error("Convert attributes to object failed", zap.String("err", convertedAttributesError.Error()))
		return nil, convertedAttributesError
	}

	return &convertedAttributes, nil
}

// GetFlattenedACM for AACAuth
func (aac *AACAuth) GetFlattenedACM(acm string) (string, []string, error) {
	defer util.Time("GetFlattenACM")()
	// Checks that dont depend on service availability
	// No ACM
	if acm == "" {
		return acm, nil, ErrACMNotSpecified
	}
	// Service state
	if aac.Service == nil {
		return acm, nil, ErrServiceNotSet
	}

	// Do request
	acmResponse, acmResponseError := aac.Service.PopulateAndValidateAcm(acm)

	// Process response
	if acmResponseError != nil {
		aac.Logger.Error("Error calling AAC.PopulateAndValidateAcm", zap.String("err", acmResponseError.Error()))
		return acm, nil, ErrFailToFlattenACM
	}
	if acmResponse == nil {
		aac.Logger.Error("Error calling AAC.PopulateAndValidateAcm", zap.String("acmResponse", "nil"))
		return acm, nil, ErrServiceNoResponse
	}
	for _, msg := range acmResponse.Messages {
		aac.Logger.Info("Message in AAC.PopulateAndValidateAcm", zap.String("message", msg))
	}

	if !acmResponse.Success {
		aac.Logger.Error("AAC.PopulateAndValidateAcm failed", zap.Bool("success", acmResponse.Success), zap.String("acm", acm))
		return acm, acmResponse.Messages, ErrACMResponseFailed
	}
	if !acmResponse.AcmValid {
		aac.Logger.Error("AAC.PopulateAndValidateAcm failed", zap.Bool("valid", acmResponse.AcmValid))
		return acm, acmResponse.Messages, ErrACMNotValid
	}
	if acmResponse.AcmInfo == nil {
		aac.Logger.Error("AAC.PopulateAndValidateAcm failed", zap.String("acmInfo", "nil"))
		return acm, acmResponse.Messages, ErrServiceNotSuccessful
	}

	// If passed all conditions, acm is flattened
	aac.Logger.Debug("AAC.PopulateAndValidateACM success", zap.String("before-acm", acm), zap.String("after-acm", acmResponse.AcmInfo.Acm))
	return acmResponse.AcmInfo.Acm, acmResponse.Messages, nil
}

// GetGroupsForUser for AACAuth
func (aac *AACAuth) GetGroupsForUser(userIdentity string) ([]string, error) {
	defer util.Time("GetGroupsForUser")()
	snippets, err := aac.GetSnippetsForUser(userIdentity)
	if err != nil {
		return nil, err
	}
	return aacGetGroupsFromSnippets(aac.Logger, snippets), nil
}

// GetGroupsFromSnippets for AACAuth
func (aac *AACAuth) GetGroupsFromSnippets(snippets *acm.ODriveRawSnippetFields) []string {
	defer util.Time("GetGroupsFromSnippets")()
	return aacGetGroupsFromSnippets(aac.Logger, snippets)
}

// GetSnippetsForUser for AACAuth
func (aac *AACAuth) GetSnippetsForUser(userIdentity string) (*acm.ODriveRawSnippetFields, error) {
	defer util.Time("GetSnippetsForUser")()
	// No User (Anonymous)
	if userIdentity == "" {
		return nil, ErrUserNotSpecified
	}

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
		aac.Logger.Info("AAC.GetSnippets response", zap.String("message", msg))
	}
	msgsString := strings.Join(getSnippetsResponse.Messages, "/")
	if !getSnippetsResponse.Success {
		aac.Logger.Error("AAC.GetSnippets failed", zap.Bool("success", getSnippetsResponse.Success))
		return nil, fmt.Errorf("%s %s", ErrServiceNotSuccessful.Error(), msgsString)
	}
	if aac.Version == "1.1" {
		if !getSnippetsResponse.Found {
			aac.Logger.Error("AAC.GetSnippets failed", zap.Bool("found", getSnippetsResponse.Found))
			return nil, fmt.Errorf("%s %s", ErrServiceNotSuccessful.Error(), msgsString)
		}
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
	defer util.Time("InjectPermissionsIntoACM")()
	return aacInjectPermissionsIntoACM(aac.Logger, permissions, acm)
}

// IsUserAuthorizedForACM for AACAuth
func (aac *AACAuth) IsUserAuthorizedForACM(userIdentity string, acm string) (bool, error) {
	defer util.Time("IsUserAuthorizedForACM")()
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
	flattenedACM, _, err := aac.GetFlattenedACM(acm)
	if err != nil {
		return false, err
	}

	// Call AAC Service.
	resp, err := aac.Service.CheckAccess(userIdentity, tokenType, flattenedACM)
	if err != nil {
		aac.Logger.Error("error calling AAC.CheckAccess", zap.String("err", err.Error()))
		return false, ErrFailToCheckUserAccess
	}
	if resp == nil {
		aac.Logger.Error("error calling AAC.CheckAccess", zap.String("response", "nil"))
		return false, ErrServiceNoResponse
	}
	for _, msg := range resp.Messages {
		aac.Logger.Info("Message in AAC.CheckAccess Response", zap.String("message", msg))
	}
	msgsString := strings.Join(resp.Messages, "/")
	if !resp.Success {
		aac.Logger.Error("AAC.CheckAccess failed", zap.Bool("success", resp.Success))
		return false, fmt.Errorf("%s %s", ErrServiceNotSuccessful.Error(), msgsString)
	}
	if !resp.HasAccess {
		aac.Logger.Error("AAC.CheckAccess failed", zap.Bool("hasAccess", resp.HasAccess))
		return false, fmt.Errorf("%s %s", ErrUserNotAuthorized.Error(), msgsString)
	}

	// If passed all conditions, user is authorized
	return true, nil
}

// IsUserOwner for AACAuth
func (aac *AACAuth) IsUserOwner(userIdentity string, resourceStrings []string, objectOwner string) bool {
	defer util.Time("IsUserOwner")()
	return aacIsUserOwner(aac.Logger, userIdentity, resourceStrings, objectOwner)
}

// NormalizePermissionsFromACM for AACAuth
func (aac *AACAuth) NormalizePermissionsFromACM(objectOwner string, permissions []models.ODObjectPermission, acm string, isCreating bool) ([]models.ODObjectPermission, string, error) {
	defer util.Time("NormalizePermissionsFromACM")()
	modifiedPermissions, modifiedACM, err := aacNormalizePermissionsFromACM(aac.Logger, objectOwner, permissions, acm, isCreating)
	// Service call for flattening populates f_* values
	modifiedACM, _, err = aac.GetFlattenedACM(modifiedACM)
	if err != nil {
		return modifiedPermissions, modifiedACM, err // fmt.Errorf("%v %s", err, strings.Join(msgs, "/"))
	}
	// Since AAC returns in its own order, we need to re-sort by keys for consistency
	acmMap, err := utils.UnmarshalStringToMap(modifiedACM)
	if err != nil {
		return modifiedPermissions, modifiedACM, fmt.Errorf("%s unmarshal error %s", ErrFailToRebuildACMFromPermissions, err.Error())
	}
	modifiedACM, err = utils.MarshalInterfaceToString(acmMap)
	if err != nil {
		return modifiedPermissions, modifiedACM, fmt.Errorf("%s marshal error %s", ErrFailToRebuildACMFromPermissions, err.Error())
	}
	// Done
	return modifiedPermissions, modifiedACM, err
}

// RebuildACMFromPermissions for AACAuth
func (aac *AACAuth) RebuildACMFromPermissions(permissions []models.ODObjectPermission, acm string) (string, error) {
	defer util.Time("RebuildACMFromPermissions")()
	return aacRebuildACMFromPermissions(aac.Logger, permissions, acm)
}

// aacCompileCheck ensures that AACAuth implements Authorization
func aacCompileCheck() Authorization {
	return &AACAuth{}
}
