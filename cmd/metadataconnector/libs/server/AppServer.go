package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/jmoiron/sqlx"
)

// AppServer contains definition for the metadata server
type AppServer struct {
	Port          int
	Bind          string
	Addr          string
	MetadataDB    *sqlx.DB
	S3            *s3.S3
	AWSSession    *session.Session
	CacheLocation string
}

// UserSession is per session information that needs to be passed around
type UserSession struct {
	User models.ODUser
}

func (h AppServer) findWho(r *http.Request) string {
	who := "anonymous" //Should be a DN
	if len(r.TLS.PeerCertificates) > 0 {
		//Direct 2way ssl connection
		who = config.GetDistinguishedName(r.TLS.PeerCertificates[0])
	} else {
		//get from a header
	}
	return who
}

// Caller provides the distinguished names obtained from specific request
// headers and peer certificate if called directly
type Caller struct {
	DistinguishedName               string
	UserDistinguishedName           string
	ExternalSystemDistinguishedName string
	CommonName                      string
}

// ServeHTTP handles the routing of requests
func (h AppServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	caller := GetCaller(r)

	// Add/Load them
	user := dao.GetUserByDistinguishedName(h.MetadataDB, caller.DistinguishedName)
	if len(user.ModifiedBy) == 0 {
		fmt.Println("User does not have modified by set!")
		jsonData, err := json.MarshalIndent(user, "", "  ")
		if err != nil {
			panic(err)
		}
		jsonified := string(jsonData)
		fmt.Println(jsonified)
	}

	switch {
	case r.URL.RequestURI() == "/":
		h.home(w, r)
	case r.URL.RequestURI() == "/favicon.ico":
		h.favicon(w, r)

	case ((strings.Index(r.URL.RequestURI(), "/object/") > -1) && (strings.Index(r.URL.RequestURI(), "/list") > -1)):
		h.listObjects(w, r, caller)
	case r.Method == "POST" && strings.HasSuffix(r.URL.RequestURI(), "/object"):
		h.createObject(w, r, caller)
	case r.Method == "GET" && ((strings.Index(r.URL.RequestURI(), "/object/") > -1) && (strings.Index(r.URL.RequestURI(), "/stream") > -1)):
		h.getObjectStream(w, r, caller)

		// review: convert to restful. these are the thrift names that were planned
	case strings.Index(r.URL.RequestURI(), "/addObjectShare") == 0:
		h.addObjectShare(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/addObjectSubscription") == 0:
		h.addObjectSubscription(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/addObjectToFavorites") == 0:
		h.addObjectToFavorites(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/addObjectToFolder") == 0:
		h.addObjectToFolder(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/changeOwner") == 0:
		h.changeOwner(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/createFolder") == 0:
		h.createFolder(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/createObject") == 0:
		h.createObject(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/deleteObjectForever") == 0:
		h.deleteObjectForever(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/deleteObject") == 0:
		h.deleteObject(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/getObjectStreamForRevision") == 0:
		h.getObjectStreamForRevision(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/getObjectStream") == 0:
		h.getObjectStream(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/getObject") == 0:
		h.getObject(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/getRelationships") == 0:
		h.getRelationships(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/listFavorites") == 0:
		h.listFavorites(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/listObjectRevisions") == 0:
		h.listObjectRevisions(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/listObjectsImages") == 0:
		h.listObjectsImages(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/listObjectsTrashed") == 0:
		h.listObjectsTrashed(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/listObjects") == 0:
		h.listObjects(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/listObjectShares") == 0:
		h.listObjectShares(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/listObjectsSubscriptions") == 0:
		h.listObjectsSubscriptions(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/listUserObjectShares") == 0:
		h.listUserObjectShares(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/listUserObjectsShared") == 0:
		h.listUserObjectsShared(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/moveObject") == 0:
		h.moveObject(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/query") == 0:
		h.query(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/removeObjectFromFavorites") == 0:
		h.removeObjectFromFavorites(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/removeObjectFromFolder") == 0:
		h.removeObjectFromFolder(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/removeObjectFromTrash") == 0:
		h.removeObjectFromTrash(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/removeObjectShare") == 0:
		h.removeObjectShare(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/removeObjectSubscription") == 0:
		h.removeObjectSubscription(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/updateObjectPermissions") == 0:
		h.updateObjectPermissions(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/updateObjectStream") == 0:
		h.updateObjectStream(w, r, caller)
	case strings.Index(r.URL.RequestURI(), "/updateObject") == 0:
		h.updateObject(w, r, caller)
	default:
		msg := caller.DistinguishedName + " requested uri: " + r.URL.RequestURI() + " from address: " + r.RemoteAddr + " with user agent: " + r.UserAgent()
		log.Println("WARN: " + msg)
		http.Error(w, "Resource not found", 404)
	}
}

// GetCaller populates a Caller object based upon request headers and peer
// certificates. Logically this is intended to work with or without NGINX as
// a front end
func GetCaller(r *http.Request) Caller {
	var caller Caller
	caller.UserDistinguishedName = r.Header.Get("USER_DN")
	caller.ExternalSystemDistinguishedName = r.Header.Get("EXTERNAL_SYS_DN")
	if caller.UserDistinguishedName != "" {
		caller.DistinguishedName = caller.UserDistinguishedName
	} else {
		if len(r.TLS.PeerCertificates) > 0 {
			caller.DistinguishedName = config.GetDistinguishedName(r.TLS.PeerCertificates[0])
		}
	}
	caller.CommonName = config.GetCommonName(caller.DistinguishedName)
	return caller
}
