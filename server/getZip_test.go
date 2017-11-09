package server_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"decipher.com/object-drive-server/server"

	"decipher.com/object-drive-server/util"

	"decipher.com/object-drive-server/protocol"
)

func TestZipCorrect(t *testing.T) {
	tester10 := 0
	duplicates := 0
	mapsFolder, err := makeFolderWithACMWithParentViaJSON("maps", "", ValidAcmCreateObjectSimple, tester10)
	if err != nil {
		t.Logf("failed to make folder: %v", err)
		t.FailNow()
	}
	//Remember individual objects
	someDataString := "lat=5,long=6"
	var objs []protocol.Object
	for i := 0; i < 10; i++ {
		objs = append(objs, testZipMakeFile(t, tester10, mapsFolder.ID, fmt.Sprintf("mapdata%d.txt", i), someDataString))
	}
	//duplicate an item to force renaming to trigger and catch rename bugs
	objs = append(objs, testZipMakeFile(t, tester10, mapsFolder.ID, fmt.Sprintf("mapdata%d.txt", 0), someDataString))
	mapsFolder2, err := makeFolderWithACMWithParentViaJSON("dat", mapsFolder.ID, ValidAcmCreateObjectSimple, tester10)
	//put in a file that came from a different directory
	objs = append(objs, testZipMakeFile(t, tester10, mapsFolder2.ID, "data.txt", someDataString))
	objs = append(objs, testZipMakeFile(t, tester10, mapsFolder2.ID, "mapdata0.txt", someDataString))
	//try to trip up the classification manifest - it will get renamed
	objs = append(objs, testZipMakeFile(t, tester10, mapsFolder2.ID, "classification_manifest.txt", someDataString))
	//put in duplicate identifiers to show that the duplicates are removed
	for i := 0; i < 60; i++ {
		objs = append(objs, objs[5])
		duplicates++
	}
	doTestZip(t, objs, someDataString, duplicates, http.StatusOK, trafficLogs[APISampleFile],
		&TrafficLogDescription{
			OperationName:       "Zip up some files",
			RequestDescription:  "Select a set of individual files (not directories) and send their identifiers in the request",
			ResponseDescription: "We get back a binary zip file as a response",
			ResponseBodyHide:    true,
		},
	)
}

//Get a 400 when we submit a trivial object list
func TestZipEmpty(t *testing.T) {
	tester10 := 0
	duplicates := 0
	//Remember individual objects
	someDataString := "lat=5,long=6"
	var objs []protocol.Object
	mapsFolder, err := makeFolderWithACMWithParentViaJSON("zipEmpty", "", ValidAcmCreateObjectSimple, tester10)
	if err != nil {
		t.Logf("error creating folder: %v", err)
		t.Fail()
	}
	objs = append(objs, *mapsFolder)
	doTestZip(t, objs, someDataString, duplicates, 400, nil, nil)
}

//Get a 404 when we submit a wrong object id
func TestZipWrong(t *testing.T) {
	tester10 := 0
	duplicates := 0
	//Remember individual objects
	someDataString := "lat=5,long=6"
	var objs []protocol.Object
	mapsFolder, err := makeFolderWithACMWithParentViaJSON("zipWrong", "", ValidAcmCreateObjectSimple, tester10)
	if err != nil {
		t.Logf("error creating folder: %v", err)
		t.Fail()
	}
	mapsFolder.ID = "e234e234e234e234e234e234e234e234"
	objs = append(objs, *mapsFolder)
	doTestZip(t, objs, someDataString, duplicates, 404, nil, nil)
}

//Get a 400 when we submit a wrong object list
func TestZipBad(t *testing.T) {
	tester01 := 1
	duplicates := 0
	//Remember individual objects
	someDataString := "lat=5,long=6"
	var objs []protocol.Object
	acm := server.ValidACMUnclassifiedFOUOSharedToTester01And02
	objs = append(objs, testZipMakeFileWithACM(t, tester01, "", fmt.Sprintf("tester01private.txt"), someDataString, acm))
	//test tester10 will perform the zip.  test tester01 owns the file.
	doTestZip(t, objs, someDataString, duplicates, 400, nil, nil)
}

func doTestZip(t *testing.T, objs []protocol.Object, someDataString string, duplicates int, expected int, trafficLog *TrafficLog, description *TrafficLogDescription) {
	tester10 := 0
	client := clients[tester10].Client

	t.Logf("Make some stuff to zip up")
	t.Logf("Generate test files that will be included in the zip")

	t.Logf("Include the individual files in the zip")
	var zipSpec protocol.Zip
	zipSpec.Disposition = "inline"
	zipSpec.FileName = "drive.zip"
	zipSpec.ObjectIDs = make([]string, 0)
	for _, o := range objs {
		zipSpec.ObjectIDs = append(zipSpec.ObjectIDs, o.ID)
	}
	jsonBytes, err := json.MarshalIndent(&zipSpec, "", "")
	if err != nil {
		t.Log("Could not marshal request to json")
		t.FailNow()
	}
	t.Logf("%s", string(jsonBytes))

	t.Logf("Actually perform a zip request and ensure that we get something back")
	uri := mountPoint + "/zip"
	t.Logf("trying: POST %s", uri)
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonBytes))
	if err != nil {
		t.Logf("unable to make request for zip: %v", err)
		t.FailNow()
	}
	if trafficLog != nil && description != nil {
		trafficLog.Request(t, req, description)
	}

	t.Logf("Starting zip: %v", time.Now())
	res, err := client.Do(req)
	t.Logf("Stopping zip: %v", time.Now())
	if err != nil {
		t.Logf("cannot get zip: %v", err)
		t.FailNow()
	}
	if trafficLog != nil && description != nil {
		trafficLog.Response(t, res)
	}
	defer util.FinishBody(res.Body)

	if res.StatusCode != expected {
		t.Logf("wrong status code: %d", res.StatusCode)
		t.FailNow()
	}

	if expected == http.StatusOK {
		t.Logf("Get the zip into a temporary file")
		tmp, err := ioutil.TempFile(".", "__tempzip__")
		if err != nil {
			t.Logf("cannot create temp zip file: %v", err)
			t.FailNow()
		}
		defer func() {
			tmp.Close()
			os.Remove(tmp.Name())
		}()
		io.Copy(tmp, res.Body)

		r, err := zip.OpenReader(tmp.Name())
		if err != nil {
			t.Logf("cannot open file for zip: %v", err)
			t.FailNow()
		}
		t.Logf("Check the contents for a manifest plus file count we inserted")
		returnedFileCount := len(objs) + 1 - duplicates
		filesRemaining := returnedFileCount
		for _, f := range r.File {
			t.Logf("%s\n", f.Name)
			filesRemaining--
			if f.Name == "classification_manifest.txt" {
				cf, err := f.Open()
				defer func() {
					cf.Close()
				}()
				if err != nil {
					t.Logf("unable to read manifest: %v", err)
					t.FailNow()
				}
				t.Logf("Zip looks valid")
			} else {
				if f.FileInfo().Size() != int64(len(someDataString)) {
					t.Logf(
						"data doesn't appear to be original length as we zipped: %d vs %d",
						f.FileInfo().Size(),
						int64(len(someDataString)),
					)
					t.FailNow()
				}
			}
		}
		if filesRemaining != 0 {
			t.Logf("wrong number of files in zip.  found %d", returnedFileCount)
			t.FailNow()
		}
		t.Logf("got %d files back", returnedFileCount)
	}
}

// Add a file into the zip file
func testZipMakeFile(t *testing.T, clientID int, parentID string, name string, data string) protocol.Object {
	return testZipMakeFileWithACM(t, clientID, parentID, name, data, ValidAcmCreateObjectSimple)
}

// Add a file into the zip file
func testZipMakeFileWithACM(t *testing.T, clientID int, parentID string, name string, data string, acm string) protocol.Object {
	client := clients[clientID].Client

	createRequest := protocol.CreateObjectRequest{
		Name:     name,
		TypeName: "File",
		RawAcm:   acm,
		ParentID: parentID,
	}

	var jsonBody []byte
	var err error
	jsonBody, err = json.Marshal(createRequest)
	if err != nil {
		t.Fail()
	}

	tmpName := name
	tmp, tmpCloser, err := GenerateTempFile(data)
	if err != nil {
		t.Errorf("Could not open temp file for write: %v\n", err)
	}
	defer tmpCloser()

	req, err := NewCreateObjectPOSTRequestRaw(
		"objects", "", tmp, tmpName, jsonBody)
	if err != nil {
		t.Errorf("Unable to create HTTP request: %v\n", err)
	}

	res, obj, err := DoWithDecodedResult2(client, req)

	if err != nil {
		t.Fail()
	}

	if res != nil && res.StatusCode != http.StatusOK {
		t.Fail()
	}

	return obj.(protocol.Object)
}
