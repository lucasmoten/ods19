package mapping

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

// MapODPermissionToPermission converts an internal ODPermission model to an
// API exposable Permission
func MapODPermissionToPermission(i *models.ODObjectPermission) protocol.Permission {
	o := protocol.Permission{}
	o.ID = hex.EncodeToString(i.ID)
	o.CreatedDate = i.CreatedDate
	o.CreatedBy = i.CreatedBy
	o.ModifiedDate = i.ModifiedDate
	o.ModifiedBy = i.ModifiedBy
	o.ChangeCount = i.ChangeCount
	o.ChangeToken = i.ChangeToken
	o.ObjectID = hex.EncodeToString(i.ObjectID)
	o.Grantee = i.Grantee
	o.AllowCreate = i.AllowCreate
	o.AllowRead = i.AllowRead
	o.AllowUpdate = i.AllowUpdate
	o.AllowDelete = i.AllowDelete
	o.AllowShare = i.AllowShare
	o.ExplicitShare = i.ExplicitShare
	return o
}

// MapODCommonPermissionToPermission converts an internal ODCommonPermission model
// to an API exposable Permission with minimal fields filled
func MapODCommonPermissionToCallerPermission(i *models.ODCommonPermission) protocol.CallerPermission {
	o := protocol.CallerPermission{}
	o.AllowCreate = i.AllowCreate
	o.AllowRead = i.AllowRead
	o.AllowUpdate = i.AllowUpdate
	o.AllowDelete = i.AllowDelete
	o.AllowShare = i.AllowShare
	return o
}

// MapODPermissionsToPermissions converts an array of internal ODPermission
// models to an array of API exposable Permission
func MapODPermissionsToPermissions(i *[]models.ODObjectPermission) []protocol.Permission {
	o := make([]protocol.Permission, len(*i))
	for p, q := range *i {
		o[p] = MapODPermissionToPermission(&q)
	}
	return o
}

// MapPermissionToODPermission converts an API exposable Permission object to
// an internally usable ODPermission model
func MapPermissionToODPermission(i *protocol.Permission) (models.ODObjectPermission, error) {
	var err error
	o := models.ODObjectPermission{}

	// ID convert string to byte, reassign to nil if empty
	ID, err := hex.DecodeString(i.ID)
	if err != nil {
		return o, fmt.Errorf("Unable to decode id from %s", i.ID)
	}
	if len(o.ID) == 0 {
		o.ID = nil
	} else {
		o.ID = ID
	}

	o.CreatedDate = i.CreatedDate
	o.CreatedBy = i.CreatedBy
	o.ModifiedDate = i.ModifiedDate
	o.ModifiedBy = i.ModifiedBy
	o.ChangeCount = i.ChangeCount
	o.ChangeToken = i.ChangeToken

	// Object ID convert string to byte, reassign to nil if empty
	objectID, err := hex.DecodeString(i.ObjectID)
	if err != nil {
		return o, fmt.Errorf("Unable to decode object id from %s", i.ObjectID)
	}
	if len(o.ObjectID) == 0 {
		o.ObjectID = nil
	} else {
		o.ObjectID = objectID
	}

	o.Grantee = i.Grantee
	o.AllowCreate = i.AllowCreate
	o.AllowRead = i.AllowRead
	o.AllowUpdate = i.AllowUpdate
	o.AllowDelete = i.AllowDelete
	o.AllowShare = i.AllowShare
	o.ExplicitShare = i.ExplicitShare
	return o, nil
}

// MapPermissionsToODPermissions converts an array of API exposable Permission
// objects into an array of internally usable ODPermission model objects
func MapPermissionsToODPermissions(i *[]protocol.Permission) ([]models.ODObjectPermission, error) {
	o := make([]models.ODObjectPermission, len(*i))
	for p, q := range *i {
		mappedPermission, err := MapPermissionToODPermission(&q)
		if err != nil {
			return o, err
		}
		o[p] = mappedPermission
	}
	return o, nil
}

// MapObjectShareToODPermissions takes an protocol ObjectShare request, and
// converts to an array of ODObjectPermission with the capability flags set
// and acmShare initialized with a single share to check against AAC to get
// the unique flattened value
func MapObjectShareToODPermissions(i *protocol.ObjectShare) ([]models.ODObjectPermission, error) {
	o := []models.ODObjectPermission{}

	acmShareUser := "{\"users\":[\"%s\"]}"
	acmShareGroup := "{\"projects\":{\"%s\":{\"disp_nm\":\"%s\",\"groups\":[\"%s\"]}}}"

	// Reference to interface
	shareInterface := i.Share

	// if no value, return empty
	if shareInterface == nil {
		return o, nil
	}

	// If interface is a string, assume single DN
	if reflect.TypeOf(shareInterface).Kind().String() == "string" {
		// Prep permission
		permission := mapODObjectShareToODPermission(i)
		// DN assignment to AcmShare
		userDN := shareInterface.(string)
		permission.AcmShare = fmt.Sprintf(acmShareUser, userDN)
		// Append to array
		o = append(o, permission)
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
		if strings.Compare(shareKey, "users") == 0 {
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
							// Prep permission
							permission := mapODObjectShareToODPermission(i)
							// DN assignment to AcmShare
							permission.AcmShare = fmt.Sprintf(acmShareUser, userValue)
							// Append to Array
							o = append(o, permission)
						}
					}
				}
			}
		} else if strings.Compare(shareKey, "projects") == 0 {
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
						projectDisplayName := ""
						for projectFieldKey, projectFieldValue := range projectValueMap {
							if projectFieldValue != nil {
								if strings.Compare(projectFieldKey, "disp_nm") == 0 {
									if strings.Compare(reflect.TypeOf(projectFieldValue).Kind().String(), "string") == 0 {
										projectDisplayName = projectFieldValue.(string)
									}
								} else if strings.Compare(projectFieldKey, "groups") == 0 {
									groupValueInterfaceArray := projectFieldValue.([]interface{})
									for _, groupValueElement := range groupValueInterfaceArray {
										if groupValueElement != nil {
											if strings.Compare(reflect.TypeOf(groupValueElement).Kind().String(), "string") == 0 {
												groupValue := groupValueElement.(string)
												if len(groupValue) > 0 {
													// Prep permission
													permission := mapODObjectShareToODPermission(i)
													// Group assignment to AcmShare
													permission.AcmShare = fmt.Sprintf(acmShareGroup, projectKey, projectDisplayName, groupValue)
													// Append to Array
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

func mapODObjectShareToODPermission(i *protocol.ObjectShare) models.ODObjectPermission {
	o := models.ODObjectPermission{}
	o.AllowCreate = i.AllowCreate
	o.AllowRead = i.AllowRead
	o.AllowUpdate = i.AllowUpdate
	o.AllowDelete = i.AllowDelete
	o.AllowShare = i.AllowShare
	return o
}
