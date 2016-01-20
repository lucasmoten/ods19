package models

import (
	"database/sql"
	"encoding/json"
)

/*
NullInt64 supports setting a null value for an integer datatype from a database
*/
type NullInt64 struct {
	sql.NullInt64
}

/*
MarshalJSON will return the jsonified expression of NullInt64
*/
func (r NullInt64) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Int64)
}
