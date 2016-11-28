package utils

import (
	"encoding/json"
	"fmt"
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

// NormalizeMarshalledInterface leverages json unmarshal and marshal to normalize interface in alpha order
func NormalizeMarshalledInterface(i string) (string, error) {
	var normalizedInterface interface{}
	if err := json.Unmarshal([]byte(i), &normalizedInterface); err != nil {
		return i, err
	}
	normalizedBytes, err := json.Marshal(normalizedInterface)
	if err != nil {
		return i, err
	}
	return string(normalizedBytes[:]), nil
}

// UnmarshalStringToMap takes a serialized string and unmarshals to a json object and then converts to a map
func UnmarshalStringToMap(astring string) (map[string]interface{}, error) {
	i, err := UnmarshalStringToInterface(astring)
	if err != nil {
		return nil, err
	}
	oMap, ok := i.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("could not convert interface to map")
	}
	return oMap, nil
}
