package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	"decipher.com/oduploader/autopilot"
	"decipher.com/oduploader/protocol"
)

/*
  Do a simple sequence to see that it actually works.
	Capture the output in markdown so that we can see the raw http.
*/

func TestUpdate(t *testing.T) {
	//Autopilot needs to keep a trace that isn't tangled with other logs.
	//So give it a file.
	logHandle, err := os.Create("TestUpdate.md")
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

	fmt.Fprintf(ap.Log, "#TestUpdate\n")
	//This test is actually fast (particularly the second time around),
	//but it does use the real server.
	if testing.Short() == false {
		var res *http.Response
		var err error
		var link *protocol.Object
		//Upload a random file
		link, res, err = ap.DoUpload(userID0, false, "uploading a file to update")
		resErrCheck(t, res, err)

		if link == nil {
			log.Printf("didnt get a link back from an upload")
			t.Fail()
		}

		fname := link.Name
		//Download that same file and get an updated link
		link, res, err = ap.DownloadLinkByName(fname, userID0, "get the file we uploaded")
		resErrCheck(t, res, err)

		if link == nil {
			log.Printf("didnt get a link back from second upload")
			t.Fail()
		}

		//Update that file (modify in the *download* cache and send it up)
		oldChangeToken := link.ChangeToken
		res, err = ap.DoUpdateLink(userID0, link, "updating a file", "xxxx")
		resErrCheck(t, res, err)

		//Change token must be new on update
		if oldChangeToken != link.ChangeToken {
			fmt.Printf("change token must be new on update")
			t.Fail()
		}

		//Download that same file.  It should have the xxxx in the tail of it.
		link, res, err = ap.DownloadLinkByName(fname, userID0, "get the file we uploaded")
		resErrCheck(t, res, err)
	}
}