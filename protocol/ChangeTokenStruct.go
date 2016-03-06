package protocol

import (
	"encoding/json"
	"errors"
	"io"
	"log"
)

// ChangeTokenStruct is a nestable structure defining the ChangeToken attribute
// for items in Object Drive
type ChangeTokenStruct struct {
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken,omitempty"`
}

// NewChangeTokenStructFromJSONBody parses a ChangeTokenStruct from a JSON body.
func NewChangeTokenStructFromJSONBody(body io.ReadCloser) (ChangeTokenStruct, error) {
	var ct ChangeTokenStruct
	var err error
	if body == nil {
		return ct, errors.New("Cannot decode ChangeTokenStruct from nil JSON body")
	}
	err = (json.NewDecoder(body)).Decode(&ct)
	if err != nil {
		if err != io.EOF {
			log.Printf("Error parsing paging information in json: %v\n", err)
			return ct, err
		}
	}
	if ct.ChangeToken == "" {
		return ct, errors.New("ChangeTokenStruct cannot be empty")
	}
	return ct, nil
}
