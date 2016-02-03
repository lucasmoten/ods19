package server

import (
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
	"encoding/hex"
	//"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	//"strconv"
)

func (h AppServer) dumpCacheLocation() {
	//XXX temp hack just so we can see in the container.
	//Files is an array, which is bad
	//if the queue is large, or accessed often!!!
	files, err := ioutil.ReadDir(h.CacheLocation)
	if err != nil {
		log.Printf("Error reading cache dir:%v", err)
		return
	}
	for _, f := range files {
		log.Printf("%s", f.Name())
	}
}

func (h AppServer) getObjectStream(w http.ResponseWriter, r *http.Request, caller Caller) {

	// Identify requested object
	objectID := getIDOfObjectTORetrieveStream(r.URL.RequestURI())
	// If not valid, return
	if objectID == "" {
		h.sendErrorResponse(w, 400, nil, "URI provided by caller does not specify an object identifier")
		return
	}
	// Convert to byte
	objectIDByte, err := hex.DecodeString(objectID)
	if err != nil {
		h.sendErrorResponse(w, 400, nil, "Identifier provided by caller is not a hexidecimal string")
		return
	}
	// Retrieve from database
	var objectRequested models.ODObject
	objectRequested.ID = objectIDByte
	object, err := dao.GetObject(h.MetadataDB, &objectRequested, false)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "cannot get object")
		return
	}

	// Authorization checks
	canRetrieve := false
	if object.OwnedBy.String == caller.DistinguishedName {
		////The AAC check function exists.  I'm not sure how to call it here yet
		////without the acm and token type already given.
		//if h.AAC.CheckAccess(caller.DistinguishedName, "pki-dias", acm) {
		// canRetrieve = true
		//}
		canRetrieve = true
	}
	// TODO Check object permission grants
	/////note... can't decrypt the stream without the grant.

	if !canRetrieve {
		h.sendErrorResponse(w, 403, nil, "Caller does not have permission to the requested object")
	}

	// TODO: Based upon object metadata, get the object from S3
	//		object.ContentConnector
	//		object.ContentHash
	if _, err = os.Stat(h.CacheLocation); os.IsNotExist(err) {
		err = os.Mkdir(h.CacheLocation, 0700)
		log.Printf("Creating cache directory %s", h.CacheLocation)
		if err != nil {
			log.Printf("Cannot create cache directory: %v", err)
			return
		}
	}
	h.dumpCacheLocation()

	//hasStream := false
	cipherTextName := h.CacheLocation + "/" + object.ContentConnector.String + ".cached"
	_, err = os.Stat(cipherTextName)
	if err == nil {
		log.Printf("file exists and is cached as: %s", object.ContentConnector.String)
		//hasStream = true
	} else {
		if os.IsNotExist(err) {
			log.Printf("file is not cached: %v", err)
			//Need to recache it - a blocking operation
		}
	}
	var key []byte
	iv := object.EncryptIV
	if len(object.Permissions) == 0 {
		log.Printf("We can't decrypt files that don't have permissions!")
		//hasStream = false
	} else {
		key = object.Permissions[0].EncryptKey
	}
	log.Printf("decrypt with iv:%v key:%v", iv, key)

	cipherText, err := os.Open(cipherTextName)
	if err != nil {
		log.Printf("Unable to open ciphertext %s:%v", cipherTextName, err)
		return
	}
	defer cipherText.Close()

	//Actually send back the ciphertext
	_, _, err = doCipherByReaderWriter(
		cipherText,
		w,
		key,
		iv,
	)
	if err != nil {
		log.Printf("error sending decrypted ciphertext %s:%v", cipherTextName, err)
		return
	}
	/*
		//XXX should be returning an http error code in this case
		if !hasStream {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, pageTemplateStart, "getObjectStream", caller.DistinguishedName)
			fmt.Fprintf(w, "No content")
			fmt.Fprintf(w, pageTemplateEnd)
			return
		}

		contentType := "text/html"
		if object.ContentType.Valid {
			contentType = object.ContentType.String
		}
		w.Header().Set("Content-Type", contentType)
		if object.ContentSize.Valid {
			w.Header().Set("Content-Length", strconv.FormatInt(object.ContentSize.Int64, 10))
		}
		fmt.Fprintf(w, pageTemplateStart, "getObjectStream", caller.DistinguishedName)
		fmt.Fprintf(w, pageTemplateEnd)
	*/
}

// getIDOfObjectTORetrieveStream accepts a passed in URI and finds whether an
// object identifier was passed within it for which the content stream is sought
func getIDOfObjectTORetrieveStream(uri string) string {
	re, _ := regexp.Compile("/object/(.*)/stream")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) == 0 {
		return ""
	}
	value := uri[matchIndexes[2]:matchIndexes[3]]
	return value
}
