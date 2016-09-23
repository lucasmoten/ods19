package server

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/uber-go/zap"
	"golang.org/x/net/context"

	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/utils"
)

func deDupe(sArr []interface{}) []interface{} {
	var oArr []interface{}
	for _, s := range sArr {
		b := false
		for _, o := range oArr {
			if s == o {
				b = true
				break
			}
		}
		if !b {
			oArr = append(oArr, s)
		}
	}
	return oArr
}

// CombineInterface combines two interfaces that use map[string]interface{}, []interface{} and string for values
func CombineInterface(sourceInterface interface{}, interfaceToAdd interface{}) interface{} {
	sMap, ok := createMapFromInterface(sourceInterface)
	if !ok {
		//log.Printf("Unable to create map from interface for sourceInterface, using intefaceToAdd")
		return interfaceToAdd
	}
	// sMapString, _ := utils.MarshalInterfaceToString(sMap)
	// log.Printf("sMap before: %s", sMapString)
	aMap, ok := createMapFromInterface(interfaceToAdd)
	if !ok {
		//log.Printf("Unable to create map from interface for interfaceToAdd, using sourceInterface")
		return sMap
	}
	//log.Printf("Comparing sourceInterface and interfaceToAdd")
	// Look at all keys in A
	for aK, aV := range aMap {
		if aV == nil {
			continue
		}
		// Flag for whether found
		aFound := false
		avType := reflect.TypeOf(aV).Kind().String()
		// Iterate all keys in source
		for sK, sV := range sMap {
			// If same
			if strings.Compare(sK, aK) == 0 {
				if sV == nil {
					continue
				}
				svType := reflect.TypeOf(sV).Kind().String()
				// Update flag
				aFound = true
				switch avType {
				case "string":
					switch svType {
					case "string":
						// only add if different
						if strings.Compare(sV.(string), aV.(string)) != 0 {
							// values differ, make new array
							vArray := make([]interface{}, 2)
							vArray[0] = sV.(string)
							vArray[1] = aV.(string)
							vArray = deDupe(vArray)
							sMap[sK] = vArray
						}
					case "slice":
						// append to existing
						iArray, ok := sV.([]interface{})
						if !ok {
							// failed to make array from existing value
							// replace with new value
							vArray := make([]interface{}, 1)
							vArray[0] = aV
							vArray = deDupe(vArray)
							sMap[sK] = vArray
						} else {
							// new interface array with space for extra element
							vArray := make([]interface{}, len(iArray)+1)
							// copy values
							for idx, iV := range iArray {
								vArray[idx] = iV
							}
							// add the new element
							vArray[len(vArray)-1] = aV
							// Assign to source
							vArray = deDupe(vArray)
							sMap[sK] = vArray
						}
					} // switch svType for avType = string
				case "slice":
					// convert to interface array
					iArray, ok := aV.([]interface{})
					if ok {
						switch svType {
						case "string":
							// make new array sized up to hold source's current value
							vArray := make([]interface{}, len(iArray)+1)
							// start with source value
							vArray[0] = sV
							// copy values
							for idx, iV := range iArray {
								vArray[idx+1] = iV
							}
							// Assign to source
							vArray = deDupe(vArray)
							sMap[sK] = vArray
						case "slice":
							sArray, ok := sV.([]interface{})
							if ok {
								// make new array sized up to hold both slices
								vArray := make([]interface{}, len(iArray)+len(sArray))
								// start with source values
								for idx, iV := range sArray {
									vArray[idx] = iV
								}
								// then add additional values
								for idx, iV := range iArray {
									vArray[len(sArray)+idx] = iV
								}
								// Assign to source
								vArray = deDupe(vArray)
								sMap[sK] = vArray
							} else {
								// source value is an unsupported type
								// so just assign the adding value in place
								sMap[sK] = aV
							}
						} // switch svType for avType = slice
					} else {
						// the value to be added is a slice of an
						// unsupported type, do nothing with it
					}
				case "map":
					// recurse
					sMap[sK] = CombineInterface(sV, aV)
				} // switch avType
			} // if the keys for source and adding maps iteration is matched
		}
		// If the key from A was not found in S
		if !aFound {
			// Add it
			sMap[aK] = aV
		}
	}
	// Done, all changes in sMap
	// sMapString, _ = utils.MarshalInterfaceToString(sMap)
	// log.Printf("sMap after: %s", sMapString)
	return sMap
}
func createMapFromInterface(sourceInterface interface{}) (map[string]interface{}, bool) {
	m, ok := sourceInterface.(map[string]interface{})
	return m, ok
}
func getACMMap(obj *models.ODObject) (*AppError, map[string]interface{}) {
	if !obj.RawAcm.Valid {
		return NewAppError(400, fmt.Errorf("The object has no valid ACM"), "Missing ACM"), nil
	}
	if len(obj.RawAcm.String) == 0 {
		return NewAppError(400, fmt.Errorf("The object has no ACM"), "Missing ACM"), nil
	}
	acmInterface, err := utils.UnmarshalStringToInterface(obj.RawAcm.String)
	if err != nil {
		return NewAppError(500, err, "ACM unparseable"), nil
	}
	acmMap, ok := createMapFromInterface(acmInterface)
	if !ok {
		return NewAppError(500, fmt.Errorf("ACM does not convert to map"), "ACM unparseable"), nil
	}
	return nil, acmMap
}
func getACMInterfacePart(obj *models.ODObject, acmKeySearch string) (*AppError, interface{}) {
	herr, acmMap := getACMMap(obj)
	if herr != nil {
		return herr, nil
	}
	var foundInterface interface{}
	for acmKey, acmValue := range acmMap {
		if strings.Compare(acmKey, acmKeySearch) == 0 {
			foundInterface = acmValue
			break
		}
	}
	return nil, foundInterface
}

func removeACMPart(ctx context.Context, obj *models.ODObject, acmKeySearch string) *AppError {
	return setACMPartFromInterface(ctx, obj, acmKeySearch, nil)
}
func setACMPartFromStringArray(ctx context.Context, obj *models.ODObject, acmKeySearch string, acmValues []string) *AppError {
	// Build the interface array of the values
	interfaceArray := make([]interface{}, len(acmValues))
	for i, v := range acmValues {
		interfaceArray[i] = v
	}
	// Assign it
	return setACMPartFromInterfaceArray(ctx, obj, acmKeySearch, interfaceArray)
}

func setACMPartFromInterfaceArray(ctx context.Context, obj *models.ODObject, acmKeySearch string, interfaceArray []interface{}) *AppError {
	return setACMPartFromInterface(ctx, obj, acmKeySearch, interfaceArray)
}
func setACMPartFromInterface(ctx context.Context, obj *models.ODObject, acmKeySearch string, interfaceValue interface{}) *AppError {
	// Get the map
	herr, acmMap := getACMMap(obj)
	if herr != nil {
		return herr
	}
	// Assign to the key in the map
	if interfaceValue != nil {
		acmMap[acmKeySearch] = interfaceValue
	} else {
		delete(acmMap, acmKeySearch)
	}
	// Convert to string
	newACM, err := utils.MarshalInterfaceToString(acmMap)
	if err != nil {
		return NewAppError(500, err, "Unable to update ACM")
	}
	normalizedNewACM, err := utils.NormalizeMarshalledInterface(newACM)
	if err != nil {
		return NewAppError(500, err, "Unable to normalize new ACM")
	}
	normalizedOriginalACM, err := utils.NormalizeMarshalledInterface(obj.RawAcm.String)
	if err != nil {
		return NewAppError(500, err, "Unable to normalize original ACM")
	}
	if strings.Compare(normalizedNewACM, normalizedOriginalACM) != 0 {
		LoggerFromContext(ctx).Debug("Changing value of ACM", zap.String("original acm", obj.RawAcm.String), zap.String("normalized original acm", normalizedOriginalACM), zap.String("normalized new acm", normalizedNewACM))
		obj.RawAcm.String = normalizedNewACM
	}
	return nil
}

func getStringArrayFromInterface(i interface{}) []string {
	var o []string
	if i != nil {
		v := i.([]interface{})
		for _, e := range v {
			if strings.Compare(reflect.TypeOf(e).Kind().String(), "string") == 0 {
				o = append(o, e.(string))
			}
		}
	}
	return o
}

func isAcmShareDifferent(acm1 string, acm2 string) (bool, *AppError) {

	different := true

	obj1 := models.ODObject{}
	obj2 := models.ODObject{}
	obj1.RawAcm.Valid = true
	obj1.RawAcm.String = acm1
	obj2.RawAcm.Valid = true
	obj2.RawAcm.String = acm2

	herr1, acmShareInterface1 := getACMInterfacePart(&obj1, "share")
	if herr1 != nil {
		return different, herr1
	}
	herr2, acmShareInterface2 := getACMInterfacePart(&obj2, "share")
	if herr2 != nil {
		return different, herr2
	}

	acmShareString1, err1 := utils.MarshalInterfaceToString(acmShareInterface1)
	if err1 != nil {
		return different, NewAppError(500, err1, "Error marshalling share interface to string")
	}
	acmShareString2, err2 := utils.MarshalInterfaceToString(acmShareInterface2)
	if err2 != nil {
		return different, NewAppError(500, err2, "Error marshalling share interface to string")
	}

	different = (strings.Compare(acmShareString1, acmShareString2) != 0)
	return different, nil
}

func normalizeObjectReadPermissions(ctx context.Context, obj *models.ODObject) *AppError {
	// Apply changes to obj.Permissions based upon what ACM has
	LoggerFromContext(ctx).Info("favoring acm")

	// Parse ACM Share as Permissions
	herr, shareInterface := getACMInterfacePart(obj, "share")
	if herr != nil {
		return herr
	}
	acmObjectShare := protocol.ObjectShare{}
	acmObjectShare.AllowRead = true
	acmObjectShare.Share = shareInterface
	acmPermissions, err := mapping.MapObjectShareToODPermissions(&acmObjectShare)
	if err != nil {
		return NewAppError(500, fmt.Errorf("ACM share does not convert to permissions"), "ACM share unparseable")
	}

	// From ACM, determine if permission for everyone should exist
	hasEveryone := false
	acmSaysEveryone := len(acmPermissions) == 0

	// Build a simple array of the existing permissions that grant read access
	var readGrants []string
	for _, permission := range obj.Permissions {
		if permission.AllowRead {
			readGrants = append(readGrants, permission.Grantee)
			// And track if we have everyone or not
			if isPermissionFor(&permission, models.EveryoneGroup) {
				LoggerFromContext(ctx).Info("permission grantee is everyone")
				hasEveryone = true
			} else {
				if obj.IsCreating() {
					// When creating and permission grants read but isnt everyone, then
					// thats an indicator that we shouldn't support read for everyone
					LoggerFromContext(ctx).Info("creating object with permission that grants read but isn't everyone", zap.String("grantee", permission.Grantee))
					acmSaysEveryone = false
				}
			}
		}
	}

	// Force Owner CRUDS
	ownerCRUDS := models.PermissionForUser(obj.OwnedBy.String, true, true, true, true, true)
	ownerR := models.PermissionForUser(obj.OwnedBy.String, false, true, false, false, false)
	if !acmSaysEveryone {
		// prep permission
		obj.Permissions = append(obj.Permissions, ownerCRUDS)
		// check if in the ACM grants already
		hasAcmGrantee := false
		for _, readGrantee := range readGrants {
			if strings.Compare(ownerCRUDS.Grantee, readGrantee) == 0 {
				hasAcmGrantee = true
				break
			}
		}
		if !hasAcmGrantee {
			// add to allowed grant
			readGrants = append(readGrants, ownerCRUDS.Grantee)
			// add to acmPermissions for later checks
			acmPermissions = append(acmPermissions, ownerCRUDS)
			obj.Permissions = append(obj.Permissions, ownerCRUDS)
			acmPermissions = append(acmPermissions, ownerR)
			obj.Permissions = append(obj.Permissions, ownerR)
			// inject into ACM share
			interfaceToAdd, err := utils.UnmarshalStringToInterface(ownerCRUDS.AcmShare)
			if err != nil {
				return NewAppError(500, fmt.Errorf("ACM share for owner CRUDS does not convert to interface"), "ACM share for owner unparseable")
			}
			combinedInterface := CombineInterface(shareInterface, interfaceToAdd)
			if herr := setACMPartFromInterface(ctx, obj, "share", combinedInterface); herr != nil {
				return herr
			}
			// inject into ACM f_share
			herr, fShareInterface := getACMInterfacePart(obj, "f_share")
			if herr != nil {
				return herr
			}
			fShareValues := getStringArrayFromInterface(fShareInterface)
			fShareFound := false
			for _, fShareValue := range fShareValues {
				if strings.Compare(fShareValue, ownerCRUDS.Grantee) == 0 {
					fShareFound = true
				}
			}
			if !fShareFound {
				fShareValues = append(fShareValues, ownerCRUDS.Grantee)
				if herr := setACMPartFromInterface(ctx, obj, "f_share", fShareValues); herr != nil {
					return herr
				}
			}
		}
	}

	// ACM is authoritative, so if it claims it shouldnt have everyone but we do...
	if hasEveryone && !acmSaysEveryone {
		// Remove everyone
		LoggerFromContext(ctx).Info("removing permissions indicating everyonegroup as grantee")
		removePermissionsForGrantee(obj, models.EveryoneGroup)
		hasEveryone = false
	}

	// Add everyone if needed
	if acmSaysEveryone && !hasEveryone {
		LoggerFromContext(ctx).Info("adding permission for everyone")
		everyonePermission := models.PermissionForGroup("", "", models.EveryoneGroup, false, true, false, false, false)
		obj.Permissions = append(obj.Permissions, everyonePermission)
		// Now we do have everyone
		hasEveryone = true
	}

	if hasEveryone {
		// Remove read only permissions that are not everyone
		for i := len(obj.Permissions) - 1; i >= 0; i-- {
			permission := obj.Permissions[i]
			if permission.AllowRead {
				if strings.Compare(permission.AcmGrantee.GroupName.String, models.EveryoneGroup) != 0 {
					if permission.IsReadOnly() {
						// A read only permission that isn't everyone when everyone is present can simply be removed.
						LoggerFromContext(ctx).Info("removing readonly permission that is not everyone", zap.String("grantee", permission.Grantee))
						if obj.IsCreating() || permission.IsCreating() {
							// creating object, remove from list
							obj.Permissions = append(obj.Permissions[:i], obj.Permissions[i+1:]...)
						} else {
							// updating object, mark permission for deletion
							obj.Permissions[i].IsDeleted = true
						}
					} else {
						// Has other permissions, need to update it
						LoggerFromContext(ctx).Info("removing read from grantee since acm gives read to everyone", zap.String("grantee", obj.Permissions[i].Grantee))
						replacementPermission := models.PermissionWithoutRead(obj.Permissions[i])
						replacementPermission.ExplicitShare = true
						if obj.IsCreating() {
							// creating object, can redfine it
							obj.Permissions[i] = replacementPermission
						} else {
							// updating object, have to add new in place, and mark old as deleted
							if !obj.Permissions[i].IsCreating() {
								obj.Permissions[i].IsDeleted = true
							} else {
								obj.Permissions = append(obj.Permissions[:i], obj.Permissions[+1:]...)
							}
							obj.Permissions = append(obj.Permissions, replacementPermission)
						}
					}
				}
			}
		}
	} else {
		// Add any missing grantees for read access from acm derived permissions not found in readGrants collected otherwise
		for _, acmPermission := range acmPermissions {
			hasAcmGrantee := false
			for _, readGrantee := range readGrants {
				if strings.Compare(acmPermission.Grantee, readGrantee) == 0 {
					hasAcmGrantee = true
					break
				}
			}
			if !hasAcmGrantee {
				LoggerFromContext(ctx).Info("adding grantee from acm", zap.String("grantee", acmPermission.Grantee))
				obj.Permissions = append(obj.Permissions, acmPermission)
				readGrants = append(readGrants, acmPermission.Grantee)
			}
		}
		// Remove read only permissions that are not found in acmGrants
		for i := len(obj.Permissions) - 1; i >= 0; i-- {
			permission := obj.Permissions[i]
			if permission.AllowRead {
				hasAcmGrantee := false
				for _, acmPermission := range acmPermissions {
					if isPermissionFor(&permission, acmPermission.Grantee) {
						hasAcmGrantee = true
						break
					}
				}
				if !hasAcmGrantee {
					if permission.IsReadOnly() {
						// A read only permission that isn't one of the acmGrantees can simply be removed.
						LoggerFromContext(ctx).Info("removing grantee not present in acm", zap.String("grantee", obj.Permissions[i].Grantee))
						if obj.IsCreating() || permission.IsCreating() {
							// creating object, remove from list
							obj.Permissions = append(obj.Permissions[:i], obj.Permissions[i+1:]...)
						} else {
							// updating object, mark permission for deletion
							obj.Permissions[i].IsDeleted = true
						}
					} else {
						// has other permissions, need to update it
						LoggerFromContext(ctx).Info("removing read from grantee not present in acm", zap.String("grantee", obj.Permissions[i].Grantee))
						replacementPermission := models.PermissionWithoutRead(obj.Permissions[i])
						replacementPermission.ExplicitShare = true
						if obj.IsCreating() {
							// creating object, can redfine it
							obj.Permissions[i] = replacementPermission
						} else {
							// updating object, have to add new in place, and mark old as deleted
							if !obj.Permissions[i].IsCreating() {
								obj.Permissions[i].IsDeleted = true
							} else {
								obj.Permissions = append(obj.Permissions[:i], obj.Permissions[+1:]...)
							}
							obj.Permissions = append(obj.Permissions, replacementPermission)
						}
					}
				}
			}
		}
	}

	// Remove any permissions that grant nothing
	for i := len(obj.Permissions) - 1; i >= 0; i-- {
		permission := obj.Permissions[i]
		if !permission.AllowCreate &&
			!permission.AllowDelete &&
			!permission.AllowRead &&
			!permission.AllowShare &&
			!permission.AllowUpdate {
			// nothing granted. remove it
			LoggerFromContext(ctx).Info("removing permission that does not grant capabilities", zap.String("grantee", permission.Grantee))
			if obj.IsCreating() || permission.IsCreating() {
				// creating object, remove from list
				obj.Permissions = append(obj.Permissions[:i], obj.Permissions[i+1:]...)
			} else {
				// updating object, mark permission for deletion
				obj.Permissions[i].IsDeleted = true
			}
		}
	}

	// No errors
	return nil
}
