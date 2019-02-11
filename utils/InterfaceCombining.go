package utils

import (
	"reflect"
	"strings"
)

// deDupe creates a new interface with any duplicate attributes removed.
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
		return interfaceToAdd
	}
	aMap, ok := createMapFromInterface(interfaceToAdd)
	if !ok {
		return sMap
	}
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
					// recursive processing of this node
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
	return sMap
}

// createMapFromInterface converts an interface structure into a map data type.
func createMapFromInterface(sourceInterface interface{}) (map[string]interface{}, bool) {
	m, ok := sourceInterface.(map[string]interface{})
	return m, ok
}
