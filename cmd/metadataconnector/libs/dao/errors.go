package dao

import "errors"

var (
	errMissingID          = errors.New("Missing ID field.")
	errMissingChangeToken = errors.New("Missing ChangeToken.")
	errMissingModifiedBy  = errors.New("Object ModifiedBy was not specified for object being updated")
)
