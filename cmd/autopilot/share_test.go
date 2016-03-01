package main

import (
	"decipher.com/oduploader/autopilot"
	"decipher.com/oduploader/protocol"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"testing"
)

/*
  Do a simple sequence to see that it actually works.
	Capture the output in markdown so that we can see the raw http.
*/

func TestShare(t *testing.T) {
	//Autopilot needs to keep a trace that isn't tangled with other logs.
	//So give it a file.
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

	fmt.Fprintf(ap.Log, "#TestShare")
	//This test is actually fast (particularly the second time around),
	//but it does use the real server.
	if testing.Short() == false {
		RunShare(t, ap)
	}

}

func RunShare(t *testing.T, ap *autopilot.AutopilotContext) {
	var res *http.Response
	var err error
	var link *protocol.Object
	var links *protocol.ObjectResultset
	var users []*protocol.User
	//Have both users do an upload and a download so they both exist
	//Remember the first upload link, because that is what we will share
	link, res, err = UploadDownload(t, ap, userID0)
	_, res, err = UploadDownload(t, ap, userID1)
	//The first user gets a list of all users, and is looking for somebody to share with
	users, res, err = ap.DoUserList(userID0, "See which users exist as a side-effect of visiting the site with their certificates.")
	resErrCheck(t, res, err)
	if len(users) < 2 {
		t.Fail()
	}
	if link != nil {
		//XXX Share with the first user that is not us
		//We need to get rid of the numerical id at some point
		res, err = ap.DoShare(userID0, link, userID1, "Alice shares file to Bob")
		resErrCheck(t, res, err)

		//List this users shares
		links, res, err = ap.FindShares(userID1, "Look at the shares that Bob has")
		resErrCheck(t, res, err)
		if len(links.Objects) == 0 {
			t.Fail()
		}
		//Second user download the first thing that was shared to him
		//XXX we could hunt for the right &links.Objects[n], but
		//just passing in link to make it simple
		res, err = DownloadLink(t, ap, userID1, link)
		resErrCheck(t, res, err)
	} else {
		t.Fail()
	}

}

func UploadDownload(t *testing.T, ap *autopilot.AutopilotContext, user int) (link *protocol.Object, res *http.Response, err error) {
	//Upload some random file
	link, res, err = ap.DoUpload(user, false, "Uploading a file for Alice")
	resErrCheck(t, res, err)
	res, err = DownloadLink(t, ap, user, link)
	return
}

func DownloadLink(t *testing.T, ap *autopilot.AutopilotContext, user int, link *protocol.Object) (res *http.Response, err error) {
	res, err = ap.DoDownloadLink(user, link, strconv.Itoa(user)+" downloads the file")
	resErrCheck(t, res, err)
	return
}
