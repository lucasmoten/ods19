package main

import (
	//"decipher.com/oduploader/cmd/metadataconnector/libs/server"
	"log"
	"testing"
)

func TestUpload(t *testing.T) {
	//Upload some random file
	link := doUpload(userID)
	log.Printf("")
	//var listing server.ObjectLinkResponse
	//getObjectLinkResponse(userID, &listing)
	//link = &listing.Objects[0]

	if link != nil {
		//Download THAT file
		doDownloadLink(userID, link)
		log.Printf("")
		//Update THAT file
		doUpdateLink(userID, link)
		log.Printf("")
		//Try to re-download it
		doDownloadLink(userID, link)
		log.Printf("")
	} else {
		log.Printf("We uploaded a file but got no link back!")
	}
}

//User 0 is actually "test tester10", using mod10 for userid to testids
var userID = 0

func init() {
	generatePopulation()
	log.Printf("Using autopilot uploadCache:%s", clients[userID].UploadCache)
}
