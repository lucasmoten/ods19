package mapping

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
)

// MapODPermissionToPermission1_0 converts an internal ODPermission model to an
// API exposable Permission1_0
func MapODPermissionToPermission1_0(i *models.ODObjectPermission) protocol.Permission1_0 {
	o := protocol.Permission1_0{}
	o.Grantee = i.Grantee
	o.ProjectName = i.AcmGrantee.ProjectName.String
	o.ProjectDisplayName = i.AcmGrantee.ProjectDisplayName.String
	o.GroupName = i.AcmGrantee.GroupName.String
	o.UserDistinguishedName = i.AcmGrantee.UserDistinguishedName.String
	o.DisplayName = i.AcmGrantee.DisplayName.String
	o.AllowCreate = i.AllowCreate
	o.AllowRead = i.AllowRead
	o.AllowUpdate = i.AllowUpdate
	o.AllowDelete = i.AllowDelete
	o.AllowShare = i.AllowShare
	return o
}

// MapODCommonPermissionToCallerPermission converts an internal ODCommonPermission model
// to an API exposable Caller Permission with minimal fields filled
func MapODCommonPermissionToCallerPermission(i *models.ODCommonPermission) protocol.CallerPermission {
	o := protocol.CallerPermission{}
	o.AllowCreate = i.AllowCreate
	o.AllowRead = i.AllowRead
	o.AllowUpdate = i.AllowUpdate
	o.AllowDelete = i.AllowDelete
	o.AllowShare = i.AllowShare
	return o
}

// MapODPermissionsToPermissions1_0 converts an array of internal ODPermission
// models to an array of API exposable Permission
func MapODPermissionsToPermissions1_0(i *[]models.ODObjectPermission) []protocol.Permission1_0 {
	o := make([]protocol.Permission1_0, len(*i))
	for p, q := range *i {
		o[p] = MapODPermissionToPermission1_0(&q)
	}
	o = applyEveryonePermissionsIfExists(o)
	return o
}

func applyEveryonePermissionsIfExists(i []protocol.Permission1_0) []protocol.Permission1_0 {
	o := make([]protocol.Permission1_0, len(i))
	hasEveryone := false
	var everyonePermissions *protocol.Permission1_0
	for _, q := range i {
		if strings.Compare(q.GroupName, models.EveryoneGroup) == 0 {
			everyonePermissions = &q
			hasEveryone = true
			break
		}
	}
	if !hasEveryone {
		return i
	}
	for idx, q := range i {
		var permWithEveryone protocol.Permission1_0
		permWithEveryone.Grantee = q.Grantee
		permWithEveryone.ProjectName = q.ProjectName
		permWithEveryone.ProjectDisplayName = q.ProjectDisplayName
		permWithEveryone.GroupName = q.GroupName
		permWithEveryone.UserDistinguishedName = q.UserDistinguishedName
		permWithEveryone.DisplayName = q.DisplayName
		permWithEveryone.AllowCreate = q.AllowCreate || everyonePermissions.AllowCreate
		permWithEveryone.AllowRead = q.AllowRead || everyonePermissions.AllowRead
		permWithEveryone.AllowUpdate = q.AllowUpdate || everyonePermissions.AllowUpdate
		permWithEveryone.AllowDelete = q.AllowDelete || everyonePermissions.AllowDelete
		permWithEveryone.AllowShare = q.AllowShare || everyonePermissions.AllowShare
		o[idx] = permWithEveryone
	}

	return o
}

// MapODPermissionsToPermission converts an array of internal ODPermission
// models to an array of API exposable Permission and applies everyone permissions
func MapODPermissionsToPermission(i *[]models.ODObjectPermission) protocol.Permission {
	o := protocol.Permission{}
	create := []string{}
	read := []string{}
	update := []string{}
	delete := []string{}
	share := []string{}
	// create := make(map[string]bool)
	// read := make(map[string]bool)
	// update := make(map[string]bool)
	// delete := make(map[string]bool)
	// share := make(map[string]bool)
	hasEveryone := false
	var everyonePermissions *models.ODObjectPermission
	for _, q := range *i {
		if strings.Compare(q.Grantee, models.AACFlatten(models.EveryoneGroup)) == 0 {
			everyonePermissions = &q
			hasEveryone = true
			break
		}
	}
	for _, q := range *i {
		resourceName := q.GetResourceName()
		if (q.AllowCreate || (hasEveryone && everyonePermissions.AllowCreate)) && !valueInStringArray(create, resourceName) {
			create = append(create, resourceName)
		}
		if (q.AllowRead || (hasEveryone && everyonePermissions.AllowRead)) && !valueInStringArray(read, resourceName) {
			read = append(read, resourceName)
		}
		if (q.AllowUpdate || (hasEveryone && everyonePermissions.AllowUpdate)) && !valueInStringArray(update, resourceName) {
			update = append(update, resourceName)
		}
		if (q.AllowDelete || (hasEveryone && everyonePermissions.AllowDelete)) && !valueInStringArray(delete, resourceName) {
			delete = append(delete, resourceName)
		}
		if (q.AllowShare || (hasEveryone && everyonePermissions.AllowShare)) && !valueInStringArray(share, resourceName) {
			share = append(share, resourceName)
		}
	}
	sort.Strings(create)
	sort.Strings(read)
	sort.Strings(update)
	sort.Strings(delete)
	sort.Strings(share)
	for _, k := range create {
		o.Create.AllowedResources = append(o.Create.AllowedResources, k)
	}
	for _, k := range read {
		o.Read.AllowedResources = append(o.Read.AllowedResources, k)
	}
	for _, k := range update {
		o.Update.AllowedResources = append(o.Update.AllowedResources, k)
	}
	for _, k := range delete {
		o.Delete.AllowedResources = append(o.Delete.AllowedResources, k)
	}
	for _, k := range share {
		o.Share.AllowedResources = append(o.Share.AllowedResources, k)
	}
	return o
}

// I'm sure we have this kind of function somewhere else but i couldnt locate
func valueInStringArray(a []string, v string) bool {
	for _, s := range a {
		if v == s {
			return true
		}
	}
	return false
}

// MapObjectSharesToODPermissions takes an array of ObjectShare request, and
// converts to an array of ODObjectPermission with capability flags set and
// acmShare initialized with a single chare to check against AAC to get the
// unique flattened value
func MapObjectSharesToODPermissions(i *[]protocol.ObjectShare) ([]models.ODObjectPermission, error) {
	o := []models.ODObjectPermission{}
	for _, q := range *i {
		mappedPermissions, err := MapObjectShareToODPermissions(&q)
		if err != nil {
			return o, err
		}
		o = append(o, mappedPermissions...)
	}
	return o, nil
}

// MapObjectShareToODPermissions takes an protocol ObjectShare request, and
// converts to an array of ODObjectPermission with the capability flags set
// and acmShare initialized with a single share to check against AAC to get
// the unique flattened value
func MapObjectShareToODPermissions(i *protocol.ObjectShare) ([]models.ODObjectPermission, error) {
	o := []models.ODObjectPermission{}

	// Reference to interface
	shareInterface := i.Share

	// if no value, return empty
	if shareInterface == nil {
		return o, nil
	}

	// If interface is a string, assume single DN
	if reflect.TypeOf(shareInterface).Kind().String() == "string" {
		// Capture DN
		userValue := shareInterface.(string)
		if len(userValue) > 0 {
			permission := models.PermissionForUser(userValue, i.AllowCreate, i.AllowRead, i.AllowUpdate, i.AllowDelete, i.AllowShare)
			o = append(o, permission)
		}
		// And return it
		return o, nil
	}

	// Interface is an object and may contain multiple users and groups
	shareMap, ok := shareInterface.(map[string]interface{})
	if !ok {
		return o, fmt.Errorf("Share does not convert to map")
	}
	// Iterate the map
	for shareKey, shareValue := range shareMap {
		if strings.Compare(strings.ToLower(shareKey), "users") == 0 {
			// Expected format:
			//    "users":[
			//       "the distinguished name of a user"
			//      ,"the distinguished name of another user"
			//      ]
			if shareValue != nil {
				shareValueInterfaceArray := shareValue.([]interface{})
				for _, shareValueElement := range shareValueInterfaceArray {
					if strings.Compare(reflect.TypeOf(shareValueElement).Kind().String(), "string") == 0 {
						// Capture DN
						userValue := shareValueElement.(string)
						if len(userValue) > 0 {
							permission := models.PermissionForUser(userValue, i.AllowCreate, i.AllowRead, i.AllowUpdate, i.AllowDelete, i.AllowShare)
							o = append(o, permission)
						}
					}
				}
			}
		} else if strings.Compare(strings.ToLower(shareKey), "projects") == 0 {
			// Expected format:
			//    "projects":{
			//      "id of project":{
			//         "disp_nm":"display name of project"
			//        ,"groups":[
			//            "group 1 id"
			//           ,"group 2 id"
			//          ]
			//        }
			//     }
			if shareValue != nil {
				shareValueMap, ok := shareValue.(map[string]interface{})
				if !ok {
					return o, fmt.Errorf("Share 'projects' does not convert to map")
				}
				for projectKey, projectValue := range shareValueMap {
					// projectKey = "id of project"
					if projectValue != nil {
						projectValueMap, ok := projectValue.(map[string]interface{})
						if !ok {
							return o, fmt.Errorf("Share 'projects' for '%s' does not convert to map", projectKey)
						}
						// Capture display name for the project
						projectDisplayName := ""
						for projectFieldKey, projectFieldValue := range projectValueMap {
							if projectFieldValue != nil {
								if strings.Compare(strings.ToLower(projectFieldKey), "disp_nm") == 0 {
									if strings.Compare(projectFieldKey, "disp_nm") != 0 {
										return o, fmt.Errorf("Share 'projects' has a field that is not the correct case. %s should be 'disp_nm'", projectFieldKey)
									}
									if strings.Compare(reflect.TypeOf(projectFieldValue).Kind().String(), "string") == 0 {
										projectDisplayName = projectFieldValue.(string)
									} else {
										return o, fmt.Errorf("Share 'projects' has an unusable value for 'disp_nm' on key %s. Value is not a string", projectFieldKey)
									}
									break
								}
							}
						}
						// Now look for groups
						for projectFieldKey, projectFieldValue := range projectValueMap {
							if projectFieldValue != nil {
								if strings.Compare(strings.ToLower(projectFieldKey), "groups") == 0 {
									groupValueInterfaceArray := projectFieldValue.([]interface{})
									for _, groupValueElement := range groupValueInterfaceArray {
										if groupValueElement != nil {
											if strings.Compare(reflect.TypeOf(groupValueElement).Kind().String(), "string") == 0 {
												groupValue := groupValueElement.(string)
												if len(groupValue) > 0 {
													permission := models.PermissionForGroup(projectKey, projectDisplayName, groupValue, i.AllowCreate, i.AllowRead, i.AllowUpdate, i.AllowDelete, i.AllowShare)
													o = append(o, permission)
												}
											}
										}
									}
								}
							}
						}

					}
				}
			}
		} else {
			// Unknown structure. Warn? Error?
		}
	}

	// Done
	return o, nil

}

// MapPermissionToODPermissions converts a protocol permission object into an array of model permissions
func MapPermissionToODPermissions(i *protocol.Permission) ([]models.ODObjectPermission, error) {
	*i = i.Consolidated()
	var o []models.ODObjectPermission
	var err error
	permissions := make(map[string]models.ODObjectPermission)
	for _, resource := range i.Create.AllowedResources {
		permissions, err = mergeODPermissions(permissions, resource, models.ODObjectPermission{AllowCreate: true})
		if err != nil {
			return o, err
		}
	}
	for _, resource := range i.Read.AllowedResources {
		permissions, err = mergeODPermissions(permissions, resource, models.ODObjectPermission{AllowRead: true})
		if err != nil {
			return o, err
		}
	}
	for _, resource := range i.Update.AllowedResources {
		permissions, err = mergeODPermissions(permissions, resource, models.ODObjectPermission{AllowUpdate: true})
		if err != nil {
			return o, err
		}
	}
	for _, resource := range i.Delete.AllowedResources {
		permissions, err = mergeODPermissions(permissions, resource, models.ODObjectPermission{AllowDelete: true})
		if err != nil {
			return o, err
		}
	}
	for _, resource := range i.Share.AllowedResources {
		permissions, err = mergeODPermissions(permissions, resource, models.ODObjectPermission{AllowShare: true})
		if err != nil {
			return o, err
		}
	}
	for k, v := range permissions {
		if v.Grantee == k {
			o = append(o, v)
		}
	}
	return o, nil
}

// mergeODPermissions adds permission passed in to the permission currently assigned the resource, creating a new one as needed
func mergeODPermissions(permissions map[string]models.ODObjectPermission, resource string, permission models.ODObjectPermission) (map[string]models.ODObjectPermission, error) {
	flattened := protocol.GetFlattenedNameFromResource(resource)
	var err error
	if flattened != "" {
		mappedPermission, ok := permissions[flattened]
		if !ok {
			mappedPermission, err = models.CreateODPermissionFromResource(resource)
			if err != nil {
				return permissions, err
			}
		}
		mappedPermission.AllowCreate = mappedPermission.AllowCreate || permission.AllowCreate
		mappedPermission.AllowRead = mappedPermission.AllowRead || permission.AllowRead
		mappedPermission.AllowUpdate = mappedPermission.AllowUpdate || permission.AllowUpdate
		mappedPermission.AllowDelete = mappedPermission.AllowDelete || permission.AllowDelete
		mappedPermission.AllowShare = mappedPermission.AllowShare || permission.AllowShare
		permissions[flattened] = mappedPermission
	}
	return permissions, nil
}
