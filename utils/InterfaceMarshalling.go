package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// MarshalInterfaceToString accepts an input interface and returns either the raw string
// or serialized json string representation of the object passed in
func MarshalInterfaceToString(aInterface interface{}) (string, error) {
	// If no value provided, return empty
	if aInterface == nil {
		return "", nil
	}
	if reflect.TypeOf(aInterface).Kind().String() == "string" {
		// The interface is a string, return directly
		return aInterface.(string), nil
	}
	// The interface is an object, serialize to a string
	aInterfaceBytes, err := json.Marshal(aInterface)
	if err != nil {
		return "", err
	}
	return string(aInterfaceBytes[:]), nil
}

// UnmarshalStringToInterface takes a serialized string and unmarshal to a json object
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
