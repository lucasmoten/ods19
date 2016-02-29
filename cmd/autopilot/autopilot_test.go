package main

import (
	"decipher.com/oduploader/autopilot"
	"decipher.com/oduploader/protocol"
	"log"
	"net/http"
	"strconv"
	"testing"
)

/*
  Do a simple sequence to see that it actually works.
	Capture the output in markdown so that we can see the raw http.
*/

var userID0 = 0
var userID1 = 1

func resErrCheck(t *testing.T, res *http.Response, err error) {
	if err != nil {
            log.Printf("error came back:%v", err)
			t.Fail()
		}
		if res.StatusCode != http.StatusOK {
            log.Printf("http status must be ok")
			t.Fail()
		}    
}

func xTestUpdate(t *testing.T) {
	//This test is actually fast (particularly the second time around),
	//but it does use the real server.
	if testing.Short() == false {
    	var res *http.Response
    	var err error
    	var link *protocol.Object
        //Upload a random file
    	link, res, err = autopilot.DoUpload(userID0, false, "uploading a file to update")
        resErrCheck(t,res,err)

        fname := link.Name
        //Download that same file and get an updated link
		link, res, err = autopilot.DownloadLinkByName(fname, userID0, "get the file we uploaded")
        resErrCheck(t,res,err)
        
        //Update that file (modify in the *download* cache and send it up)
        oldChangeToken := link.ChangeToken
        res, err = autopilot.DoUpdateLink(userID0, link, "updating a file", "xxxx")
        resErrCheck(t,res, err)

        //Change token must be new on update
        if oldChangeToken == link.ChangeToken {
            log.Printf("change token must be new on update")
            t.Fail()
        }
        
        //Download that same file.  It should have the xxxx in the tail of it.
		link, res, err = autopilot.DownloadLinkByName(fname, userID0, "get the file we uploaded")
        resErrCheck(t,res, err)
	}    
}

func TestShare(t *testing.T) {
	//This test is actually fast (particularly the second time around),
	//but it does use the real server.
	if testing.Short() == false {
		RunShare(t)
	}
}

func RunShare(t *testing.T) {
	var res *http.Response
	var err error
	var link *protocol.Object
	var links *protocol.ObjectResultset
	var users []*protocol.User
	//Have both users do an upload and a download so they both exist
	//Remember the first upload link, because that is what we will share
	link, res, err = UploadDownload(t, userID0)
	_, res, err = UploadDownload(t, userID1)
	//The first user gets a list of all users, and is looking for somebody to share with
	users, res, err = autopilot.DoUserList(userID0, "See which users exist as a side-effect of visiting the site with their certificates.")
    resErrCheck(t,res, err)
	if len(users) < 2 {
		t.Fail()
	}
	if link != nil {
		//XXX Share with the first user that is not us
		//We need to get rid of the numerical id at some point
		res, err = autopilot.DoShare(userID0, link, userID1, "Alice shares file to Bob")
        resErrCheck(t,res, err)

		//List this users shares
		links, res, err = autopilot.FindShares(userID1, "Look at the shares that Bob has")
        resErrCheck(t,res, err)
		if len(links.Objects) == 0 {
			t.Fail()
		}
		//Second user download the first thing that was shared to him
		//XXX we could hunt for the right &links.Objects[n], but
		//just passing in link to make it simple
		res, err = DownloadLink(t, userID1, link)
        resErrCheck(t,res, err)
	} else {
		t.Fail()
	}

}

func UploadDownload(t *testing.T, user int) (link *protocol.Object, res *http.Response, err error) {
	//Upload some random file
	link, res, err = autopilot.DoUpload(user, false, "Uploading a file for Alice")
    resErrCheck(t,res, err)
	res, err = DownloadLink(t, user, link)
	return
}

func DownloadLink(t *testing.T, user int, link *protocol.Object) (res *http.Response, err error) {
	res, err = autopilot.DoDownloadLink(user, link, strconv.Itoa(user)+" downloads the file")
    resErrCheck(t,res, err)
	return
}

func init() {
	autopilot.Init()
}
