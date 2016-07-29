package models

import (
	"database/sql"
	"encoding/json"
)

// NullString supports setting a null value for a string datatype from a database
type NullString struct {
	sql.NullString
}

// MarshalJSON will return the jsonified expression of NullString
func (r NullString) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String)
}

// UnmarshalJSON transforms a NullString into a type JSON understands.
func (r NullString) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		r.Valid = false
		return nil
	} else {
		r.String = string(b)
		return nil
	}
}

// ToNullString is a helper
func ToNullString(s string) NullString {
	return NullString{sql.NullString{String: s, Valid: s != ""}}
}
