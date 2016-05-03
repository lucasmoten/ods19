package util

import (
	"encoding/json"
	"io"
)

// StringToInt8Slice converts a string to []int8. This conversion is required to
// talk to Java services. A Go []int8 is equivalent to a Java byte array.
func StringToInt8Slice(input string) ([]int8, error) {
	byteSliced := []byte(input)
	result := make([]int8, len(byteSliced))
	for i := 0; i < len(byteSliced); i++ {
		// TODO this can panic. Is this a case for panic/recover?
		result[i] = int8(byteSliced[i])
	}
	return result, nil
}

func FullDecode(r io.ReadCloser, obj interface{}) error {
	d := json.NewDecoder(r)
	err := d.Decode(obj)
	//drain the decoder completely. ignore the result.
	//the point is to read to EOF.
	var junk *interface{}
	d.Decode(junk)
	r.Close()
	return err
}
