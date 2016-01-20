package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/cmd/metadataconnector/libs/server"
)

func main() {

	// Load Configuration from conf.json
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection
	serverConfig := appConfiguration.ServerSettings

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

	// Setup web server
	s, err := makeServer(serverConfig, db)
	// with TLS support
	stls := serverConfig.GetTLSConfig()
	s.TLSConfig = &stls
	serverCertFile := serverConfig.ServerCertChain
	serverKeyFile := serverConfig.ServerKey
	// start it
	log.Println("Starting server on " + s.Addr)
	log.Fatalln(s.ListenAndServeTLS(serverCertFile, serverKeyFile))

	//dbtest()
}

func makeServer(serverConfig config.ServerSettingsConfiguration, db *sqlx.DB) (*http.Server, error) {
	httpHandler := server.AppServer{
		Port:       serverConfig.ListenPort,
		Bind:       serverConfig.ListenBind,
		Addr:       serverConfig.ListenBind + ":" + strconv.Itoa(serverConfig.ListenPort),
		MetadataDB: db,
	}
	return &http.Server{
		Addr:           string(httpHandler.Addr),
		Handler:        httpHandler,
		ReadTimeout:    10000 * time.Second, //This breaks big downloads
		WriteTimeout:   10000 * time.Second,
		MaxHeaderBytes: 1 << 20, //This prevents clients from DOS'ing us
	}, nil
}

// This function will be removed as these sample calls are transitioned into corresponding handlers
func dbtest() {

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
