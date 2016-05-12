package server

// // This is part of functions that need to begin by retrieving an object that was mentioned in uri
// func retrieveObject(d dao.DAO, re *regexp.Regexp, uri string, loadProperties bool) (models.ODObject, *AppError, error) {
// 	var object models.ODObject
// 	matchIndexes := re.FindStringSubmatchIndex(uri)
// 	var objectID string

// 	if len(matchIndexes) == 0 {
// 		msg := "URI doesnt have an object ID in it"
// 		return object, NewAppError(400, nil, msg), nil
// 	}
// 	objectID = uri[matchIndexes[2]:matchIndexes[3]]

// 	// If not valid, return
// 	if objectID == "" {
// 		msg := "URI provided by caller does not specify an object identifier"
// 		return object, NewAppError(400, nil, msg), nil
// 	}
// 	// Convert to byte
// 	objectIDByte, err := hex.DecodeString(objectID)
// 	if err != nil {
// 		msg := "Identifier provided by caller is not a hexidecimal string"
// 		return object, NewAppError(400, err, msg), err
// 	}
// 	// Retrieve from database
// 	var objectRequested models.ODObject
// 	objectRequested.ID = objectIDByte
// 	object, err = d.GetObject(objectRequested, loadProperties)
// 	if err != nil {
// 		msg := "cannot get object"
// 		return object, NewAppError(500, err, msg), err
// 	}
// 	return object, nil, nil
// }
