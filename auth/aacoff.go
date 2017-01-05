package auth

import (
	"fmt"
	"strings"

	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/utils"
	"github.com/uber-go/zap"
)

// AACAuthOff is an Authorization implementation modeled on AAC in an offline / not-ready state.
type AACAuthOff struct {
	Logger zap.Logger
}

const (
	snippetUnclassifiedUSA = `{
    "f_macs": {"field": "f_macs", "treatment": "allow", "values":[]},
    "f_oc_org": {"field": "f_oc_org", "treatment": "allow", "values": []},
    "f_accms": {"field": "f_accms", "treatment": "allow", "values":[]},
    "f_sap": {"field": "f_sar_id", "treatment": "allow", "values":[]},
    "f_clearance": {"field": "f_clearance", "treatment": "allow", "values": ["u"]},
    "f_regions": {"field": "f_regions", "treatment": "allow", "values": []},
    "f_missions": {"field": "f_missions", "treatment": "allow", "values": []},
    "f_share": {"field": "f_share", "treatment": "allow", "values": ["%s"]},
    "f_aea": {"field": "f_atom_energy", "treatment": "allow", "values": []},
    "f_sci_ctrls": {"field": "f_sci_ctrls", "treatment": "allow", "values": []},
    "dissem_countries": {"field": "dissem_countries", "treatment": "allow", "values": ["USA"]}
    }`
)

// NewAACAuthOff is a helper that builds an AACAuthOff from a provided logger
func NewAACAuthOff(logger zap.Logger) *AACAuthOff {
	a := &AACAuthOff{Logger: logger}
	return a
}

// GetFlattenedACM for AACAuthOff
func (aac *AACAuthOff) GetFlattenedACM(acm string) (string, error) {
	// Checks that dont depend on service availability
	// No ACM
	if acm == "" {
		return acm, ErrACMNotSpecified
	}
	return acm, ErrServiceNotSet
}

// GetGroupsForUser for AACAuthOff
func (aac *AACAuthOff) GetGroupsForUser(userIdentity string) ([]string, error) {
	var err error
	var snippets *acm.ODriveRawSnippetFields
	snippets, err = aac.GetSnippetsForUser(userIdentity)
	if err != nil {
		return nil, err
	}
	return aacGetGroupsFromSnippets(aac.Logger, snippets), nil
}

// GetGroupsFromSnippets for AACAuthOff
func (aac *AACAuthOff) GetGroupsFromSnippets(snippets *acm.ODriveRawSnippetFields) []string {
	return aacGetGroupsFromSnippets(aac.Logger, snippets)
}

// GetSnippetsForUser for AACAuthOff
func (aac *AACAuthOff) GetSnippetsForUser(userIdentity string) (*acm.ODriveRawSnippetFields, error) {
	var snippets *acm.ODriveRawSnippetFields
	// No User (Anonymous)
	if userIdentity == "" {
		return nil, ErrUserNotSpecified
	}

	flattenedForward := aacFlatten(userIdentity)
	snippetString := fmt.Sprintf(snippetUnclassifiedUSA, flattenedForward)

	// Convert to Snippet Fields
	convertedSnippets, convertedSnippetsError := acm.NewODriveRawSnippetFieldsFromSnippetResponse(snippetString)
	if convertedSnippetsError != nil {
		aac.Logger.Error("Convert snippets to fields failed", zap.String("err", convertedSnippetsError.Error()))
		return nil, convertedSnippetsError
	}

	snippets = &convertedSnippets
	return snippets, nil
}

// InjectPermissionsIntoACM for AACAuth
func (aac *AACAuthOff) InjectPermissionsIntoACM(permissions []models.ODObjectPermission, acm string) (string, error) {
	return aacInjectPermissionsIntoACM(aac.Logger, permissions, acm)
}

// IsUserAuthorizedForACM for AACAuth
func (aac *AACAuthOff) IsUserAuthorizedForACM(userIdentity string, acm string) (bool, error) {
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
	return false, ErrServiceNotSet

}

// IsUserOwner for AACAuthOff
func (aac *AACAuthOff) IsUserOwner(userIdentity string, resourceStrings []string, objectOwner string) bool {
	return aacIsUserOwner(aac.Logger, userIdentity, resourceStrings, objectOwner)
}

// NormalizePermissionsFromACM for AACAuthOff
func (aac *AACAuthOff) NormalizePermissionsFromACM(objectOwner string, permissions []models.ODObjectPermission, acm string, isCreating bool) ([]models.ODObjectPermission, string, error) {
	return aacNormalizePermissionsFromACM(aac.Logger, objectOwner, permissions, acm, isCreating)
}

// RebuildACMFromPermissions for AACAuthOff
func (aac *AACAuthOff) RebuildACMFromPermissions(permissions []models.ODObjectPermission, acm string) (string, error) {
	return aacRebuildACMFromPermissions(aac.Logger, permissions, acm)
}

// aacOffCompileCheck ensures that AACAuthOff implements Authorization
func aacOffCompileCheck() Authorization {
	return &AACAuthOff{}
}

func aacFlatten(inVal string) string {
	emptyList := []string{" ", ",", "=", "'", ":", "(", ")", "$", "[", "]", "{", "}", "|", "\\"}
	underscoreList := []string{".", "-"}
	outVal := strings.ToLower(inVal)
	for _, s := range emptyList {
		outVal = strings.Replace(outVal, s, "", -1)
	}
	for _, s := range underscoreList {
		outVal = strings.Replace(outVal, s, "_", -1)
	}
	return outVal
}
func aacGetGroupsFromSnippets(logger zap.Logger, snippets *acm.ODriveRawSnippetFields) []string {
	var groups []string
	if snippets != nil {
		for _, field := range snippets.Snippets {
			if field.FieldName == snippetShareKey {
				groups = field.Values
				break
			}
		}
	}
	return groups
}
func aacInjectPermissionsIntoACM(logger zap.Logger, permissions []models.ODObjectPermission, acm string) (string, error) {
	var modifiedACM string
	var err error
	var emptyInterface interface{}

	// Convert to an addressable map
	acmMap, err := utils.UnmarshalStringToMap(acm)
	if err != nil {
		return acm, fmt.Errorf("%s %s", ErrFailToInjectPermissions, err.Error())
	}

	// Process permissions
	for idx, permission := range permissions {
		if permission.IsDeleted {
			continue
		}
		if !permission.AllowRead {
			continue
		}
		// If permission gives read to everyone, we need to reset share back to blank!
		if aacIsPermissionFor(permission, models.EveryoneGroup) {
			delete(acmMap, acmShareKey)
			break
		}
		acmShareInterface, ok := acmMap[acmShareKey]
		if !ok {
			acmShareInterface = emptyInterface
		}
		permissionInterface, err := utils.UnmarshalStringToInterface(permission.AcmShare)
		if err != nil {
			return acm, fmt.Errorf("%s permission %d unmarshal error from %s %s", ErrFailToInjectPermissions, idx, permission.AcmShare, err.Error())
		}
		acmMap[acmShareKey] = utils.CombineInterface(acmShareInterface, permissionInterface)
	}

	modifiedACM, err = utils.MarshalInterfaceToString(acmMap)
	if err != nil {
		return acm, fmt.Errorf("%s marshal error %s", ErrFailToInjectPermissions, err.Error())
	}
	modifiedACM, err = utils.NormalizeMarshalledInterface(modifiedACM)
	if err != nil {
		return acm, fmt.Errorf("%s normalize error %s", ErrFailToInjectPermissions, err.Error())
	}
	return modifiedACM, err
}
func aacIsPermissionFor(permission models.ODObjectPermission, grantee string) bool {
	return aacFlatten(permission.Grantee) == aacFlatten(grantee)
}
func aacIsUserOwner(logger zap.Logger, userIdentity string, resourceStrings []string, objectOwner string) bool {
	// No User (Anonymous)
	if userIdentity == "" {
		return false
	}

	// As the user in native format
	if strings.TrimSpace(userIdentity) == strings.TrimSpace(objectOwner) {
		return true
	}
	// As the user in resource string format
	if ("user/" + strings.TrimSpace(userIdentity)) == strings.TrimSpace(objectOwner) {
		return true
	}
	// As a group the user is a member of
	for _, resourceString := range resourceStrings {
		if strings.TrimSpace(objectOwner) == strings.TrimSpace(resourceString) {
			return true
		}
	}
	return false
}
func aacNormalizePermissionsFromACM(logger zap.Logger, objectOwner string, permissions []models.ODObjectPermission, acm string, isCreating bool) ([]models.ODObjectPermission, string, error) {
	var modifiedPermissions []models.ODObjectPermission
	var modifiedACM string
	var err error

	// Derive current read permissions from the ACM
	acmMap, err := utils.UnmarshalStringToMap(acm)
	if err != nil {
		return nil, "", fmt.Errorf("%s error unmarshalling acm %s", ErrFailToNormalizePermissions.Error(), err.Error())
	}
	shareInterface := acmMap[acmShareKey]
	var sharePermissions []models.ODObjectPermission
	// TODO: Consider moving MapObjectShareToODPermissions into this pacakge, and resolve protocl.ObjectShare in the process
	sharePermissions, err = mapping.MapObjectShareToODPermissions(&protocol.ObjectShare{AllowRead: true, Share: shareInterface})
	if err != nil {
		return nil, "", fmt.Errorf("%s error converting acm to permissions %s", ErrFailToNormalizePermissions.Error(), err.Error())
	}
	// Add read permissions from acm
	for _, permission := range sharePermissions {
		modifiedPermissions = append(modifiedPermissions, permission)
	}

	// Everyone tracking
	acmSaysEveryone := len(sharePermissions) == 0
	hasEveryone := false
	for _, permission := range permissions {
		if !permission.IsDeleted && permission.AllowRead && aacIsPermissionFor(permission, models.EveryoneGroup) {
			hasEveryone = true
			if !acmSaysEveryone {
				permission.IsDeleted = true
			}
		}
		if !permission.IsDeleted || !isCreating {
			modifiedPermissions = append(modifiedPermissions, permission)
		}
	}
	if acmSaysEveryone && !hasEveryone {
		everyonePermission := models.PermissionForGroup("", "", models.EveryoneGroup, false, true, false, false, false)
		modifiedPermissions = append(modifiedPermissions, everyonePermission)
	}

	// CRUDS for Owner
	ownerCRUDS, _ := models.PermissionForOwner(objectOwner)
	modifiedPermissions = append(modifiedPermissions, ownerCRUDS)
	sharePermissions = append(sharePermissions, ownerCRUDS)

	// Adjustments if has everyone, or not found in share
	for i := len(modifiedPermissions) - 1; i >= 0; i-- {
		permission := modifiedPermissions[i]
		// Identify whether permission is referenced in share
		foundInShare := false
		for _, sharePermission := range sharePermissions {
			if aacIsPermissionFor(permission, sharePermission.Grantee) {
				foundInShare = true
				break
			}
		}
		// Alter or recreate permissions if has everyone
		if (!foundInShare || acmSaysEveryone) && !permission.IsDeleted && permission.AllowRead && !aacIsPermissionFor(permission, models.EveryoneGroup) {
			replacementPermission := models.PermissionWithoutRead(permission)
			permission.IsDeleted = true
			modifiedPermissions[i] = permission
			modifiedPermissions = append(modifiedPermissions, replacementPermission)
		}
	}
	// Delete those granting nothing
	for i := len(modifiedPermissions) - 1; i >= 0; i-- {
		permission := modifiedPermissions[i]
		if !permission.AllowCreate &&
			!permission.AllowRead &&
			!permission.AllowUpdate &&
			!permission.AllowDelete &&
			!permission.AllowShare {
			if isCreating || permission.IsCreating() {
				modifiedPermissions = append(modifiedPermissions[:i], modifiedPermissions[i+1:]...)
			} else {
				permission.IsDeleted = true
				modifiedPermissions[i] = permission
			}
		}
	}

	// At this point, modifiedPermissions should reflect the overall state. Rebuild the ACM
	modifiedACM, err = aacRebuildACMFromPermissions(logger, modifiedPermissions, acm)
	if err != nil {
		return modifiedPermissions, modifiedACM, fmt.Errorf("%s error rebuilding acm %s", ErrFailToNormalizePermissions.Error(), err.Error())
	}

	return modifiedPermissions, modifiedACM, nil
}
func aacRebuildACMFromPermissions(logger zap.Logger, permissions []models.ODObjectPermission, acm string) (string, error) {
	var modifiedACM string
	var err error
	// Convert to an addressable map
	acmMap, err := utils.UnmarshalStringToMap(acm)
	if err != nil {
		return acm, fmt.Errorf("%s %s", ErrFailToRebuildACMFromPermissions, err.Error())
	}

	// Clear existing (defaults to share to everyone)
	delete(acmMap, acmShareKey)

	modifiedACM, err = utils.MarshalInterfaceToString(acmMap)
	if err != nil {
		return acm, fmt.Errorf("%s marshal error %s", ErrFailToRebuildACMFromPermissions, err.Error())
	}

	modifiedACM, err = aacInjectPermissionsIntoACM(logger, permissions, modifiedACM)
	if err != nil {
		return acm, fmt.Errorf("%s marshal error %s", ErrFailToRebuildACMFromPermissions, err.Error())
	}
	return modifiedACM, err
}