package dao

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// getSanitizedPageNumber takes an input number, and ensures that it is no less
// than 1
func getSanitizedPageNumber(pageNumber int) int {
	if pageNumber < 1 {
		return 1
	}
	return pageNumber
}

// getSanitizedPageSize takes an input number, and ensures it is within the
// range of 1 .. 10000
func getSanitizedPageSize(pageSize int) int {
	if pageSize < 1 {
		return 1
	}
	if pageSize > 10000 {
		return 10000
	}
	return pageSize
}

// getLimit is used for determining the upper bound of records to request from
// the database, specifically pageNumber * pageSize
func getLimit(pageNumber int, pageSize int) int {
	return getSanitizedPageNumber(pageNumber) * getSanitizedPageSize(pageSize)
}

// getOffset is used for determining the lower bound of records to request from
// the database, starting with the first item on a given page based on size
func getOffset(pageNumber int, pageSize int) int {
	return getLimit(pageNumber, pageSize) - pageSize
}

// getPageCount determines the total number of pages that would exist when the
// totalRows and pageSize are known
func getPageCount(totalRows int, pageSize int) int {
	var pageCount int
	pageCount = totalRows / pageSize
	for (pageCount * pageSize) < totalRows {
		pageCount++
	}
	return pageCount
}

// GetRootObjects retrieves a list of Objects in Object Drive that are not
// nested beneath any other objects natively (natural parentId is null)
func GetRootObjects(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	limit := getLimit(pageNumber, pageSize)
	offset := getOffset(pageNumber, pageSize)
	query := `select sql_calc_found_rows * from object where isdeleted = 0 and parentid is null`
	if len(orderByClause) > 0 {
		query += ` order by ` + orderByClause
	} else {
		query += ` order by createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := db.Select(&response.Objects, query)
	if err != nil {
		print(err.Error())
	}
	err = db.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		print(err.Error())
	}
	response.PageNumber = pageNumber
	response.PageSize = pageSize
	response.PageRows = len(response.Objects)
	response.PageCount = getPageCount(response.TotalRows, pageSize)
	return response, err
}

// GetChildObjects retrieves a list of Objects in Object Drive that are nested
// beneath a specified object by parentID
func GetChildObjects(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int, parentID string) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	limit := getLimit(pageNumber, pageSize)
	offset := getOffset(pageNumber, pageSize)
	query := `select sql_calc_found_rows * from object where isdeleted = 0 and parentid = unhex(?)`
	if len(orderByClause) > 0 {
		query += ` order by ` + orderByClause
	} else {
		query += ` order by createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := db.Select(&response.Objects, query, parentID)
	if err != nil {
		print(err.Error())
	}
	err = db.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		print(err.Error())
	}
	response.PageNumber = pageNumber
	response.PageSize = pageSize
	response.PageRows = len(response.Objects)
	response.PageCount = getPageCount(response.TotalRows, pageSize)
	return response, err
}

// GetRootObjectsByOwner retrieves a list of Objects in Object Drive that are
// not nested beneath any other objects natively (natural parentId is null) and
// are owned by the specified user or group.
func GetRootObjectsByOwner(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int, owner string) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	limit := getLimit(pageNumber, pageSize)
	offset := getOffset(pageNumber, pageSize)
	query := `select sql_calc_found_rows * from object where isdeleted = 0 and parentid is null and ownedby = ?`
	if len(orderByClause) > 0 {
		query += ` order by ` + orderByClause
	} else {
		query += ` order by createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := db.Select(&response.Objects, query, owner)
	if err != nil {
		print(err.Error())
	}
	// TODO: This relies on sql_calc_found_rows from previous call, but I dont know if its guaranteed that the reference to db here
	// for this call would be the same as that used above from the built in connection pooling perspective.  If it isn't, then it
	// could conceivably get the result from a concurrent instance performing a similar operation.
	err = db.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		print(err.Error())
	}
	response.PageNumber = pageNumber
	response.PageSize = pageSize
	response.PageRows = len(response.Objects)
	response.PageCount = getPageCount(response.TotalRows, pageSize)
	return response, err
}

// GetChildObjectsByOwner retrieves a list of Objects in Object Drive that are
// nested beneath a specified object by parentID and are owned by the specified
// user or group
func GetChildObjectsByOwner(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int, parentID string, owner string) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	limit := getLimit(pageNumber, pageSize)
	offset := getOffset(pageNumber, pageSize)
	query := `select sql_calc_found_rows * from object where isdeleted = 0 and parentid = ? and ownedby = ?`
	if len(orderByClause) > 0 {
		query += ` order by ` + orderByClause
	} else {
		query += ` order by createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := db.Select(&response.Objects, query, parentID, owner)
	if err != nil {
		print(err.Error())
	}
	// TODO: This relies on sql_calc_found_rows from previous call, but I dont know if its guaranteed that the reference to db here
	// for this call would be the same as that used above from the built in connection pooling perspective.  If it isn't, then it
	// could conceivably get the result from a concurrent instance performing a similar operation.
	err = db.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		print(err.Error())
	}
	response.PageNumber = pageNumber
	response.PageSize = pageSize
	response.PageRows = len(response.Objects)
	response.PageCount = getPageCount(response.TotalRows, pageSize)
	return response, err
}

// GetPropertiesForObject retrieves the properties for a given object
func GetPropertiesForObject(db *sqlx.DB, objectID []byte) ([]models.ODObjectPropertyEx, error) {
	response := []models.ODObjectPropertyEx{}
	query := `select p.* from property p inner join object_property op on p.id = op.propertyid where p.isdeleted = 0 and op.isdeleted = 0 and op.objectid = ?`
	err := db.Select(&response, query, objectID)
	if err != nil {
		print(err.Error())
	}
	return response, err
}

// GetRootObjectsWithProperties retrieves a list of Objects and their Properties
// in Object Drive that are not nested beneath any other objects natively
// (natural parentId is null)
func GetRootObjectsWithProperties(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error) {
	response, err := GetRootObjects(db, orderByClause, pageNumber, pageSize)
	if err != nil {
		print(err.Error())
		return response, err
	}
	for _, object := range response.Objects {
		properties, err := GetPropertiesForObject(db, object.ID)
		if err != nil {
			print(err.Error())
			return response, err
		}
		object.Properties = properties
	}
	return response, err
}

// GetChildObjectsWithProperties retrieves a list of Objects and their
// Properties in Object Drive that are nested beneath the specified parent
// object
func GetChildObjectsWithProperties(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int, parentID string) (models.ODObjectResultset, error) {
	response, err := GetChildObjects(db, orderByClause, pageNumber, pageSize, parentID)
	if err != nil {
		print(err.Error())
		return response, err
	}
	for i := 0; i < len(response.Objects); i++ {
		properties, err := GetPropertiesForObject(db, response.Objects[i].ID)
		if err != nil {
			print(err.Error())
			return response, err
		}
		response.Objects[i].Properties = properties
	}
	return response, err
}

// GetRootObjectsWithPropertiesByOwner retrieves a list of Objects and their
// Properties in Object Drive that are not nested beneath any other objects
// natively (natural parentId is null) and are owned by the specified user or
// group.
func GetRootObjectsWithPropertiesByOwner(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int, owner string) (models.ODObjectResultset, error) {
	response, err := GetRootObjectsByOwner(db, orderByClause, pageNumber, pageSize, owner)
	if err != nil {
		print(err.Error())
		return response, err
	}
	for i := 0; i < len(response.Objects); i++ {
		properties, err := GetPropertiesForObject(db, response.Objects[i].ID)
		if err != nil {
			print(err.Error())
			return response, err
		}
		response.Objects[i].Properties = properties
	}
	return response, err
}

// GetChildObjectsWithPropertiesByOwner retrieves a list of Objects and their
// Properties in Object Drive that are nested beneath the specified object by
// parentID and are owned by the specified user or group.
func GetChildObjectsWithPropertiesByOwner(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int, parentID string, owner string) (models.ODObjectResultset, error) {
	response, err := GetChildObjectsByOwner(db, orderByClause, pageNumber, pageSize, parentID, owner)
	if err != nil {
		print(err.Error())
		return response, err
	}
	for i := 0; i < len(response.Objects); i++ {
		properties, err := GetPropertiesForObject(db, response.Objects[i].ID)
		if err != nil {
			print(err.Error())
			return response, err
		}
		response.Objects[i].Properties = properties
	}
	return response, err
}

// CreateObjectType adds a new object type definition to the database based upon
// the passed in object type settings.  At a minimm, createdBy and the name of
// the object type must exist.  Once added, the record is retrieved and the
// object type passed in by reference is updated with the remaining attributes
func CreateObjectType(db *sqlx.DB, objectType *models.ODObjectType) {
	// Setup the statement
	addObjectTypeStatement, err := db.Prepare(`insert object_type set createdBy = ?, name = ?, description = ?, contentConnector = ?`)
	if err != nil {
		print(err.Error())
	}
	// Add it
	result, err := addObjectTypeStatement.Exec(objectType.CreatedBy, objectType.Name, objectType.Description.String, objectType.ContentConnector.String)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if rowCount < 1 {
		fmt.Println("No rows added from inserting object type")
		return
	}
	// Get the ID of the newly created object type and assign to passed in objectType
	getObjectTypeStatement := `select * from object_type where createdBy = ? and name = ? and isdeleted = 0 order by createdDate desc limit 1`
	err = db.Get(objectType, getObjectTypeStatement, objectType.CreatedBy, objectType.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Type was not found even after just adding!")
		}
	}
}

// GetObjectTypeByName looks up an object type by its name, and if it doesn't
// exist, optionally calls CreateObjectType to add it.
func GetObjectTypeByName(db *sqlx.DB, typeName string, addIfMissing bool, createdBy string) models.ODObjectType {
	var objectType models.ODObjectType
	// Get the ID of the newly created object and assign to passed in object
	getObjectTypeStatement := `select * from object_type where name = ?	and isdeleted = 0 order by createddate desc limit 1`
	err := db.Get(&objectType, getObjectTypeStatement, typeName)
	if err != nil {
		if err == sql.ErrNoRows {
			if addIfMissing {
				objectType.Name = typeName
				objectType.CreatedBy = createdBy
				CreateObjectType(db, &objectType)
			} // if addIfMissing {
		} else {
			panic(err)
		} // if err == sql.NoRows
	} // if err != nil

	return objectType
}

// CreateUser adds a new user definition to the database based upon the passed
// in ODUser object settings. At a minimm, createdBy and the distinguishedName
// of the user must already be assigned.  Once added, the record is retrieved
// and the user passed in by reference is updated with the remaining attributes
func CreateUser(db *sqlx.DB, user *models.ODUser) {
	addUserStatement, err := db.Prepare(`insert user set createdBy = ?, distinguishedName = ?, displayName = ?, email = ?`)
	if err != nil {
		panic(err)
	}

	result, err := addUserStatement.Exec(user.CreatedBy, user.DistinguishedName, "", "")
	if err != nil {
		panic(err)
	}
	rowCount, err := result.RowsAffected()
	if rowCount < 1 {
		fmt.Println("No rows added from inserting user")
	}
	getUserStatement := `select * from user where distinguishedName = ?`
	err = db.Get(user, getUserStatement, user.DistinguishedName)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("User was not found even after just adding!")
		}
	}
}

// GetUserByDistinguishedName looks up user record from the database using the
// provided distinguished name
func GetUserByDistinguishedName(db *sqlx.DB, distinguishedName string) models.ODUser {
	var user models.ODUser
	getUserStatement := `select * from user where distinguishedName = ?`
	err := db.Get(&user, getUserStatement, distinguishedName)
	if err != nil {
		if err == sql.ErrNoRows {
			user.DistinguishedName = distinguishedName
			user.CreatedBy = distinguishedName
			CreateUser(db, &user)
		} // if err == sql.NoRows
	} // if err != nil
	return user
}

// CreateObject uses the passed in object and acm configuration and makes the
// appropriate sql calls to the database to insert the object, insert the acm
// configuration, associate the two together. Identifiers are captured and
// assigned to the relevant objects
func CreateObject(db *sqlx.DB, object *models.ODObject, acm *models.ODACM) error {

	// lookup type, assign its id to the object for reference
	if object.TypeID == nil {
		objectType := GetObjectTypeByName(db, object.TypeName.String, true, object.CreatedBy)
		object.TypeID = objectType.ID
	}

	// insert object
	addObjectStatement, err := db.Prepare(`insert object set createdBy = ?, typeId = ?, name = ?, description = ?, parentId = ?, contentConnector = ?, encryptIV = ?, encryptKey = ?, contentType = ?, contentSize = ? `)
	if err != nil {
		print(err.Error())
	}
	// Add it
	result, err := addObjectStatement.Exec(object.CreatedBy, object.TypeID,
		object.Name, object.Description.String, object.ParentID,
		object.ContentConnector.String, object.EncryptIV.String,
		object.EncryptKey.String, object.ContentType.String,
		object.ContentSize)
	if err != nil {
		print(err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		panic(err)
	}
	if rowsAffected <= 0 {
		panic("Object inserted but no rows affected")
	}
	// Get the ID of the newly created object and assign to passed in object
	// The following block uses all parameters but doesnt take into account null
	// values...
	// getObjectStatement := `select * from object where createdby = ? and typeId = ? and name = ? and description = ? and parentId = ? and contentConnector = ? and encryptIV = ? and encryptKey = ? and contentType = ? and contentSize = ? and isdeleted = 0 order by createddate desc limit 1`
	// err = db.Get(object, getObjectStatement, object.CreatedBy, object.TypeID,
	// 	object.Name, object.Description.String, object.ParentID,
	// 	object.ContentConnector.String, object.EncryptIV.String,
	// 	object.EncryptKey.String, object.ContentType.String,
	// 	object.ContentSize)
	getObjectStatement := `select * from object where createdby = ? and typeId = ? and name = ? and isdeleted = 0 order by createddate desc limit 1`
	err = db.Get(object, getObjectStatement, object.CreatedBy, object.TypeID, object.Name)
	if err != nil {
		panic(err)
	}

	// TODO: add properties of object.Properties []models.ODObjectPropertyEx

	// insert acm

	return nil
}

// AddPropertyToObject creates a new property with the provided name and value,
// and then associates that Property object to the Object indicated by ObjectID
func AddPropertyToObject(db *sqlx.DB, createdBy string, objectID []byte, propertyName string, propertyValue string, classificationPM string) {
	// Setup the statement
	addPropertyStatement, err := db.Prepare(`insert property set createdby = ?, name = ?, propertyvalue = ?, classificationpm = ?`)
	if err != nil {
		print(err.Error())
	}
	// Add it
	result, err := addPropertyStatement.Exec(createdBy, propertyName, propertyValue, classificationPM)
	if err != nil {
		print(err.Error())
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if rowCount < 1 {
		print("No rows added from inserting property")
	}
	// Get the ID of the newly created property
	var newPropertyID []byte
	getPropertyIDStatement, err := db.Prepare(`select id from property where createdby = ? and name = ? and propertyvalue = ? and classificationpm = ? order by createddate desc limit 1`)
	if err != nil {
		print(err.Error())
	}
	err = getPropertyIDStatement.QueryRow(createdBy, propertyName, propertyValue, classificationPM).Scan(&newPropertyID)
	if err != nil {
		log.Fatal(err)
	}
	// Add association to the object
	addObjectPropertyStatement, err := db.Prepare(`insert object_property set createdby = ?, objectid = ?, propertyid = ?`)
	result, err = addObjectPropertyStatement.Exec(createdBy, objectID, newPropertyID)
	if err != nil {
		log.Fatal(err)
	}
}
