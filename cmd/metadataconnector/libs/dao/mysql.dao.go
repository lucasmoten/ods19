package dao

import (
	"log"
	"strconv"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

func getSanitizedPageNumber(pageNumber int) int {
	if pageNumber < 1 {
		return 1
	}
	return pageNumber
}
func getSanitizedPageSize(pageSize int) int {
	if pageSize < 1 {
		return 1
	}
	if pageSize > 10000 {
		return 10000
	}
	return pageSize
}
func getLimit(pageNumber int, pageSize int) int {
	return getSanitizedPageNumber(pageNumber) * getSanitizedPageSize(pageSize)
}
func getOffset(pageNumber int, pageSize int) int {
	return getLimit(pageNumber, pageSize) - pageSize
}
func getPageCount(totalRows int, pageSize int) int {
	var pageCount int
	pageCount = totalRows / pageSize
	for (pageCount * pageSize) < totalRows {
		pageCount++
	}
	return pageCount
}

/*
GetRootObjects retrieves a list of Objects in Object Drive that are not nested
beneath any other objects natively (natural parentId is null)
*/
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

/*
GetRootObjectsByOwner retrieves a list of Objects in Object Drive that are not
nested beneath any other objects natively (natural parentId is null) and are
owned by the specified user or group.
*/
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

/*
GetPropertiesForObject retrieves the properties for a given object
*/
func GetPropertiesForObject(db *sqlx.DB, objectID []byte) ([]models.ODObjectPropertyEx, error) {
	response := []models.ODObjectPropertyEx{}
	query := `select p.* from property p inner join object_property op on p.id = op.propertyid where p.isdeleted = 0 and op.isdeleted = 0 and op.objectid = ?`
	err := db.Select(&response, query, objectID)
	if err != nil {
		print(err.Error())
	}
	return response, err
}

/*
GetRootObjectsWithProperties retrieves a list of Objects and their Properties in
Object Drive that are not nested beneath any other objects natively (natural
parentId is null)
*/
func GetRootObjectsWithProperties(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error) {
	response, err := GetRootObjects(db, orderByClause, pageNumber, pageSize)
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

/*
GetRootObjectsWithPropertiesByOwner retrieves a list of Objects and their
Properties in Object Drive that are not nested beneath any other objects
natively (natural parentId is null) and are owned by the specified user or group.
*/
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

/*
AddPropertyToObject creates a new property with the provided name and value, and
then associates that Property object to the Object indicated by ObjectID
*/
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
