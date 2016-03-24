package server

import (
	"encoding/hex"

	"regexp"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

// This is part of functions that need to begin by retrieving an object that was mentioned in uri
func retrieveObject(d dao.DAO, re *regexp.Regexp, uri string) (models.ODObject, *AppError, error) {
	var object models.ODObject
	matchIndexes := re.FindStringSubmatchIndex(uri)
	var objectID string

	if len(matchIndexes) == 0 {
		msg := "URI doesnt have an object ID in it"
		return object, &AppError{400, nil, msg}, nil
	}
	objectID = uri[matchIndexes[2]:matchIndexes[3]]

	// If not valid, return
	if objectID == "" {
		msg := "URI provided by caller does not specify an object identifier"
		return object, &AppError{400, nil, msg}, nil
	}
	// Convert to byte
	objectIDByte, err := hex.DecodeString(objectID)
	if err != nil {
		msg := "Identifier provided by caller is not a hexidecimal string"
		return object, &AppError{400, err, msg}, err
	}
	// Retrieve from database
	var objectRequested models.ODObject
	objectRequested.ID = objectIDByte
	object, err = d.GetObject(objectRequested, false)
	if err != nil {
		msg := "cannot get object"
		return object, &AppError{500, err, msg}, err
	}
	return object, nil, nil
}
