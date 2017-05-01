package client

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"decipher.com/object-drive-server/protocol"
	"github.com/stretchr/testify/assert"
)

// conf contains configuration necessary for the client to connect to a running odrive instance.
var conf = Config{
	Cert:   os.Getenv("GOPATH") + "/src/decipher.com/object-drive-server/defaultcerts/clients/test_0.cert.pem",
	Trust:  os.Getenv("GOPATH") + "/src/decipher.com/object-drive-server/defaultcerts/clients/client.trust.pem",
	Key:    os.Getenv("GOPATH") + "/src/decipher.com/object-drive-server/defaultcerts/clients/test_0.key.pem",
	Remote: "https://dockervm:8080/services/object-drive/1.0",
}

// TestMain setups up the necessary files for the test-suite.
func TestMain(m *testing.M) {

	message := []byte("Testing....testing...is this thing on?")
	os.Mkdir("fixtures", os.FileMode(int(0700)))
	err := ioutil.WriteFile("fixtures/testParticle1.txt", message, os.FileMode(int(0700)))
	if err != nil {
		log.Println(err)
	}

	code := m.Run()

	os.Remove("fixtures/testParticle1.txt")

	os.Exit(code)
}

// TestNewClient simple starts up a new client with using included certs and a default
// Object-drive instance.  The drive must be up and running for this to succeed.
func TestNewClient(t *testing.T) {
	_, err := NewClient(conf)
	assert.Nil(t, err, fmt.Sprintf("ERROR creating new client: %s", err))
}

// TestRoundTrip tests the upload/download mechanisms by iterating through every file in the
// fixtures directory and performing a sequence of upload and download to verify the operations
// complete successfully.
func TestRoundTrip(t *testing.T) {
	me, err := NewClient(conf)

	var dataDir = "./fixtures"
	files, err := ioutil.ReadDir(dataDir)
	if err != nil {
		log.Fatal(err)
	}

	// Run tests for all files in the fixtures folder
	for _, file := range files {
		// Upload local test fixtures
		fullFilePath := path.Join(dataDir, file.Name())
		log.Println(fullFilePath)

		var permissions = protocol.Permission{
			Read: protocol.PermissionCapability{
				AllowedResources: []string{"user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"},
			}}

		var upObj = protocol.CreateObjectRequest{
			TypeName:              "File",
			Name:                  fullFilePath,
			NamePathDelimiter:     fmt.Sprintf("%s", os.PathSeparator),
			Description:           "A test Particle ",
			ParentID:              "",
			RawAcm:                `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`,
			ContainsUSPersonsData: "Unknown",
			ExemptFromFOIA:        "",
			Permission:            permissions,
			OwnedBy:               "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
		}

		newObj, err := me.CreateObject(upObj, nil)
		assert.Nil(t, err, fmt.Sprintf("Creating object hit an error: %s", err))

		log.Printf("Uploaded object has ID: %s", newObj.ID)

		// Pull the fixtures back down
		reader, err := me.GetObjectStream(newObj.ID)
		assert.Nil(t, err, fmt.Sprintf("Retrieving stream hit an error: %s", err))

		outName := path.Join("./retrieved", newObj.Name)
		log.Println("ChangeToken: ", newObj.ChangeToken)
		err = WriteObject(outName, reader)
		assert.Nil(t, err, fmt.Sprintf("Writing encountered an error: %s", err))

		// Delete the fixture
		log.Printf("Deleting object")
		delResponse, err := me.DeleteObject(newObj.ID, newObj.ChangeToken)
		assert.Nil(t, err, "Error on deleting object %s", err)
		log.Printf("Response from delete: %v", delResponse)

	}
}
