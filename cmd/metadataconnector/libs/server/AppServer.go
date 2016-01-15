package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"github.com/jmoiron/sqlx"
)

/*
AppServer contains definition for the metadata server
*/
type AppServer struct {
	Port       int
	Bind       string
	Addr       string
	MetadataDB *sqlx.DB
}

/* ServeHTTP handles the routing of requests
 */
func (h AppServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// See who is making the call
	who := config.GetDistinguishedName(r.TLS.PeerCertificates[0])
	// Add/Load them
	user := dao.GetUserByDistinguishedName(h.MetadataDB, who)
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
		h.listObjects(w, r)
	case r.Method == "POST" && strings.HasSuffix(r.URL.RequestURI(), "/object"):
		h.createObject(w, r)

		// review: convert to restful. these are the thrift names that were planned
	case strings.Index(r.URL.RequestURI(), "/addObjectShare") == 0:
		h.addObjectShare(w, r)
	case strings.Index(r.URL.RequestURI(), "/addObjectSubscription") == 0:
		h.addObjectSubscription(w, r)
	case strings.Index(r.URL.RequestURI(), "/addObjectToFavorites") == 0:
		h.addObjectToFavorites(w, r)
	case strings.Index(r.URL.RequestURI(), "/addObjectToFolder") == 0:
		h.addObjectToFolder(w, r)
	case strings.Index(r.URL.RequestURI(), "/changeOwner") == 0:
		h.changeOwner(w, r)
	case strings.Index(r.URL.RequestURI(), "/createFolder") == 0:
		h.createFolder(w, r)
	case strings.Index(r.URL.RequestURI(), "/createObject") == 0:
		h.createObject(w, r)
	case strings.Index(r.URL.RequestURI(), "/deleteObjectForever") == 0:
		h.deleteObjectForever(w, r)
	case strings.Index(r.URL.RequestURI(), "/deleteObject") == 0:
		h.deleteObject(w, r)
	case strings.Index(r.URL.RequestURI(), "/getObjectStreamForRevision") == 0:
		h.getObjectStreamForRevision(w, r)
	case strings.Index(r.URL.RequestURI(), "/getObjectStream") == 0:
		h.getObjectStream(w, r)
	case strings.Index(r.URL.RequestURI(), "/getObject") == 0:
		h.getObject(w, r)
	case strings.Index(r.URL.RequestURI(), "/getRelationships") == 0:
		h.getRelationships(w, r)
	case strings.Index(r.URL.RequestURI(), "/listFavorites") == 0:
		h.listFavorites(w, r)
	case strings.Index(r.URL.RequestURI(), "/listObjectRevisions") == 0:
		h.listObjectRevisions(w, r)
	case strings.Index(r.URL.RequestURI(), "/listObjectsImages") == 0:
		h.listObjectsImages(w, r)
	case strings.Index(r.URL.RequestURI(), "/listObjectsTrashed") == 0:
		h.listObjectsTrashed(w, r)
	case strings.Index(r.URL.RequestURI(), "/listObjects") == 0:
		h.listObjects(w, r)
	case strings.Index(r.URL.RequestURI(), "/listObjectShares") == 0:
		h.listObjectShares(w, r)
	case strings.Index(r.URL.RequestURI(), "/listObjectsSubscriptions") == 0:
		h.listObjectsSubscriptions(w, r)
	case strings.Index(r.URL.RequestURI(), "/listUserObjectShares") == 0:
		h.listUserObjectShares(w, r)
	case strings.Index(r.URL.RequestURI(), "/listUserObjectsShared") == 0:
		h.listUserObjectsShared(w, r)
	case strings.Index(r.URL.RequestURI(), "/moveObject") == 0:
		h.moveObject(w, r)
	case strings.Index(r.URL.RequestURI(), "/query") == 0:
		h.query(w, r)
	case strings.Index(r.URL.RequestURI(), "/removeObjectFromFavorites") == 0:
		h.removeObjectFromFavorites(w, r)
	case strings.Index(r.URL.RequestURI(), "/removeObjectFromFolder") == 0:
		h.removeObjectFromFolder(w, r)
	case strings.Index(r.URL.RequestURI(), "/removeObjectFromTrash") == 0:
		h.removeObjectFromTrash(w, r)
	case strings.Index(r.URL.RequestURI(), "/removeObjectShare") == 0:
		h.removeObjectShare(w, r)
	case strings.Index(r.URL.RequestURI(), "/removeObjectSubscription") == 0:
		h.removeObjectSubscription(w, r)
	case strings.Index(r.URL.RequestURI(), "/updateObjectPermissions") == 0:
		h.updateObjectPermissions(w, r)
	case strings.Index(r.URL.RequestURI(), "/updateObjectStream") == 0:
		h.updateObjectStream(w, r)
	case strings.Index(r.URL.RequestURI(), "/updateObject") == 0:
		h.updateObject(w, r)
	default:
		who := config.GetDistinguishedName(r.TLS.PeerCertificates[0])
		msg := who + " requested uri: " + r.URL.RequestURI() + " from address: " + r.RemoteAddr + " with user agent: " + r.UserAgent()
		log.Println("WARN: " + msg)
		http.Error(w, "Resource not found", 404)
	}
}
