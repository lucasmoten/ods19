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
