package util

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
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

// FullDecode gets json body and ensures that we are done dealing with Body
func FullDecode(r io.ReadCloser, obj interface{}) error {
	d := json.NewDecoder(r)
	//Bad parse will be signalled by an unset object
	d.Decode(obj)
	return FinishBody(r)
}

// FinishBody ensures that body is completely consumed - call in a defer
func FinishBody(r io.ReadCloser) error {
	if r != nil {
		//This has the potential to run us out of memory, and I just did
		//run out of memory.
		_, err := io.Copy(ioutil.Discard, r)
		if err != nil && err != io.EOF && err.Error() != "http: read on closed response body" {
			log.Printf("FullDecode: %v", err)
			return err
		}
		r.Close()
	}
	return nil
}
