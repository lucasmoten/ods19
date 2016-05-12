package server

import (
	"io"
	"mime/multipart"
)

// getFormValueAsString reads a multipart value into a limited length byte
// array and returns it.
// TODO: Move to a utility file since this is useful for all other requests
// doing multipart.
// TODO: This effectively limits the acceptable length of a field to 1KB which
// is too restrictive for certain values (lengthy descriptions, abstracts, etc)
// which will need revisited
func getFormValueAsString(part *multipart.Part) (string, *AppError) {
	valueAsBytes := make([]byte, 10240)
	n, err := part.Read(valueAsBytes)
	if err != nil {
		if err != io.EOF {
			return "", NewAppError(400, err, "Unable to parse value from part")
		} else {
			return "", nil
		}
	} // if err != nil
	return string(valueAsBytes[0:n]), nil
}
