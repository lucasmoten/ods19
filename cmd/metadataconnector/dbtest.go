package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"decipher.com/oduploader/cmd/metadataconnector/libs"
	"decipher.com/oduploader/metadata/models"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func main() {

	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection
	dbTLS := dbConfig.GetTLSConfig()
	mysql.RegisterTLSConfig("custom", &dbTLS)

	// ==========================================================================================================

	// Get handle to database (not yet a connection)
	db, err := sqlx.Open("mysql", dbConfig.GetDSN())
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	// Validate DSN
	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	// Retrieve some objects
	objects := []models.ODObject{}
	err = db.Select(&objects, "SELECT * FROM object limit 100")
	if err != nil {
		panic(err.Error())
	}
	// ==========================================================================================================

	// Load properties for each in the result set
	for e := 0; e < len(objects); e++ {
		err = db.Select(&objects[e].Properties, "SELECT p.* FROM property p INNER JOIN object_property op ON p.id = op.propertyId WHERE op.objectId = ?", objects[e].ID)
	}

	// Convert the objects to JSON and display
	jsonData, err := json.Marshal(objects)
	jsonified := string(jsonData)
	fmt.Println(jsonified)
	fmt.Println("-----------------------")

	// For 5 objects, show reading individual attributes name and modification date
	for i := 50; i < 55; i++ {
		fmt.Println(objects[i].Name, "was created on", objects[i].ModifiedDate)
	}
	fmt.Println("-----------------------")

	// Size of the array of objects
	fmt.Println("There are", len(objects), "rows in the resultset")
	fmt.Println("-----------------------")

	// JSON of a single object, with whitespace indentation instead of flow wrapping
	objectIndex := 42
	//`SELECT p.* FROM property p INNER JOIN object_property op ON p.id = op.propertyId WHERE op.objectId = ?`
	//objectProperties := []ODObjectPropertyEx{}
	err = db.Select(&objects[objectIndex].Properties, "SELECT p.* FROM property p INNER JOIN object_property op ON p.id = op.propertyId WHERE op.objectId = ?", objects[objectIndex].ID)
	//objects[objectIndex].Properties = objectProperties
	fmt.Println("Object ", objectIndex, "JSONIFIED...")
	j, err := json.MarshalIndent(objects[objectIndex], "", "  ")
	fmt.Println(string(j))

	// ==========================================================================================================
	// Below is an example of adding a new property and gettings its id and then associating to the object.

	newPropCreatedBy := objects[objectIndex].CreatedBy
	newPropValue := time.Now()
	newPropName := "Prop" + strconv.Itoa(newPropValue.Nanosecond()) // hex.EncodeToString(localhasher.Sum(nil))
	newPropClass := "U"
	// Add a property to that object...
	addPropertyStmt, err := db.Prepare(`INSERT property SET createdBy = ?, name = ?, propertyValue = ?, classificationPM = ?`)
	if err != nil {
		panic(err.Error())
	}
	res, err := addPropertyStmt.Exec(newPropCreatedBy, newPropName, newPropValue, newPropClass)
	if err != nil {
		log.Fatal(err)
	}
	rowCnt, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Rows affected:", rowCnt)

	// Get the ID for the new property
	var newPropertyID []byte
	getPropertyIDStmt, err := db.Prepare(`SELECT id FROM property WHERE createdBy = ? AND name = ? and propertyValue = ? and classificationPM = ? ORDER BY createdDate DESC LIMIT 1`)
	if err != nil {
		panic(err.Error())
	}
	err = getPropertyIDStmt.QueryRow(newPropCreatedBy, newPropName, newPropValue, newPropClass).Scan(&newPropertyID)
	if err != nil {
		log.Fatal(err)
	}

	// Add association to the object
	addObjectPropertyStmt, err := db.Prepare(`INSERT object_property SET createdBy = ?, objectId = ?, propertyId = ?`)
	res, err = addObjectPropertyStmt.Exec(newPropCreatedBy, objects[objectIndex].ID, newPropertyID)
	if err != nil {
		log.Fatal(err)
	}

}
