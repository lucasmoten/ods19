package protocol

import (
	"encoding/json"
	"errors"
	"io"
)

// ObjectResultset encapsulates the Object defined herein as an array with
// resultset metric information to expose page size, page number, total rows,
// and page count information when retrieving from the data store
type ObjectResultset struct {
	// Resultset contains meta information about the resultset
	Resultset
	// Objects contains the list of objects in this (page of) results.
	Objects []Object `json:"objects,omitempty"`
}

// NewObjectResultsetFromJSONBody parses an ObjectResultset from a JSON body.
func NewObjectResultsetFromJSONBody(body io.Reader) (ObjectResultset, error) {
	var resultset ObjectResultset
	var err error
	if body == nil {
		return resultset, errors.New("JSON body was nil")
	}
	err = (json.NewDecoder(body)).Decode(&resultset)
	if err != nil {
		return resultset, err
	}
	return resultset, nil
}
