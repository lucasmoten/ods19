package main

import (
	"decipher.com/oduploader/autopilot"
	"decipher.com/oduploader/protocol"
	"fmt"
	"net/http"
	"testing"
)

/*
  Do a simple sequence to see that it actually works.
	Capture the output in markdown so that we can see the raw http.
*/

func doTestUpdate(t *testing.T) {
    fmt.Println("#TestUpdate")
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
            fmt.Printf("change token must be new on update")
            t.Fail()
        }
        
        //Download that same file.  It should have the xxxx in the tail of it.
		link, res, err = autopilot.DownloadLinkByName(fname, userID0, "get the file we uploaded")
        resErrCheck(t,res, err)
	}    
}

