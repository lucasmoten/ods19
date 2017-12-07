package protocol

import (
	"fmt"
	"strings"

	"github.com/deciphernow/object-drive-server/metadata/models"
)

// CallerPermission is a structure defining the attributes for
// permissions granted on an object for the caller of an operation where an
// object is returned
type CallerPermission struct {
	// AllowCreate indicates whether the caller has permission to create child
	// objects beneath this object
	AllowCreate bool `json:"allowCreate"`
	// AllowRead indicates whether the caller has permission to read this
	// object. This is the most fundamental permission granted, and should always
	// be true as only records need to exist where permissions are granted as
	// the system denies access by default. Read access to an object is necessary
	// to perform any other action on the object.
	AllowRead bool `json:"allowRead"`
	// AllowUpdate indicates whether the caller has permission to update this
	// object
	AllowUpdate bool `json:"allowUpdate"`
	// AllowDelete indicates whether the caller has permission to delete this
	// object
	AllowDelete bool `json:"allowDelete"`
	// AllowShare indicates whether the caller has permission to view and
	// alter permissions on this object
	AllowShare bool `json:"allowShare"`
}

// String satisfies the Stringer interface for CallerPermission.
func (cp CallerPermission) String() string {
	template := "[%s%s%s%s%s]"
	s := fmt.Sprintf(template, iifString(cp.AllowCreate, "C", "-"), iifString(cp.AllowRead, "R", "-"), iifString(cp.AllowUpdate, "U", "-"), iifString(cp.AllowDelete, "D", "-"), iifString(cp.AllowShare, "S", "-"))
	return s
}

// WithRolledUp takes a permission declaration of allowed and denied resources, and determines whether the caller has permission for each capability
func (cp CallerPermission) WithRolledUp(caller Caller, permission Permission) CallerPermission {
	cp.AllowCreate = IsInResources(caller, permission.Create.AllowedResources) && !IsInResources(caller, permission.Create.DeniedResources)
	cp.AllowRead = IsInResources(caller, permission.Read.AllowedResources) && !IsInResources(caller, permission.Read.DeniedResources)
	cp.AllowUpdate = IsInResources(caller, permission.Update.AllowedResources) && !IsInResources(caller, permission.Update.DeniedResources)
	cp.AllowDelete = IsInResources(caller, permission.Delete.AllowedResources) && !IsInResources(caller, permission.Delete.DeniedResources)
	cp.AllowShare = IsInResources(caller, permission.Share.AllowedResources) && !IsInResources(caller, permission.Share.DeniedResources)
	return cp
}

// IsInResources examines if the caller distinguished name, or the everyone group is contained within a passed in list of resources
func IsInResources(caller Caller, resources []string) bool {
	for _, resource := range resources {
		flattened := GetFlattenedNameFromResource(resource)
		if flattened == models.AACFlatten(caller.DistinguishedName) || flattened == models.AACFlatten(models.EveryoneGroup) || caller.InGroup(flattened) {
			return true
		}
	}
	return false
}

// GetFlattenedNameFromResource returns the equivalent of AAC flattened share/f_share from a resource
func GetFlattenedNameFromResource(resource string) (flattened string) {
	if len(resource) == 0 {
		return ""
	}
	if strings.HasPrefix(resource, "user/") {
		return GetFlattenedUserFromResource(resource)
	}
	if strings.HasPrefix(resource, "group/") {
		return GetFlattenedGroupFromResource(resource)
	}
	if strings.HasPrefix(resource, "unknown/") {
		return GetFlattenedUnknownFromResource(resource)
	}
	return models.AACFlatten(resource)
}

// GetFlattenedUserFromResource returns the equivalent of AAC flattened share/f_share from a user/ resource
func GetFlattenedUserFromResource(resource string) (flattened string) {
	o := strings.Split(strings.Replace(resource, "user/", "", 1), "/")
	if len(o) > 0 {
		return models.AACFlatten(o[0])
	}
	return ""
}

// GetFlattenedGroupFromResource returns the equivalent of AAC flattened share/f_share from a group/ resource
func GetFlattenedGroupFromResource(resource string) (flattened string) {
	o := strings.Split(strings.Replace(resource, "group/", "", 1), "/")
	switch len(o) {
	case 1: // groupName
		fallthrough
	case 2: // groupName/groupDisplayName
		return models.AACFlatten(o[0])
	case 3: // projectName/projectDisplayName/groupName
		fallthrough
	case 4: // projectName/projectDisplayName/groupName/groupDisplayName
		return models.AACFlatten(o[0] + "_" + o[2])
	}
	return ""
}

// GetFlattenedUnknownFromResource returns the equivalent of AAC flattened share/f_share from a unknown/ resource
func GetFlattenedUnknownFromResource(resource string) (flattened string) {
	o := strings.Split(strings.Replace(resource, "unknown/", "", 1), "/")
	if len(o) > 0 {
		return models.AACFlatten(o[0])
	}
	return ""
}
