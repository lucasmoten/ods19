package utils

import (
	"encoding/json"
	"reflect"
)

// MarshalInterfaceToString accepts an input interface and returns either the raw string
// or seralized json string representation of the object passed in
func MarshalInterfaceToString(ainterface interface{}) (string, error) {
	// If no value provided, return empty
	if ainterface == nil {
		return "", nil
	}
	if reflect.TypeOf(ainterface).Kind().String() == "string" {
		// The interface is a string, return directly
		return ainterface.(string), nil
	}
	// The interface is an object, serialize to a string
	ainterfaceBytes, err := json.Marshal(ainterface)
	if err != nil {
		return "", err
	}
	return string(ainterfaceBytes[:]), nil
}

// UnmarshalStringToInterface takes a serialized string and unmarshals to a json object
func UnmarshalStringToInterface(astring string) (interface{}, error) {
	var result interface{}
	if err := json.Unmarshal([]byte(astring), &result); err != nil {
		return result, err
	}
	return result, nil
}
