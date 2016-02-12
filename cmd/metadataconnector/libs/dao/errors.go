package dao

import "errors"

var (
	errMissingID          = errors.New("Missing ID field.")
	errMissingChangeToken = errors.New("Missing ChangeToken.")
)
