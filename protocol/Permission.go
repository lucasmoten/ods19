package protocol

// Permission is a nestable structure defining the attributes for
// permissions granted on an object for users who have access to the object
// in Object Drive
type Permission struct {
	// Create contains the resources who are allowed to create child objects
	Create PermissionCapability `json:"create,omitempty"`
	// Read contains the resources who are allowed to read this object metadata,
	// list its contents or view its stream
	Read PermissionCapability `json:"read,omitempty"`
	// Update contains the resources who are allowed to update the object
	// metadata or stream
	Update PermissionCapability `json:"update,omitempty"`
	// Delete contains the resources who are allowed to delete this object,
	// restore it from the trash, or expunge it forever.
	Delete PermissionCapability `json:"delete,omitempty"`
	// Share contains the resources who are allowed to alter the permissions
	// on this object directly, or indirectly via metadata such as acm.
	Share PermissionCapability `json:"share,omitempty"`
}

// Consolidated consolidates permission by capability taking into account
// removal of duplicate resource entries, and removing resources from
// allowed list if also appears in denied list
func (permission Permission) Consolidated() Permission {
	permission.Create = permission.Create.RemoveDuplicates()
	permission.Read = permission.Read.RemoveDuplicates()
	permission.Update = permission.Update.RemoveDuplicates()
	permission.Delete = permission.Delete.RemoveDuplicates()
	permission.Share = permission.Share.RemoveDuplicates()
	return permission
}

// PermissionCapability contains the list of resources who are allowed or denied
// the referenced capability
type PermissionCapability struct {
	// AllowedResources is a list of resources who are permitted this capability
	AllowedResources []string `json:"allow,omitempty"`
	// DeniedResources is a list of resources who will be denied this capability
	// even if allowed through other means.
	DeniedResources []string `json:"deny,omitempty"`
}

// RemoveDuplicates removes duplicate entries from AllowedResources and
// DeniedResources and then removes any entries from AllowedResources that
// appears in DeniedResources
func (capability PermissionCapability) RemoveDuplicates() PermissionCapability {
	allowed := make(map[string]bool)
	denied := make(map[string]bool)
	newCapability := PermissionCapability{}
	for _, r := range capability.AllowedResources {
		allowed[r] = true
	}
	for _, r := range capability.DeniedResources {
		denied[r] = true
		allowed[r] = false
	}
	for k, v := range allowed {
		if v {
			newCapability.AllowedResources = append(newCapability.AllowedResources, k)
		}
	}
	for k, v := range denied {
		if v {
			newCapability.DeniedResources = append(newCapability.DeniedResources, k)
		}
	}
	return newCapability
}
