package client

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/protocol"
	"github.com/stretchr/testify/assert"
)

// testDir defines the location for files used in upload/download tests.
var testDir string

// conf contains configuration necessary for the client to connect to a running odrive instance.
var conf = Config{
	Cert:       os.Getenv("GOPATH") + "/src/decipher.com/object-drive-server/defaultcerts/clients/test_0.cert.pem",
	Trust:      os.Getenv("GOPATH") + "/src/decipher.com/object-drive-server/defaultcerts/clients/client.trust.pem",
	Key:        os.Getenv("GOPATH") + "/src/decipher.com/object-drive-server/defaultcerts/clients/test_0.key.pem",
	SkipVerify: true,
	Remote:     fmt.Sprintf("https://%s:%s/services/object-drive/1.0", config.DockerVM, config.Port),
}

var permissions = protocol.Permission{
	Read: protocol.PermissionCapability{
		AllowedResources: []string{"user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"},
	}}

// TestMain setups up the necessary files for the test-suite.
func TestMain(m *testing.M) {
	testDir, _ = ioutil.TempDir("", "testData")

	testFile, err := ioutil.TempFile(testDir, "particle")
	if err != nil {
		fmt.Printf("Error creating test file %s", testFile)
	}

	code := m.Run()

	os.RemoveAll(testDir)

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

	t.Log("Reading from temporary directory for files to upload: ", testDir)

	files, err := ioutil.ReadDir(testDir)
	if err != nil {
		t.Log("Can't read anything from the test directory")
		log.Fatal(err)
	}

	// Run tests for all files in the fixtures folder
	for _, file := range files {
		// Upload local test fixtures
		fullFilePath := path.Join(testDir, file.Name())
		t.Log(fullFilePath)

		var upObj = protocol.CreateObjectRequest{
			TypeName:              "File",
			Name:                  file.Name(),
			NamePathDelimiter:     fmt.Sprintf("%s", os.PathSeparator),
			Description:           "A test Particle ",
			ParentID:              "",
			RawAcm:                `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`,
			ContainsUSPersonsData: "Unknown",
			ExemptFromFOIA:        "",
			Permission:            permissions,
			OwnedBy:               "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
		}

		fReader, err := os.Open(fullFilePath)
		if err != nil {
			t.Log(err)
		}
		newObj, err := me.CreateObject(upObj, fReader)
		assert.Nil(t, err, fmt.Sprintf("Creating object hit an error: %s", err))

		t.Log("Uploaded object has ID: ", newObj.ID)

		// Pull the fixtures back down
		reader, err := me.GetObjectStream(newObj.ID)
		assert.Nil(t, err, fmt.Sprintf("Retrieving stream hit an error: %s", err))

		os.MkdirAll(path.Join(testDir, "retrieved"), os.FileMode(int(0700)))
		outName := path.Join(testDir, "retrieved", newObj.Name)

		t.Log("Preparing to pull down file to: ", outName)
		t.Log("ChangeToken: ", newObj.ChangeToken)
		err = writeObjectToDisk(outName, reader)
		assert.Nil(t, err, fmt.Sprintf("Writing encountered an error: %s", err))

		// Delete the fixture
		t.Log("Deleting object")
		delResponse, err := me.DeleteObject(newObj.ID, newObj.ChangeToken)
		assert.Nil(t, err, "Error on deleting object %s", err)
		t.Log("Response from delete: ", delResponse)

	}
}

// TestCreteObjectNoSTream tests the creation of an object with no stream, just metadata,
// such as a folder.
func TestCreateObjectNoStream(t *testing.T) {
	me, err := NewClient(conf)

	var upObj = protocol.CreateObjectRequest{
		TypeName:              "Folder",
		Name:                  "TestDir",
		NamePathDelimiter:     fmt.Sprintf("%s", os.PathSeparator),
		Description:           "A test Particle ",
		ParentID:              "",
		RawAcm:                `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`,
		ContainsUSPersonsData: "Unknown",
		ExemptFromFOIA:        "",
		Permission:            permissions,
		OwnedBy:               "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
	}

	retObj, err := me.CreateObject(upObj, nil)
	t.Log("Object created is: ", retObj.ID)
	assert.Nil(t, err, "Error creating object with no stream %s", err)

}

// writeObjectToDisk retrieves an object and writes it to the filesystem.
func writeObjectToDisk(name string, reader io.Reader) error {
	file, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(name, file, os.FileMode(int(0700)))
	if err != nil {
		return err
	}

	return nil
}