package dao

import "errors"

var (
	ErrMissingID          = errors.New("Missing ID field.")
	ErrMissingChangeToken = errors.New("Missing ChangeToken.")
	ErrMissingModifiedBy  = errors.New("Object ModifiedBy was not specified for object being updated")
	ErrNoRows             = errors.New("sql: no rows in result set")
)
