package client_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"strings"

	"bitbucket.di2e.net/dime/object-drive-server/client"
	"bitbucket.di2e.net/dime/object-drive-server/events"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

const ValidACMUnclassifiedFOUOSharedToTester10 = `{"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"f_clearance":["u"],"f_share":["cntesttester01oupeopleoudaeouchimeraou_s_governmentcus"],"portion":"U//FOUO","share":{"users":["cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`

// This is duplicated from server_test, so that these variables cannot
// accidentally be used in the running server.  We can't do imports
// of foreign test packages.

var mountPoint = util.GetClientMountPoint()

// testDir defines the location for files used in upload/download tests.
var testDir string

// conf contains configuration necessary for the client to connect to a running odrive instance.
var conf = client.Config{
	Cert:       os.Getenv("GOPATH") + "/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/clients/test_0.cert.pem",
	Trust:      os.Getenv("GOPATH") + "/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/server/server.trust.pem",
	Key:        os.Getenv("GOPATH") + "/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/clients/test_0.key.pem",
	SkipVerify: false,
	ServerName: util.GetEnvWithDefault("OD_PEER_CN", "twl-server-generic2"), // If you set OD_PEER_CN, then this matches it
	Remote:     mountPoint,
}

var permissions = client.Permission{
	Read: client.PermissionCapability{
		AllowedResources: []string{"user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"},
	}}

// TestMain setups up the necessary files for the test-suite.
func TestMain(m *testing.M) {
	testDir, _ = ioutil.TempDir("", "testData")
	testFile, err := ioutil.TempFile(testDir, "particle")
	if err != nil {
		log.Println("error creating temp file:", testFile, err)
	}
	if code := stallForAvailability(); code != 0 {
		os.Exit(code)
	}
	code := m.Run()

	os.RemoveAll(testDir)

	os.Exit(code)
}

// TestNewClient simple starts up a new client with using included certs and a default
// Object-drive instance.  The drive must be up and running for this to succeed.
func TestNewClient(t *testing.T) {
	_, err := client.NewClient(conf)
	require.Nil(t, err, fmt.Sprintf("ERROR creating new client: %s", err))
}

// TestCreteObjectNoStream tests the creation of an object with no stream, just metadata,
// such as a folder.
func TestCreateObjectNoStream(t *testing.T) {
	me, err := client.NewClient(conf)
	if err != nil {
		t.Fatalf("could not create client: %v", err)
	}

	var upObj = client.CreateObjectRequest{
		TypeName:              "Folder",
		Name:                  "TestDir",
		NamePathDelimiter:     fmt.Sprintf("%v", os.PathSeparator),
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
	require.Nil(t, err, "Error creating object with no stream %s", err)

}

func TestImpersonation(t *testing.T) {
	t.Log("create a new config with impersonation")
	cnf := conf
	cnf.Cert = os.Getenv("GOPATH") + "/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/server/server.cert.pem"
	cnf.Trust = os.Getenv("GOPATH") + "/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/server/server.trust.pem"
	cnf.Key = os.Getenv("GOPATH") + "/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/server/server.key.pem"
	cnf.Impersonation = "cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	c, err := client.NewClient(cnf)
	if err != nil {
		t.Fatalf("could not create client with impersonation: %v", err)
	}
	c.Verbose = testing.Verbose()

	t.Logf("MyDN: %s", c.MyDN)
	cor := client.CreateObjectRequest{
		Name:   "impersonados",
		RawAcm: ValidACMUnclassifiedFOUOSharedToTester10,
	}
	obj, err := c.CreateObject(cor, nil)
	if err != nil {
		t.Errorf("create object with impersonation did not succeed: %v", err)
		t.FailNow()
	}
	if !strings.HasPrefix(obj.OwnedBy, "user/cn=test tester01") {
		t.Errorf("expected tester01 to be the owner, since tester01 was impersonated")
	}
}

func TestListRootObjects(t *testing.T) {
	c, err := client.NewClient(conf)
	if err != nil {
		t.Fatalf("could not create client: %v", err)
	}
	c.Verbose = testing.Verbose()
	pr := client.PagingRequest{}
	_, err = c.Search(pr, false)
	if err != nil {
		t.Errorf("search error listing root objects: %v", err)
	}
}

func TestMoveObject(t *testing.T) {
	me, err := client.NewClient(conf)
	if err != nil {
		t.Fatalf("could not create client: %v", err)
	}

	// Create file at root.
	testFile, err := ioutil.TempFile(testDir, "particle")
	if err != nil {
		fmt.Printf("error creating test file")
	}

	var fileReq = client.CreateObjectRequest{
		TypeName:              "File",
		Name:                  "ToMoveOrNotToMove",
		NamePathDelimiter:     fmt.Sprintf("%v", os.PathSeparator),
		Description:           "This had better move to NOT the root folder.",
		ParentID:              "",
		RawAcm:                `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`,
		ContainsUSPersonsData: "Unknown",
		ExemptFromFOIA:        "",
		Permission:            permissions,
		OwnedBy:               "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
	}

	fileObj, err := me.CreateObject(fileReq, testFile)
	t.Log("File created is: ", fileObj.ID)
	require.Nil(t, err, "error creating file to move", err)

	// Now create the folder in which to move it.
	var dirReq = client.CreateObjectRequest{
		TypeName:              "Folder",
		Name:                  "MovedTo",
		NamePathDelimiter:     fmt.Sprintf("%v", os.PathSeparator),
		Description:           "Give me some files!",
		ParentID:              "",
		RawAcm:                `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`,
		ContainsUSPersonsData: "Unknown",
		ExemptFromFOIA:        "",
		Permission:            permissions,
		OwnedBy:               "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
	}

	dirObj, err := me.CreateObject(dirReq, nil)
	t.Log("Folder created is: ", dirObj.ID)
	require.Nil(t, err, "Error creating object with no stream %s", err)

	// Mow perform the move
	var moveReq = client.MoveObjectRequest{
		ID:          fileObj.ID,
		ChangeToken: fileObj.ChangeToken,
		ParentID:    dirObj.ID,
	}

	moved, err := me.MoveObject(moveReq)
	t.Log("Moved object to", dirObj.Name, " with ID ", moved.ParentID)
	require.Nil(t, err, "error moving object %s", err)
}

func TestResponder(t *testing.T) {
	t.Skip()
	// Connect to kafka
	c, err := client.NewOdriveResponder(
		conf,
		"odrive_to_text",
		os.Getenv("OD_EVENT_ZK_ADDRS"),
		fetcher,
	)
	if err != nil {
		log.Printf("error creating: %v", err)
		t.FailNow()
	}
	c.Timeout = 1 * time.Second
	c.DebugMode = true
	c.Note("connect to kafka")
	for {
		err = c.ConsumeKafka()
		if err != nil {
			log.Printf("error connecting: %v", err)
			t.FailNow()
		}
	}
}

func TestRestoreVersion(t *testing.T) {
	c, err := client.NewClient(conf)
	if err != nil {
		t.Fatalf("could not create client: %v", err)
	}
	c.Verbose = testing.Verbose()
	cor := client.CreateObjectRequest{
		Name:       "restoreversion1",
		RawAcm:     `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"]}`,
		OwnedBy:    "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
		Properties: []client.Property{client.Property{Name: "p1", Value: "original"}},
	}
	obj, err := c.CreateObject(cor, nil)
	if err != nil {
		t.Errorf("create object error: %v", err)
	}
	var uor1 = client.UpdateObjectRequest{
		Name:        "change1",
		ID:          obj.ID,
		ChangeToken: obj.ChangeToken,
		Properties:  []client.Property{client.Property{Name: "p1", Value: "v1"}, client.Property{Name: "p2", Value: "v2"}},
	}
	uo1, err := c.UpdateObject(uor1)
	if err != nil {
		t.Errorf("error updating object: %v", err)
	}
	var uor2 = client.UpdateObjectRequest{
		Name:        "change2",
		ID:          obj.ID,
		ChangeToken: uo1.ChangeToken,
		Properties:  []client.Property{client.Property{Name: "p1", Value: "changed"}, client.Property{Name: "p2", Value: ""}, client.Property{Name: "p3", Value: "new"}},
	}
	uo2, err := c.UpdateObject(uor2)
	if err != nil {
		t.Errorf("error updating object: %v", err)
	}

	// restore version 2 (changecount=1)
	uo3, err := c.RestoreRevision(obj.ID, uo2.ChangeToken, uo1.ChangeCount)
	if err != nil {
		t.Errorf("error restoring object: %v", err)
	}
	if uo3.Name != uo1.Name {
		t.Errorf("restored version name was %s, expected %s", uo3.Name, uo1.Name)
	}
	if len(uo3.Properties) != len(uo1.Properties) {
		t.Errorf("restored version had %d properties, expected %d", len(uo3.Properties), len(uo1.Properties))
	}

	// restore version 3 (changecount=2)
	uo4, err := c.RestoreRevision(obj.ID, uo3.ChangeToken, uo2.ChangeCount)
	if err != nil {
		t.Errorf("error restoring object: %v", err)
	}
	if uo4.Name != uo2.Name {
		t.Errorf("restored version name was %s, expected %s", uo4.Name, uo2.Name)
	}
	if len(uo4.Properties) != len(uo2.Properties) {
		t.Errorf("restored version had %d properties, expected %d", len(uo4.Properties), len(uo2.Properties))
	}

	// restore original version (changecount=0)
	uo5, err := c.RestoreRevision(obj.ID, uo4.ChangeToken, obj.ChangeCount)
	if err != nil {
		t.Errorf("error restoring object: %v", err)
	}
	if uo5.Name != obj.Name {
		t.Errorf("restored version name was %s, expected %s", uo5.Name, obj.Name)
	}
	if len(uo5.Properties) > len(obj.Properties) {
		t.Errorf("restored version had %d properties, expected %d", len(uo5.Properties), len(obj.Properties))
	}
	if uo5.ChangeCount != 5 {
		t.Errorf("change count of restored object was %d", uo5.ChangeCount)
	}
	for _, p := range uo5.Properties {
		if p.Name == "p1" && p.Value != "original" {
			t.Errorf("Expected property value for p1 to be 'original', got %s", p.Value)
		}
	}

	t.Log(uo4)
}

// TestRoundTrip tests the upload/download mechanisms by iterating through every file in the
// fixtures directory and performing a sequence of upload and download to verify the operations
// complete successfully.
func TestRoundTrip(t *testing.T) {
	me, err := client.NewClient(conf)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Reading from temporary directory for files to upload: ", testDir)

	files, err := ioutil.ReadDir(testDir)
	if err != nil {
		t.Fatal(err)
	}

	// Run tests for all files in the fixtures folder
	for _, file := range files {
		// Upload local test fixtures
		fullFilePath := path.Join(testDir, file.Name())
		t.Log(fullFilePath)

		var upObj = client.CreateObjectRequest{
			TypeName:              "File",
			Name:                  file.Name(),
			NamePathDelimiter:     fmt.Sprintf("%v", os.PathSeparator),
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
		require.Nil(t, err, fmt.Sprintf("Creating object hit an error: %s", err))

		t.Log("Uploaded object has ID: ", newObj.ID)

		// Pull the fixtures back down
		reader, err := me.GetObjectStream(newObj.ID)
		require.Nil(t, err, fmt.Sprintf("Retrieving stream hit an error: %s", err))

		os.MkdirAll(path.Join(testDir, "retrieved"), os.FileMode(int(0700)))
		outName := path.Join(testDir, "retrieved", newObj.Name)

		t.Log("Preparing to pull down file to: ", outName)
		t.Log("ChangeToken: ", newObj.ChangeToken)
		err = writeObjectToDisk(outName, reader)
		require.Nil(t, err, fmt.Sprintf("Writing encountered an error: %s", err))

		// Delete the fixture
		t.Log("Deleting object")
		delResponse, err := me.DeleteObject(newObj.ID, newObj.ChangeToken)
		require.Nil(t, err, "Error on deleting object %s", err)
		t.Log("Response from delete: ", delResponse)

	}
}

func TestSearchObjects(t *testing.T) {
	c, err := client.NewClient(conf)
	if err != nil {
		t.Fatalf("could not create client: %v", err)
	}
	c.Verbose = testing.Verbose()

	// Leverage another test to create a file named `ToMoveOrNotToMove` and move it into a folder `MovedTo`
	TestMoveObject(t)

	pr := client.PagingRequest{}
	pr.FilterSettings = append(pr.FilterSettings, client.FilterSetting{FilterField: "name", Condition: "equals", Expression: "ToMoveOrNotToMove"})
	ors, err := c.Search(pr, true)
	if err != nil {
		t.Errorf("search error listing root objects: %v", err)
	}
	if ors.TotalRows == 0 {
		t.Errorf("expected at least one item to be found in search")
	}
}

func TestUpdateObject(t *testing.T) {
	c, err := client.NewClient(conf)
	if err != nil {
		t.Fatalf("could not create client: %v", err)
	}
	c.Verbose = testing.Verbose()
	cor := client.CreateObjectRequest{
		RawAcm:  `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`,
		OwnedBy: "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
	}
	obj, err := c.CreateObject(cor, nil)
	if err != nil {
		t.Errorf("create object error: %v", err)
	}
	var uor = client.UpdateObjectRequest{
		Name:        "espresso",
		ID:          obj.ID,
		ChangeToken: obj.ChangeToken,
	}
	uo, err := c.UpdateObject(uor)
	if err != nil {
		t.Errorf("error updating object: %v", err)
	}
	t.Log(uo)

}

func TestUpdateObjectAndStream(t *testing.T) {
	c, err := client.NewClient(conf)
	if err != nil {
		t.Fatalf("could not create client: %v", err)
	}
	c.Verbose = testing.Verbose()
	cor := client.CreateObjectRequest{
		Name:    "Mets",
		RawAcm:  `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`,
		OwnedBy: "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
	}
	obj, err := c.CreateObject(cor, nil)
	if err != nil {
		t.Errorf("create object error: %v", err)
	}
	var uoasr = client.UpdateObjectAndStreamRequest{
		Name:        "Astros",
		ID:          obj.ID,
		ChangeToken: obj.ChangeToken,
	}
	buf := bytes.NewBuffer([]byte("Altuve"))
	uo, err := c.UpdateObjectAndStream(uoasr, buf)
	if err != nil {
		t.Errorf("error updating object: %v", err)
	}
	if uo.Name != "Astros" {
		t.Errorf("expected Astros but got %s", uo.Name)
	}

}

// writeObjectToDisk retrieves an object and writes it to the filesystem.
func writeObjectToDisk(name string, reader io.Reader) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(name, data, os.FileMode(int(0700)))
	if err != nil {
		return err
	}

	return nil
}

func stallForAvailability() int {
	c, err := client.NewClient(conf)
	if err != nil {
		log.Printf("could not create client: %v", err)
		return -9
	}

	// Do this on every try to check the server
	retryFunc := func() int {
		res, err := c.Ping()
		if err != nil {
			log.Printf("bad request: %v", err)
			return -11
		}
		if !res {
			log.Printf("odrive not ready to serve")
			return -10
		}
		return 0
	}

	// Try every few seconds
	tck := time.NewTicker(1 * time.Second)
	defer tck.Stop()

	// Give up after a while.  We need enough time to cover from when containers are brought up to when they should pass
	timeout := time.After(5 * time.Minute)

	// Attempt to check the server.  Quit if we pass timeout
	for {
		select {
		case <-tck.C:
			code := retryFunc()
			if code == 0 {
				return 0
			}
		case <-timeout:
			return -12
		}
	}
}

func fetcher(c *client.OdriveResponder, gem *events.GEM) error {
	userDn := gem.Payload.UserDN
	objectId := gem.Payload.ObjectID

	if gem.Action == "create" {
		odc, err := client.NewClient(c.Conf)
		if err != nil {
			return err
		}
		odc.MyDN = userDn
		obj, err := odc.GetObject(objectId)
		if err != nil {
			return err
		}
		c.Note("created: %s %s %s", objectId, obj.ContentType, obj.Name)
	}
	return nil
}
