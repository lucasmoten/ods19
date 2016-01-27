package config

import (
	"fmt"
	"os"
	"reflect"
)

// ExpandEnvironmentVariables accepts a struct and through reflection iterates
// over fields expanding environment variables in those fields that are of type
// string, and recursively applying to any nested structs
func ExpandEnvironmentVariables(m interface{}) {
	// The type of the interface
	interfaceType := reflect.TypeOf(m)
	// Get dereferenced object as necessary
	if interfaceType.Kind() == reflect.Ptr {
		interfaceType = interfaceType.Elem()
	}
	// if interfaceType.Name() == "Value" {
	// 	panic("STOP!")
	// }
	// fmt.Println(interfaceType, interfaceType.Name())
	// Iterate over the fields in this interface
	fieldCount := interfaceType.NumField()
	for fieldNum := 0; fieldNum < fieldCount; fieldNum++ {
		fieldKind := interfaceType.Field(fieldNum).Type.Kind()
		switch fieldKind {
		case reflect.Float32, reflect.Float64, reflect.Bool, reflect.Int:
			// Nothing to process
		case reflect.Slice, reflect.Map:
			// Nothing to process
		case reflect.String:
			// For strings, get the value, expand it, and reassign if different
			currentValue := reflect.ValueOf(m).Elem().Field(fieldNum).String()
			newValue := os.ExpandEnv(currentValue)
			if currentValue != newValue {
				reflect.ValueOf(m).Elem().Field(fieldNum).SetString(newValue)
			}
		case reflect.Array:
			fmt.Println("reflect.Array")
		case reflect.Interface:
			fmt.Println("reflect.Interface")
			// Traverse through it
			ExpandEnvironmentVariables(fieldKind)
			// 		case reflect.Struct:
			// fmt.Println("reflect.Struct")
			// fieldStruct := reflect.ValueOf(m).Elem().Field(fieldNum)
			// ExpandEnvironmentVariables(fieldStruct)
		// case reflect.Ptr:
		// fmt.Println("reflect.Ptr")
		// fieldVal := reflect.ValueOf(m).Field(fieldNum).Elem()
		// // Nil check
		// if !fieldVal.IsValid() {
		// 	return
		// }
		// ExpandEnvironmentVariables(reflect.ValueOf(fieldVal))
		default:
			fmt.Println(fieldKind, "not supported in ExpandEnvironmentVariables")
		}
	}
}
