package server_test

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"

	"encoding/hex"
	"encoding/json"

	"time"

	"bitbucket.di2e.net/dime/object-drive-server/ciphertext"
	"bitbucket.di2e.net/dime/object-drive-server/client"
	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/crypto"
	"bitbucket.di2e.net/dime/object-drive-server/dao"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

// TestDBMigration20161230 leverages the server startup that ensures docker containers for DB and server are up
// 1. Migrate down to 20161223.
// 2. Create object using DAO that has path in name. Capture generated ID.
// 3. Migrate up to latest, which is at least 20161230.
// 4. Retrieve object using API. Verify breadcrumbs reflect multiple objects
// As of 20170403 this test disabled from schema and logic changes for issue #409, schema 20170331 which
// logically expects to be able to set acmId on the object when calls to d.CreateObject are made.
// Tests can't always remain forwards compatible.
func TestDBMigration20161230(t *testing.T) {
	t.Skip()
	if testing.Short() {
		t.Skip()
	}

	// Copypasta from dao_setup_test to get a reference to database
	var db *sqlx.DB
	var d *dao.DataAccessLayer
	os.Setenv(config.OD_TOKENJAR_LOCATION, "../defaultcerts/token.jar")
	appConfiguration := newAppConfigurationWithDefaults()
	dbConfig := appConfiguration.DatabaseConnection
	dbConfig.CAPath = os.ExpandEnv("$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-mysql/trust")
	dbConfig.ClientCert = os.ExpandEnv("$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-mysql/id/client-cert.pem")
	dbConfig.ClientKey = os.ExpandEnv("$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-mysql/id/client-key.pem")
	var err error
	db, err = dbConfig.GetDatabaseHandle()
	if err != nil {
		panic(err)
	}
	d = &dao.DataAccessLayer{MetadataDB: db, Logger: config.RootLogger}
	user := models.ODUser{DistinguishedName: fakeDN0, DisplayName: models.ToNullString(config.GetCommonName(fakeDN0)), CreatedBy: fakeDN0}
	_, err = d.CreateUser(user)

	// 1. Migrate down to schema before the migration being tested
	migrateDBDownTo(t, d, "20161230")
	migrateDBDown(t, d)
	t.Logf("Schema version is %s", getSchemaVersion(t, d))

	// 2. Create object. Can use DAO handler as long as permissions not given here. Will add those separately.
	// Cannot add permission through the handler since the migration adds a resourceString column to  the
	// grantee and the DAO handler for permissions expects that to be there.
	username := fakeDN0
	var modelObj models.ODObject
	randomGUID, _ := util.NewGUID()
	defaultPathDelimiter := string(rune(30)) // 20161230 was a slash, 20170301 is now the record separator character code 30
	modelObj.Name = strings.Join([]string{randomGUID, "TestDBMigration20161230", "path", "delimiters"}, defaultPathDelimiter)
	t.Logf("Original name before migration: %s", modelObj.Name)
	modelObj.EncryptIV = crypto.CreateIV()
	modelObj.CreatedBy = username
	modelObj.TypeName.String = "File"
	modelObj.TypeName.Valid = true
	modelObj.RawAcm.String = ValidACMUnclassified
	createdObj, err := d.CreateObject(&modelObj)
	if err != nil {
		t.Logf("Error creating object: %v\n", err)
		migrateDBUp(t, d)
		t.FailNow()
	}
	// setup permissions to add to object
	ownerPermission := models.PermissionForUser(username, true, false, true, true, true)
	dp := ciphertext.FindCiphertextCacheByObject(&modelObj)
	masterKey := dp.GetMasterKey()
	models.SetEncryptKey(masterKey, &ownerPermission)
	ownerPermission.PermissionMAC = models.CalculatePermissionMAC(masterKey, &ownerPermission)
	models.CopyEncryptKey(masterKey, &ownerPermission, &ownerPermission)
	everyonePermission := models.PermissionForGroup("", "", models.EveryoneGroup, false, true, false, false, false)
	models.CopyEncryptKey(masterKey, &ownerPermission, &everyonePermission)
	// add the permissions
	addPermissionToObjectBefore20161230(t, d, createdObj, ownerPermission)
	addPermissionToObjectBefore20161230(t, d, createdObj, everyonePermission)
	t.Logf("Created name before migration: %s", createdObj.Name)

	// 3. Migrate up to latest - our recently created object will be transformed
	migrateDBUp(t, d)

	// 4. Retrieve object with API
	objectID := hex.EncodeToString(createdObj.ID)
	t.Logf("ObjectID is %s", objectID)
	tester10 := 0
	req, _ := NewGetObjectRequest(objectID, "")
	resp, _ := clients[tester10].Client.Do(req)
	data, _ := ioutil.ReadAll(resp.Body)
	var obj protocol.Object
	json.Unmarshal(data, &obj)
	t.Logf("Name after migration: %s", obj.Name)
	t.Logf("Length of breadcrumbs: %d", len(obj.Breadcrumbs))
	t.Logf("%33s %33s %33s", "Breadcrumb ID", "Breadcrumb Name", "Breadcrumb ParentID")
	for bidx, breadcrumb := range obj.Breadcrumbs {
		t.Logf("%33s %33s %33s", breadcrumb.ID, breadcrumb.Name, breadcrumb.ParentID)
		switch bidx {
		case 0:
			if breadcrumb.Name != randomGUID {
				t.Logf("First part of breadcrumb was %s expected %s", breadcrumb.Name, randomGUID)
				t.Fail()
			}
		case 1:
			if breadcrumb.Name != "TestDBMigration20161230" {
				t.Logf("Second part of breadcrumb was %s expected %s", breadcrumb.Name, "TestDBMigration20161230")
				t.Fail()
			}
		case 2:
			if breadcrumb.Name != "path" {
				t.Logf("Third part of breadcrumb was %s expected %s", breadcrumb.Name, "path")
				t.Fail()
			}
		default:
			t.Logf("Too many breadcrumb components!")
			t.Fail()
		}
	}
}

func TestDBSchemaVersionReadOnly(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Logf("Get a reference to database")
	os.Setenv(config.OD_TOKENJAR_LOCATION, os.ExpandEnv("$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/token.jar"))
	appConfiguration := newAppConfigurationWithDefaults()
	dbConfig := appConfiguration.DatabaseConnection
	dbConfig.CAPath = os.ExpandEnv("$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-mysql/trust")
	dbConfig.ClientCert = os.ExpandEnv("$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-mysql/id/client-cert.pem")
	dbConfig.ClientKey = os.ExpandEnv("$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-mysql/id/client-key.pem")
	db, err := dbConfig.GetDatabaseHandle()
	if err != nil {
		panic(err)
	}
	d := &dao.DataAccessLayer{MetadataDB: db, Logger: config.RootLogger}

	t.Logf("Set schema version to unexpected")
	nanotime := strconv.Itoa(time.Now().UTC().Nanosecond())
	newSchemaVersion := fmt.Sprintf("0.x%s-%s", nanotime, strings.Join(strings.Split(dao.SchemaVersionsSupported[0], ""), "."))
	t.Logf("New schema version will be %s", newSchemaVersion)
	setSchemaVersion(t, d, newSchemaVersion)

	t.Logf("Wait for server to detect the change")
	time.Sleep(30 * time.Second)

	t.Logf("Attempt to create an object in read only state")
	obj := client.CreateObjectRequest{
		Name:     fmt.Sprintf("TestDBSchemaVersionReadOnly %s", nanotime),
		RawAcm:   `{"classif":"U"}`,
		TypeName: "Folder"}
	_, err = clients[0].C.CreateObject(obj, nil)
	if err != nil {
		t.Logf("%s", err.Error())
	} else {
		t.Logf("Object creation was successful even though database should be in readonly state")
		t.Fail()
	}

	t.Logf("Restore schema version back as other tests depend on write access")
	setSchemaVersion(t, d, dao.SchemaVersionsSupported[0])

	t.Logf("Wait for server to detect the change")
	time.Sleep(30 * time.Second)

	t.Logf("Attempt to create an object in writeable state")
	writeableobj := client.CreateObjectRequest{Name: fmt.Sprintf("TestDBSchemaVersionWritable %s", nanotime), RawAcm: `{"classif":"U"}`, TypeName: "Folder"}
	_, err = clients[0].C.CreateObject(writeableobj, nil)
	if err != nil {
		t.Logf("%s", err.Error())
		t.Fail()
	}
}

func addGranteeIfNotExistsBefore20161230(t *testing.T, d *dao.DataAccessLayer, permission models.ODObjectPermission) {
	tx, err := d.MetadataDB.Beginx()
	if err != nil {
		t.Logf("Error beginning transaction %v", err)
		t.Fail()
		return
	}

	var response models.ODAcmGrantee
	query := `
    select 
        grantee
        ,projectName
        ,projectDisplayName
        ,groupName
        ,userDistinguishedName
        ,displayName
    from acmgrantee  
    where
        grantee = ?`
	err = tx.Unsafe().Get(&response, query, permission.Grantee)
	if err != nil {
		if err != sql.ErrNoRows {
			tx.Rollback()
			t.Logf("Error fetching grantee %v", err)
			t.Fail()
			return
		}
		// grantee not exists
		addAcmGranteeStatement, err := tx.Preparex(
			`insert acmgrantee 
			set grantee = ?, projectName = ?, projectDisplayName = ?, groupName = ?, userDistinguishedName = ?, displayName = ?`)
		addAcmGranteeStatement.Close()
		if err != nil {
			tx.Rollback()
			t.Logf("Error preparing statement for acmgrantee %v", err)
			t.Fail()
			return
		}
		_, err = addAcmGranteeStatement.Exec(permission.AcmGrantee.Grantee,
			permission.AcmGrantee.ProjectName, permission.AcmGrantee.ProjectDisplayName, permission.AcmGrantee.GroupName,
			permission.AcmGrantee.UserDistinguishedName, permission.AcmGrantee.DisplayName)
		if err != nil {
			tx.Rollback()
			t.Logf("Error executing statement for acmgrantee: %v", err)
			t.Fail()
			return
		}
		tx.Commit()
	}
	// grantee exists
	tx.Rollback()
}
func addPermissionToObjectBefore20161230(t *testing.T, d *dao.DataAccessLayer, obj models.ODObject, permission models.ODObjectPermission) {
	permission.CreatedBy = obj.CreatedBy
	tx, err := d.MetadataDB.Beginx()
	if err != nil {
		t.Logf("Error beginning transaction %v", err)
		t.Fail()
		return
	}
	addGranteeIfNotExistsBefore20161230(t, d, permission)
	// Setup the statement
	addPermissionStatement, err := tx.Preparex(`insert object_permission set 
        createdby = ?
        ,objectId = ?
        ,grantee = ?
        ,acmShare = ?
        ,allowCreate = ?
        ,allowRead = ?
        ,allowUpdate = ?
        ,allowDelete = ?
        ,allowShare = ?
        ,explicitShare = ?
        ,encryptKey = ?
		,permissionIV = ?
		,permissionMAC = ?
    `)
	if err != nil {
		tx.Rollback()
		t.Logf("Error preparing statement for permission %v", err)
		t.Fail()
		return
	}
	// Add it
	_, err = addPermissionStatement.Exec(permission.CreatedBy, obj.ID,
		permission.Grantee, permission.AcmShare, permission.AllowCreate,
		permission.AllowRead, permission.AllowUpdate, permission.AllowDelete,
		permission.AllowShare, permission.ExplicitShare, permission.EncryptKey,
		permission.PermissionIV, permission.PermissionMAC)
	if err != nil {
		tx.Rollback()
		t.Logf("Error executing statement for permission: %v", err)
		t.Fail()
		return
	}
	addPermissionStatement.Close()
	tx.Commit()
}
func setSchemaVersion(t *testing.T, d *dao.DataAccessLayer, version string) {
	tx, err := d.MetadataDB.Beginx()
	if err != nil {
		t.Logf("Error beginning transaction %v", err)
		t.Fail()
		return
	}
	// Setup the statement
	stmt, err := tx.Preparex(`update dbstate set schemaversion = ?`)
	if err != nil {
		tx.Rollback()
		t.Logf("Error preparing statement for dbstate %v", err)
		t.Fail()
		return
	}
	// Set it
	_, err = stmt.Exec(version)
	if err != nil {
		tx.Rollback()
		t.Logf("Error executing statement for dbstate: %v", err)
		t.Fail()
		return
	}
	stmt.Close()
	tx.Commit()
}

func newAppConfigurationWithDefaults() config.AppConfiguration {
	var conf config.AppConfiguration
	projectRoot := filepath.Join(os.Getenv("GOPATH"), "src", "bitbucket.di2e.net", "dime", "object-drive-server")
	whitelist := []string{"cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"}
	opts := config.ValueOpts{
		Ciphers:           []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
		Conf:              filepath.Join(projectRoot, "dao", "testfixtures", "testconf.yml"),
		TLSMinimumVersion: "1.2",
	}
	conf = config.NewAppConfiguration(opts)
	conf.ServerSettings.ACLImpersonationWhitelist = whitelist
	return conf
}

func getLastLine(i string) string {
	lines := strings.Split(i, "\n")
	lastline := strings.TrimSpace(lines[len(lines)-1])
	if len(lastline) == 0 {
		lastline = strings.TrimSpace(lines[len(lines)-2])
	}
	return lastline
}
func getSchemaVersion(t *testing.T, d *dao.DataAccessLayer) string {
	dbstate, err := d.GetDBState()
	if err != nil {
		t.Logf("Unable to get DBState %v", err)
		t.FailNow()
		return ""
	}
	return dbstate.SchemaVersion
}
func isCircleCI() bool {
	return len(os.Getenv("CIRCLE_BUILD_NUM")) > 0
}
func makeMigrateCommand(direction string) *exec.Cmd {
	var args []string
	useLXC := false
	if isCircleCI() {
		if useLXC {
			args = append(args, "lxc-attach", "-n", "$(docker inspect --format \"{{.Id}}\" docker_metadatadb_1)", "--", "bash", "-c", "/usr/local/bin/odrive-database", "migrate", direction, "--useEmbedded=true")
			return exec.Command("sudo", args...)
		} else {
			args = append(args, "migrate", direction, "--useEmbedded=true", "--conf", "/home/ubuntu/.go_workspace/src/bitbucket.di2e.net/dime/object-drive-server/cmd/odrive-database/db.yml")
			return exec.Command("/home/ubuntu/.go_workspace/src/bitbucket.di2e.net/dime/object-drive-server/cmd/odrive-database/odrive-database")
		}
	} else {
		args = append(args, "exec", "-t", "docker_metadatadb_1", "/usr/local/bin/odrive-database", "migrate", direction, "--useEmbedded=true")
		return exec.Command("docker", args...)
	}
}
func migrateDBDownTo(t *testing.T, d *dao.DataAccessLayer, targetVersion string) {
	beforeMigrate := getSchemaVersion(t, d)
	if beforeMigrate == targetVersion {
		t.Logf("Schema version is %s", beforeMigrate)
		return
	}
	minimumPermittedVersion := "20160824"
	afterMigrate := ""
	schemaVersion := beforeMigrate
	var cmdOut []byte
	var err error
	for schemaVersion != targetVersion && schemaVersion != "" && schemaVersion != afterMigrate && schemaVersion != minimumPermittedVersion {
		afterMigrate = schemaVersion
		t.Logf("Current version is %s, migrating down", schemaVersion)
		schemaVersion = ""
		if cmdOut, err = makeMigrateCommand("down").Output(); err != nil {
			t.Logf("%s: %v", "There was an error migrating down", err)
			t.FailNow()
		}
		stringified := string(cmdOut)
		t.Logf(stringified)
		schemaVersion = getSchemaVersion(t, d)
	}
	afterMigrate = schemaVersion
	t.Logf("Migrated (down) from %s to %s", beforeMigrate, afterMigrate)
}
func migrateDBDown(t *testing.T, d *dao.DataAccessLayer) {
	beforeMigrate := getSchemaVersion(t, d)
	var cmdOut []byte
	var err error
	if cmdOut, err = makeMigrateCommand("down").Output(); err != nil {
		t.Logf("%s: %v", "There was an error migrating down", err)
		t.FailNow()
	}
	stringified := string(cmdOut)
	lastline := getLastLine(stringified)
	t.Logf("%s", lastline)
	afterMigrate := getSchemaVersion(t, d)
	t.Logf("Migrated (down) from %s to %s", beforeMigrate, afterMigrate)
}
func migrateDBUp(t *testing.T, d *dao.DataAccessLayer) {
	beforeMigrate := getSchemaVersion(t, d)
	var cmdOut []byte
	var err error
	if cmdOut, err = makeMigrateCommand("up").Output(); err != nil {
		t.Logf("%s: %v", "There was an error migrating up", err)
		t.FailNow()
	}
	stringified := string(cmdOut)
	lastline := getLastLine(stringified)
	t.Logf("%s", lastline)
	afterMigrate := getSchemaVersion(t, d)
	t.Logf("Migrated (up) from %s to %s", beforeMigrate, afterMigrate)
}
