package util

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
)

const (
	errClosedResponseBody = "http: read on closed response body"
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
	err := d.Decode(obj)
	FinishBody(r)
	return err
}

// FinishBody ensures that body is completely consumed - call in a defer
func FinishBody(r io.ReadCloser) {
	if r != nil {
		//Throw the bytes away if there are any
		_, copyErr := io.Copy(ioutil.Discard, r)
		//Close if it's not closed
		closeErr := r.Close()
		//The best we can do is to log these.
		//All err values I have ever seen are not errors at all.
		if copyErr != nil {
			if copyErr != io.EOF && copyErr.Error() != errClosedResponseBody {
				log.Printf("FinishBody cannot discard bytes: %v", copyErr)
			}
		}
		if closeErr != nil {
			log.Printf("FinishBody cannot close: %v", closeErr)
		}
	}
}
