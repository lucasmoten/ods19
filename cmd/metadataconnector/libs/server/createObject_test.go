package server_test

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	"decipher.com/oduploader/autopilot"
	"decipher.com/oduploader/protocol"
)

func TestCreateRealObject(t *testing.T) {
	if testing.Short() {
		t.Skip()
	} else {
		runTestCreateRealObject(t)
	}
}

func runTestCreateRealObject(t *testing.T) {
	userID0 := 0
	logHandle, err := os.Create("TestShare.md")
	if err != nil {
		log.Printf("Unable to start scenarion: %v", err)
		t.Fail()
	}
	defer logHandle.Close()

	ap, err := autopilot.NewAutopilotContext(logHandle)
	if err != nil {
		log.Printf("Unable to start autopilot context: %v", err)
		t.Fail()
	}

	fmt.Fprintf(ap.Log, "#TestCreateObject\n")

	var link *protocol.Object
	//Have both users do an upload and a download so they both exist
	//Remember the first upload link, because that is what we will share
	link, res, err := ap.DoUpload(userID0, false, "Uploading a file for User 0")

	if err != nil {
		log.Printf("error came back:%v", err)
		t.Fail()
	}
	if res == nil {
		log.Printf("we got a null result back")
		t.Fail()
	}
	if res.StatusCode != http.StatusOK {
		log.Printf("http status must be ok.  we got %d.  %s", res.StatusCode, res.Status)
		t.Fail()
	}

	if len(link.ID) == 0 {
		log.Printf("no ID on created object")
		t.Fail()
	}

	if len(link.ContentHash) == 0 {
		log.Printf("no Hash on created object")
		t.Fail()
	}

	if len(link.ChangeToken) == 0 {
		log.Printf("no Hash on created object")
		t.Fail()
	}
}
