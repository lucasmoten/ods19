package models

import "time"

type DBState struct {
	//Date of first schema
	CreateDate time.Time
	//Date of last schema
	ModifedDate time.Time
	//Code should be using the same schema version as us
	SchemaVersion string
	//A unique id for this database instance
	Identifier string
}
