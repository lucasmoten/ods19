package models

import "time"

type DBState struct {
	//Date of first schema
	CreateDate time.Time `db:"createdDate"`
	//Date of last schema
	ModifedDate time.Time `db:"modifiedDate"`
	//Code should be using the same schema version as us
	SchemaVersion string `db:"schemaVersion"`
	//A unique id for this database instance
	Identifier string `db:"identifier"`
}
