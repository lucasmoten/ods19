package models

import (
	"encoding/json"

	"github.com/go-sql-driver/mysql"
)

// NullTime supports setting a null value for a timestamp datatype from a database
type NullTime struct {
	mysql.NullTime
}

// MarshalJSON will return the jsonified expression of NullTime if considered
// valid or nil otherwise
func (r NullTime) MarshalJSON() ([]byte, error) {
	if r.Valid {
		return json.Marshal(r.Time)
	}
	return json.Marshal(nil)
}
