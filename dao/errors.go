package dao

import "errors"

// Database errors
var (
	ErrMissingID          = errors.New("missing id field")
	ErrMissingChangeToken = errors.New("missing changetoken")
	ErrMissingModifiedBy  = errors.New("object modifiedby was not specified for object being updated")
	ErrNoRows             = errors.New("sql: no rows in result set")
	ErrMissingTypeID      = errors.New("missing typeid field")
)
