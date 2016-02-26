package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
)

// NullInt64 supports setting a null value for an integer datatype from a database
type NullInt64 struct {
	sql.NullInt64
}

// MarshalJSON will return the jsonified expression of NullInt64
func (r NullInt64) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Int64)
}

// UnmarshalJSON transforms a NullInt64 into a type JSON understands.
func (r NullInt64) UnmarshalJSON(b []byte) error {
	s := string(b)
	if v, err := strconv.Atoi(s); err == nil {
		r.Int64 = int64(v)
		return nil
	}
	return errors.New("Invalid NullInt64: " + s)
}
