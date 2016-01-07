package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
)

func main() {

	// Load Configuration from conf.json
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection

	// Setup handle to the database
	db, err := dbConfig.GetDatabaseHandle()
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	// Validate the DSN for the database by pinging it
	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	// ===========================================================================
	// Retrieve Alice's root objects
	response, err := dao.GetRootObjectsWithPropertiesByOwner(db,
		"createdDate DESC", 1, 20, "Alice")
	objects := response.Objects
	if err != nil {
		panic(err.Error())
	}
	jsonData, err := json.MarshalIndent(response, "", "  ")
	jsonified := string(jsonData)
	fmt.Println(jsonified)
	// ===========================================================================
	// Choose a random object in the resultset
	rns := rand.NewSource(int64(time.Now().Nanosecond()))
	objectIndex := rand.New(rns).Intn(len(objects))
	// ===========================================================================
	// Add a new property to the chosen object
	fmt.Println("Adding property to " + strconv.Itoa(objectIndex))
	if len(objects) > objectIndex {
		newPropertyCreatedBy := objects[objectIndex].CreatedBy
		newPropertyName := "Prop" + strconv.Itoa(time.Now().Nanosecond())
		newPropertyValue := time.Now().Format(time.RFC3339)
		newPropertyClassification := "U"

		dao.AddPropertyToObject(db, newPropertyCreatedBy, objects[objectIndex].ID,
			newPropertyName, newPropertyValue, newPropertyClassification)
	}
	// ===========================================================================
	// Retrieve Alice's root objects
	response, err = dao.GetRootObjectsWithPropertiesByOwner(db,
		"createdDate DESC", 1, 20, "Alice")
	objects = response.Objects
	if err != nil {
		panic(err.Error())
	}
	jsonData, err = json.MarshalIndent(response, "", "  ")
	jsonified = string(jsonData)
	fmt.Println(jsonified)

}
