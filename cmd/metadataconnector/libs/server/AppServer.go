package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"

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
	ServicePrefix string
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

	var uri = r.URL.RequestURI()

	// These regular expressions to match uri patterns
	var rxFavorites = regexp.MustCompile(h.ServicePrefix + "/favorites$")
	var rxFolder = regexp.MustCompile(h.ServicePrefix + "/folder$")
	var rxImages = regexp.MustCompile(h.ServicePrefix + "/images$")
	var rxObject = regexp.MustCompile(h.ServicePrefix + "/object$")
	var rxQuery = regexp.MustCompile(h.ServicePrefix + "/query/.*")
	var rxShared = regexp.MustCompile(h.ServicePrefix + "/shared$")
	var rxShares = regexp.MustCompile(h.ServicePrefix + "/shares$")
	var rxTrash = regexp.MustCompile(h.ServicePrefix + "/trash$")
	var rxObjectChangeOwner = regexp.MustCompile(h.ServicePrefix + "/object/.*/changeowner/.*")
	var rxObjectExpunge = regexp.MustCompile(h.ServicePrefix + "/object/.*/expunge$")
	var rxObjectFavorite = regexp.MustCompile(h.ServicePrefix + "/object/.*/favorite$")
	var rxObjectLink = regexp.MustCompile(h.ServicePrefix + "/object/.*/link/.*")
	var rxObjectLinks = regexp.MustCompile(h.ServicePrefix + "/object/.*/links$")
	var rxObjectMove = regexp.MustCompile(h.ServicePrefix + "/object/.*/move/.*")
	var rxObjectPermission = regexp.MustCompile(h.ServicePrefix + "/object/.*/permission/.*")
	var rxObjectProperties = regexp.MustCompile(h.ServicePrefix + "/object/.*/properties$")
	var rxObjects = regexp.MustCompile(h.ServicePrefix + "/objects$")
	var rxObjectShare = regexp.MustCompile(h.ServicePrefix + "/object/.*/share$")
	var rxObjectStream = regexp.MustCompile(h.ServicePrefix + "/object/.*/stream$")
	var rxObjectStreamRevision = regexp.MustCompile(h.ServicePrefix + "/object/.*/history/.*/stream$")
	var rxObjectSubscription = regexp.MustCompile(h.ServicePrefix + "/object/.*/subscribe$")
	var rxListObjects = regexp.MustCompile(h.ServicePrefix + "/object/.*/list$")
	var rxListObjectRevisions = regexp.MustCompile(h.ServicePrefix + "/object/.*/history$")
	var rxListObjectShares = regexp.MustCompile(h.ServicePrefix + "/object/.*/shares$")
	var rxListObjectSubscriptions = regexp.MustCompile(h.ServicePrefix + "/object/.*/subscriptions$")
	var rxListImages = regexp.MustCompile(h.ServicePrefix + "/images/.*/list$")
	var rxTrashObject = regexp.MustCompile(h.ServicePrefix + "/trash/.*")

	switch r.Method {
	case "GET":
		switch {
		case uri == h.ServicePrefix+"/", uri == h.ServicePrefix+"//":
			h.home(w, r)
		case uri == h.ServicePrefix+"/favicon.ico", uri == h.ServicePrefix+"//favicon.ico":
			h.favicon(w, r)
		// from longest to shortest...
		case rxObjectStreamRevision.MatchString(uri):
			h.getObjectStreamForRevision(w, r, caller)
		case rxObjectStream.MatchString(uri):
			h.getObjectStream(w, r, caller)
		case rxObjectProperties.MatchString(uri):
			h.getObject(w, r, caller)
		case rxObjectLinks.MatchString(uri):
			h.getRelationships(w, r, caller)
		case rxObjects.MatchString(uri):
			h.listObjects(w, r, caller)
		case rxListObjects.MatchString(uri):
			h.listObjects(w, r, caller)
		case rxImages.MatchString(uri), rxListImages.MatchString(uri):
			h.listObjectsImages(w, r, caller)
		case rxListObjectRevisions.MatchString(uri):
			h.listObjectRevisions(w, r, caller)
		case rxListObjectShares.MatchString(uri):
			h.listObjectShares(w, r, caller)
		case rxListObjectSubscriptions.MatchString(uri):
			h.listObjectsSubscriptions(w, r, caller)
		// single quick matchers
		case rxFavorites.MatchString(uri):
			h.listFavorites(w, r, caller)
		case rxShared.MatchString(uri):
			h.listUserObjectsShared(w, r, caller)
		case rxShares.MatchString(uri):
			h.listUserObjectShares(w, r, caller)
		// TODO: Find out why this is showing up for /object//list
		case rxObject.MatchString(uri):
			h.createObject(w, r, caller)
		case rxTrash.MatchString(uri):
			h.listObjectsTrashed(w, r, caller)
		case rxQuery.MatchString(uri):
			h.query(w, r, caller)
		default:
			msg := caller.DistinguishedName + " from address " + r.RemoteAddr + " using " + r.UserAgent() + " unhandled operation " + r.Method + " " + uri
			log.Println("WARN: " + msg)
			http.Error(w, "Resource not found", 404)
		}
	case "POST":
		switch {
		case rxObjectShare.MatchString(uri):
			h.addObjectShare(w, r, caller)
		case rxObjectSubscription.MatchString(uri):
			h.addObjectSubscription(w, r, caller)
		case rxObjectFavorite.MatchString(uri):
			h.addObjectToFavorites(w, r, caller)
		case rxObjectLink.MatchString(uri):
			h.addObjectToFolder(w, r, caller)
		case rxFolder.MatchString(uri):
			h.createFolder(w, r, caller)
		case rxObject.MatchString(uri):
			h.createObject(w, r, caller)
		case rxListObjects.MatchString(uri):
			h.listObjects(w, r, caller)
		case rxQuery.MatchString(uri):
			h.query(w, r, caller)
		default:
			msg := caller.DistinguishedName + " from address " + r.RemoteAddr + " using " + r.UserAgent() + " unhandled operation " + r.Method + " " + uri
			log.Println("WARN: " + msg)
			http.Error(w, "Resource not found", 404)
		}
	case "PUT":
		switch {
		case rxObjectChangeOwner.MatchString(uri):
			h.changeOwner(w, r, caller)
		case rxObjectMove.MatchString(uri):
			h.moveObject(w, r, caller)
		case rxObjectPermission.MatchString(uri):
			h.updateObjectPermissions(w, r, caller)
		case rxObjectProperties.MatchString(uri):
			h.updateObject(w, r, caller)
		case rxObjectStream.MatchString(uri):
			h.updateObjectStream(w, r, caller)
		default:
			msg := caller.DistinguishedName + " from address " + r.RemoteAddr + " using " + r.UserAgent() + " unhandled operation " + r.Method + " " + uri
			log.Println("WARN: " + msg)
			http.Error(w, "Resource not found", 404)
		}
	case "DELETE":
		switch {
		case rxObject.MatchString(uri):
			h.deleteObject(w, r, caller)
		case rxObjectExpunge.MatchString(uri):
			h.deleteObjectForever(w, r, caller)
		case rxObjectFavorite.MatchString(uri):
			h.removeObjectFromFavorites(w, r, caller)
		case rxObjectLink.MatchString(uri):
			h.removeObjectFromFolder(w, r, caller)
		case rxTrashObject.MatchString(uri):
			h.removeObjectFromTrash(w, r, caller)
		case rxObjectShare.MatchString(uri):
			h.removeObjectShare(w, r, caller)
		case rxObjectSubscription.MatchString(uri):
			h.removeObjectSubscription(w, r, caller)
		default:
			msg := caller.DistinguishedName + " from address " + r.RemoteAddr + " using " + r.UserAgent() + " unhandled operation " + r.Method + " " + uri
			log.Println("WARN: " + msg)
			http.Error(w, "Resource not found", 404)
		}
	}

	// switch {
	//
	// // case ((strings.Index(r.URL.RequestURI(), "/object/") > -1) && (strings.Index(r.URL.RequestURI(), "/list") > -1)):
	// // 	h.listObjects(w, r, caller)
	// // case r.Method == "POST" && strings.HasSuffix(r.URL.RequestURI(), "/object"):
	// // 	h.createObject(w, r, caller)
	// // case r.Method == "GET" && ((strings.Index(r.URL.RequestURI(), "/object/") > -1) && (strings.Index(r.URL.RequestURI(), "/stream") > -1)):
	// // 	h.getObjectStream(w, r, caller)
	//
	// // review: convert to restful. these are the thrift names that were planned
	//
	// case strings.Index(r.URL.RequestURI(), "/addObjectToFavorites") == 0:
	// 	h.addObjectToFavorites(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/addObjectToFolder") == 0:
	// 	h.addObjectToFolder(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/changeOwner") == 0:
	// 	h.changeOwner(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/createFolder") == 0:
	// 	h.createFolder(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/createObject") == 0:
	// 	h.createObject(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/deleteObjectForever") == 0:
	// 	h.deleteObjectForever(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/deleteObject") == 0:
	// 	h.deleteObject(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/getObjectStreamForRevision") == 0:
	// 	h.getObjectStreamForRevision(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/getObjectStream") == 0:
	// 	h.getObjectStream(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/getObject") == 0:
	// 	h.getObject(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/getRelationships") == 0:
	// 	h.getRelationships(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/listFavorites") == 0:
	// 	h.listFavorites(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/listObjectRevisions") == 0:
	// 	h.listObjectRevisions(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/listObjectsImages") == 0:
	// 	h.listObjectsImages(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/listObjectsTrashed") == 0:
	// 	h.listObjectsTrashed(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/listObjects") == 0:
	// 	h.listObjects(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/listObjectShares") == 0:
	// 	h.listObjectShares(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/listObjectsSubscriptions") == 0:
	// 	h.listObjectsSubscriptions(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/listUserObjectShares") == 0:
	// 	h.listUserObjectShares(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/listUserObjectsShared") == 0:
	// 	h.listUserObjectsShared(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/moveObject") == 0:
	// 	h.moveObject(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/query") == 0:
	// 	h.query(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/removeObjectFromFavorites") == 0:
	// 	h.removeObjectFromFavorites(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/removeObjectFromFolder") == 0:
	// 	h.removeObjectFromFolder(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/removeObjectFromTrash") == 0:
	// 	h.removeObjectFromTrash(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/removeObjectShare") == 0:
	// 	h.removeObjectShare(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/removeObjectSubscription") == 0:
	// 	h.removeObjectSubscription(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/updateObjectPermissions") == 0:
	// 	h.updateObjectPermissions(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/updateObjectStream") == 0:
	// 	h.updateObjectStream(w, r, caller)
	// case strings.Index(r.URL.RequestURI(), "/updateObject") == 0:
	// 	h.updateObject(w, r, caller)
	// default:
	// 	msg := caller.DistinguishedName + " requested uri: " + r.URL.RequestURI() + " from address: " + r.RemoteAddr + " with user agent: " + r.UserAgent()
	// 	log.Println("WARN: " + msg)
	// 	http.Error(w, "Resource not found", 404)
	// }
}

// GetCaller populates a Caller object based upon request headers and peer
// certificates. Logically this is intended to work with or without NGINX as
// a front end
func GetCaller(r *http.Request) Caller {
	var caller Caller
	caller.UserDistinguishedName = r.Header.Get("USER_DN")
	caller.ExternalSystemDistinguishedName = r.Header.Get("EXTERNAL_SYS_DN")
	if caller.UserDistinguishedName != "" {
		log.Println("Assigning distinguished name from USER_DN")
		caller.DistinguishedName = caller.UserDistinguishedName
	} else {
		if len(r.TLS.PeerCertificates) > 0 {
			log.Println("Assigning distinguished name from peer certificate")
			caller.DistinguishedName = config.GetDistinguishedName(r.TLS.PeerCertificates[0])
		} else {
			log.Println("WARNING: No distinguished name set!!!")
		}
	}
	caller.CommonName = config.GetCommonName(caller.DistinguishedName)
	return caller
}
