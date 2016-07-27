package dao_test

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	globalconfig "decipher.com/object-drive-server/config"
	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/cmd/odrive/libs/config"
	"decipher.com/object-drive-server/cmd/odrive/libs/dao"
	"decipher.com/object-drive-server/metadata/models"
)

var db *sqlx.DB
var d *dao.DataAccessLayer
var usernames = make([]string, 10)

// NewAppConfigurationWithDefaults provides some defaults to the constructor
// function for AppConfiguration. Normally these parameters are specified
// on the command line.
func newAppConfigurationWithDefaults() config.AppConfiguration {
	var conf config.AppConfiguration
	projectRoot := filepath.Join(os.Getenv("GOPATH"), "src", "decipher.com", "object-drive-server")
	whitelist := []string{"cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"}
	opts := config.CommandLineOpts{
		Ciphers:           []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
		UseTLS:            true,
		StaticRootPath:    filepath.Join("libs", "server", "static"),
		TemplateDir:       filepath.Join("libs", "server", "static", "templates"),
		Conf:              filepath.Join(projectRoot, "cmd", "odrive", "libs", "dao", "testfixtures", "testconf.yml"),
		TLSMinimumVersion: "1.2",
	}
	conf = config.NewAppConfiguration(opts)
	conf.ServerSettings.AclImpersonationWhitelist = whitelist
	return conf
}

func init() {

	appConfiguration := newAppConfigurationWithDefaults()
	dbConfig := appConfiguration.DatabaseConnection

	// DAO tests hit a locally-running database directly.
	// This is a hack to get correct paths to certs. Depends on GOPATH.
	dbConfig.CAPath = os.ExpandEnv("$GOPATH/src/decipher.com/object-drive-server/defaultcerts/client-mysql/trust")
	dbConfig.ClientCert = os.ExpandEnv("$GOPATH/src/decipher.com/object-drive-server/defaultcerts/client-mysql/id/client-cert.pem")
	dbConfig.ClientKey = os.ExpandEnv("$GOPATH/src/decipher.com/object-drive-server/defaultcerts/client-mysql/id/client-key.pem")

	var err error
	db, err = dbConfig.GetDatabaseHandle()
	if err != nil {
		panic(err)
	}

	d = &dao.DataAccessLayer{MetadataDB: db, Logger: globalconfig.RootLogger}

	// Create users referenced by these tests
	user := models.ODUser{}
	for i := 0; i < len(usernames); i++ {
		if i == 0 {
			usernames[i] = "CN=[DAOTEST]test tester10, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		} else if i > 0 && i < 10 {
			usernames[i] = "CN=[DAOTEST]test tester0" + strconv.Itoa(i) + ", O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		} else {
			usernames[i] = "CN=[DAOTEST]test tester" + strconv.Itoa(i) + ", O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		}
		user.DistinguishedName = usernames[i]
		user.DisplayName.String = config.GetCommonName(user.DistinguishedName)
		user.DisplayName.Valid = true
		user.CreatedBy = user.DistinguishedName
		_, err = d.CreateUser(user)
	}

	user.DistinguishedName = "Bob"
	user.CreatedBy = "Bob"
	_, err = d.CreateUser(user)

}

func TestTransactionalUpdate(t *testing.T) {

	// Always skip.
	t.Skip()

	tx := db.MustBegin()

	// Add
	typeName := "New Type " + strconv.Itoa(time.Now().UTC().Nanosecond())
	addObjectTypeStatement, err := tx.Preparex(
		`insert object_type set createdBy = ?, name = ?, description = ?, contentConnector = ?`)
	if err != nil {
		log.Printf("Error %v", err)
	}
	_, err = addObjectTypeStatement.Exec("Bob", typeName, "No Decription", "No Content Connector")
	if err != nil {
		log.Printf("Error %v", err)
	}

	// Select 1st time
	dbObjectType1 := models.ODObjectType{}
	if testing.Verbose() {
		jsonData, err := json.MarshalIndent(dbObjectType1, "", "  ")
		if err != nil {
			log.Printf("Error %v", err)
		}
		jsonified := string(jsonData)
		fmt.Println(jsonified)
	}

	getObjectTypeStatement1 := `select
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,changeCount
        ,changeToken
        ,name
        ,description
        ,contentConnector    
    from object_type where name = ?`
	err = tx.Get(&dbObjectType1, getObjectTypeStatement1, typeName)
	if err != nil {
		log.Printf("Error %v", err)
	}
	if testing.Verbose() {
		log.Printf("Change Count = %d", dbObjectType1.ChangeCount)
		jsonData, err := json.MarshalIndent(dbObjectType1, "", "  ")
		if err != nil {
			log.Printf("Error %v", err)
		}
		jsonified := string(jsonData)
		fmt.Println(jsonified)
	}

	// Update (Triggers will alter the changeCount and modifiedDate and changeToken)
	newTypeName := "New Type " + strconv.Itoa(time.Now().UTC().Nanosecond())
	updateObjectTypeStatement, err := tx.Preparex(
		`update object_type set modifiedBy = ?, name = ? where name = ?`)
	if err != nil {
		log.Printf("Error %v", err)
	}
	_, err = updateObjectTypeStatement.Exec("Bob", newTypeName, typeName)
	if err != nil {
		log.Printf("Error %v", err)
	}

	// Select 2nd time
	var dbObjectType2 models.ODObjectType
	getObjectTypeStatement2 := `select
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,changeCount
        ,changeToken
        ,name
        ,description
        ,contentConnector    
    from object_type where name = ?`
	err = tx.Get(&dbObjectType2, getObjectTypeStatement2, newTypeName)
	if err != nil {
		log.Printf("Error %v", err)
	}
	if testing.Verbose() {
		log.Printf("Change Count = %d", dbObjectType2.ChangeCount)
		jsonData, err := json.MarshalIndent(dbObjectType2, "", "  ")
		if err != nil {
			log.Printf("Error %v", err)
		}
		jsonified := string(jsonData)
		fmt.Println(jsonified)
	}
	tx.Commit()
}
