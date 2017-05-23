package dao_test

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
)

const (
	SnippetDAOTP01 = "{\"f_macs\":\"{\\\"field\\\":\\\"f_macs\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"tide\\\",\\\"bir\\\",\\\"watchdog\\\"]}\",\"f_oc_org\":\"{\\\"field\\\":\\\"f_oc_org\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"dia\\\"]}\",\"f_accms\":\"{\\\"field\\\":\\\"f_accms\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[]}\",\"f_sap\":\"{\\\"field\\\":\\\"f_sar_id\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_clearance\":\"{\\\"field\\\":\\\"f_clearance\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"u\\\"]}\",\"f_regions\":\"{\\\"field\\\":\\\"f_regions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_missions\":\"{\\\"field\\\":\\\"f_missions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_share\":\"{\\\"field\\\":\\\"f_share\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"cndaotesttesttester01ou_s_governmentouchimeraoudaeoupeoplecus\\\"]}\",\"f_aea\":\"{\\\"field\\\":\\\"f_atom_energy\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_sci_ctrls\":\"{\\\"field\\\":\\\"f_sci_ctrls\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"g\\\",\\\"hcs\\\",\\\"hcs_p\\\",\\\"kdk\\\",\\\"rsv\\\",\\\"si\\\",\\\"tk\\\"]}\",\"dissem_countries\":\"{\\\"field\\\":\\\"dissem_countries\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"USA\\\"]}\"}"
	SnippetDAOTP02 = "{\"f_macs\":\"{\\\"field\\\":\\\"f_macs\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"tide\\\",\\\"bir\\\",\\\"watchdog\\\"]}\",\"f_oc_org\":\"{\\\"field\\\":\\\"f_oc_org\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"dia\\\"]}\",\"f_accms\":\"{\\\"field\\\":\\\"f_accms\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[]}\",\"f_sap\":\"{\\\"field\\\":\\\"f_sar_id\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_clearance\":\"{\\\"field\\\":\\\"f_clearance\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"u\\\"]}\",\"f_regions\":\"{\\\"field\\\":\\\"f_regions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_missions\":\"{\\\"field\\\":\\\"f_missions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_share\":\"{\\\"field\\\":\\\"f_share\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"cndaotesttesttester02ou_s_governmentouchimeraoudaeoupeoplecus\\\"]}\",\"f_aea\":\"{\\\"field\\\":\\\"f_atom_energy\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_sci_ctrls\":\"{\\\"field\\\":\\\"f_sci_ctrls\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"g\\\",\\\"hcs\\\",\\\"hcs_p\\\",\\\"kdk\\\",\\\"rsv\\\",\\\"si\\\",\\\"tk\\\"]}\",\"dissem_countries\":\"{\\\"field\\\":\\\"dissem_countries\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"USA\\\"]}\"}"
	SnippetDAOTP11 = "{\"f_macs\":\"{\\\"field\\\":\\\"f_macs\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"tide\\\",\\\"bir\\\",\\\"watchdog\\\"]}\",\"f_oc_org\":\"{\\\"field\\\":\\\"f_oc_org\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"dia\\\"]}\",\"f_accms\":\"{\\\"field\\\":\\\"f_accms\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[]}\",\"f_sap\":\"{\\\"field\\\":\\\"f_sar_id\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_clearance\":\"{\\\"field\\\":\\\"f_clearance\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"u\\\"]}\",\"f_regions\":\"{\\\"field\\\":\\\"f_regions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_missions\":\"{\\\"field\\\":\\\"f_missions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_share\":\"{\\\"field\\\":\\\"f_share\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"cndaotesttesttester1_ou_s_governmentouchimeraoudaeoupeoplecus\\\"]}\",\"f_aea\":\"{\\\"field\\\":\\\"f_atom_energy\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_sci_ctrls\":\"{\\\"field\\\":\\\"f_sci_ctrls\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"g\\\",\\\"hcs\\\",\\\"hcs_p\\\",\\\"kdk\\\",\\\"rsv\\\",\\\"si\\\",\\\"tk\\\"]}\",\"dissem_countries\":\"{\\\"field\\\":\\\"dissem_countries\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"USA\\\"]}\"}"
)

var db *sqlx.DB
var d *dao.DataAccessLayer
var usernames = make([]string, 15)
var users = make([]models.ODUser, 15)

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
		Conf:              filepath.Join(projectRoot, "dao", "testfixtures", "testconf.yml"),
		TLSMinimumVersion: "1.2",
	}
	conf = config.NewAppConfiguration(opts)
	conf.ServerSettings.AclImpersonationWhitelist = whitelist
	return conf
}

func init() {
	os.Setenv(config.OD_TOKENJAR_LOCATION, "../defaultcerts/token.jar")

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

	d = &dao.DataAccessLayer{MetadataDB: db, Logger: config.RootLogger}

	// Create users referenced by these tests
	user := models.ODUser{}
	var createdUser models.ODUser
	for i := 0; i < len(usernames); i++ {
		if i == 0 {
			usernames[i] = "CN=[DAOTEST]test tester10, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		} else if i > 0 && i < 10 {
			usernames[i] = "CN=[DAOTEST]test tester0" + strconv.Itoa(i) + ", O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		} else if i == 10 {
			usernames[i] = "CN=[DAOTEST]test tester10, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		} else if i == 11 {
			usernames[i] = "CN=[DAOTEST]test tester'1, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		} else {
			usernames[i] = "CN=[DAOTEST]test tester" + strconv.Itoa(i) + ", O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		}
		user.DistinguishedName = usernames[i]
		user.DisplayName = models.ToNullString(config.GetCommonName(user.DistinguishedName))
		user.CreatedBy = user.DistinguishedName
		createdUser, err = d.CreateUser(user)
		if err != nil {
			log.Printf("Error creating user %s %v", user.DistinguishedName, err)
		} else {
			if len(createdUser.ID) == 0 {
				log.Printf("Could not get id for user %s", user.DistinguishedName)
			}

			snippetString := strings.Replace(SnippetDAOTP01, "cndaotesttesttester01ou_s_governmentouchimeraoudaeoupeoplecus", models.AACFlatten(user.DistinguishedName), -1)
			if err := PopulateSnippetsForTestUser(&createdUser, snippetString); err != nil {
				log.Printf("Error populating snippets %v", err)
			}
		}
		users[i] = createdUser
	}

	user.DistinguishedName = "Bob"
	user.DisplayName = models.ToNullString("Bob")
	user.CreatedBy = "Bob"
	createdUser, err = d.CreateUser(user)
	if err != nil {
		log.Printf("Error creating user %s %v", user.DistinguishedName, err)
	} else {
		snippetString := strings.Replace(SnippetDAOTP01, "cndaotesttesttester01ou_s_governmentouchimeraoudaeoupeoplecus", models.AACFlatten(user.DistinguishedName), -1)
		if err := PopulateSnippetsForTestUser(&createdUser, snippetString); err != nil {
			log.Printf("Error populating snippets %v", err)
		}
	}
}

func PopulateSnippetsForTestUser(user *models.ODUser, snippetString string) error {

	useraocache, err := d.GetUserAOCacheByDistinguishedName(*user)
	if err != nil {
		return err
	}
	var ptrUserAOCache *models.ODUserAOCache
	if useraocache.ID != 0 {
		useraocache.UserID = user.ID
		useraocache.CacheDate.Time = time.Now()
		ptrUserAOCache = &useraocache
	} else {
		ptrUserAOCache = nil
	}
	snippets, err := acm.NewODriveRawSnippetFieldsFromSnippetResponse(snippetString)
	if err != nil {
		return err
	}
	user.Snippets = &snippets
	if err := d.SetUserAOCacheByDistinguishedName(ptrUserAOCache, *user); err != nil {
		return err
	}
	return nil
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
