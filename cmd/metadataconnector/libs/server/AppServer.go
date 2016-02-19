package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/performance"
	aac "decipher.com/oduploader/services/aac"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/jmoiron/sqlx"
)

// AppServer contains definition for the metadata server
type AppServer struct {
	// Port is the TCP port that the web server listens on
	Port int
	// Bind is the Network Address that the web server will use.
	Bind string
	// Addr is the combined network address and port the server listens on
	Addr string
	// MetadataDB is a handle to the database connection
	MetadataDB *sqlx.DB
	DAO        dao.DataAccessLayer
	// TODO: Convert this as appropriate to non implementation specific
	// S3 is the handle to the S3 Client
	S3 *s3.S3
	// AWSSession is a handle to active Amazon Web Service session
	AWSSession *session.Session
	// CacheLocation is the location locally for temporary storage of content
	// streams encrypted at rest
	CacheLocation string
	// ServicePrefix is the base RootURL for all public operations of web server
	ServicePrefix string
	// AAC is a handle to the Authorization and Access Control client
	// TODO: This will need to be converted to be pluggable later
	AAC *aac.AacServiceClient
	// TODO: Classifications is ????
	Classifications map[string]string
	// MasterKey is the secret passphrase used in scrambling keys
	MasterKey string
	// Tracker captures metrics about upload/download begin and end time and transfer bytes
	Tracker *performance.JobReporters
	// TemplateCache is location of HTML templates used by server
	TemplateCache *template.Template
	// StaticDir is location of static objects like javascript
	StaticDir string
}

// Caller provides the distinguished names obtained from specific request
// headers and peer certificate if called directly
type Caller struct {
	// DistinguishedName is the unique identity of a user
	DistinguishedName string
	// UserDistinguishedName holds the value passed in header USER_DN
	UserDistinguishedName string
	// ExternalSystemDistinguishedName holds the value passed in header EXTERNAL_SYS_DN
	ExternalSystemDistinguishedName string
	// CommonName is the CN value part of the DistinguishedName
	CommonName string
}

// initRegex should be used everywhere to ensure that static regexes are compiled.
func initRegex(rx string) *regexp.Regexp {
	compiled, err := regexp.Compile(rx)
	if err != nil {
		log.Printf("Unable to compile regex %s:%v", rx, err)
		return nil
	}
	return compiled
}

// StaticRx is a bunch of static compiled regexes
type StaticRx struct {
	Favorites               *regexp.Regexp
	Folder                  *regexp.Regexp
	Home                    *regexp.Regexp
	Images                  *regexp.Regexp
	Object                  *regexp.Regexp
	Query                   *regexp.Regexp
	Shared                  *regexp.Regexp
	Shares                  *regexp.Regexp
	Shareto                 *regexp.Regexp
	Trash                   *regexp.Regexp
	Users                   *regexp.Regexp
	ObjectChangeOwner       *regexp.Regexp
	ObjectExpunge           *regexp.Regexp
	ObjectFavorite          *regexp.Regexp
	ObjectLink              *regexp.Regexp
	ObjectLinks             *regexp.Regexp
	ObjectMove              *regexp.Regexp
	ObjectPermission        *regexp.Regexp
	ObjectProperties        *regexp.Regexp
	Objects                 *regexp.Regexp
	ObjectShare             *regexp.Regexp
	ObjectStream            *regexp.Regexp
	ObjectStreamRevision    *regexp.Regexp
	ObjectSubscription      *regexp.Regexp
	ListObjects             *regexp.Regexp
	ListObjectRevisions     *regexp.Regexp
	ListObjectShares        *regexp.Regexp
	ListObjectSubscriptions *regexp.Regexp
	ListImages              *regexp.Regexp
	TrashObject             *regexp.Regexp
	StatsObject             *regexp.Regexp
	StaticFiles             *regexp.Regexp
}

func (h AppServer) initRegex() *StaticRx {
	return &StaticRx{
		// These regular expressions to match uri patterns
		Favorites:               initRegex(h.ServicePrefix + "/favorites$"),
		Folder:                  initRegex(h.ServicePrefix + "/folder$"),
		Home:                    initRegex(h.ServicePrefix + "/?$"),
		Images:                  initRegex(h.ServicePrefix + "/images$"),
		Object:                  initRegex(h.ServicePrefix + "/object$"),
		Query:                   initRegex(h.ServicePrefix + "/query/.*"),
		Shared:                  initRegex(h.ServicePrefix + "/shared$"),
		Shares:                  initRegex(h.ServicePrefix + "/shares$"),
		Shareto:                 initRegex(h.ServicePrefix + "/shareto$"),
		Trash:                   initRegex(h.ServicePrefix + "/trash$"),
		Users:                   initRegex(h.ServicePrefix + "/users$"),
		ObjectChangeOwner:       initRegex(h.ServicePrefix + "/object/.*/changeowner/.*"),
		ObjectExpunge:           initRegex(h.ServicePrefix + "/object/.*/expunge$"),
		ObjectFavorite:          initRegex(h.ServicePrefix + "/object/.*/favorite$"),
		ObjectLink:              initRegex(h.ServicePrefix + "/object/.*/link/.*"),
		ObjectLinks:             initRegex(h.ServicePrefix + "/object/.*/links$"),
		ObjectMove:              initRegex(h.ServicePrefix + "/object/.*/move/.*"),
		ObjectPermission:        initRegex(h.ServicePrefix + "/object/.*/permission/.*"),
		ObjectProperties:        initRegex(h.ServicePrefix + "/object/.*/properties$"),
		Objects:                 initRegex(h.ServicePrefix + "/objects$"),
		ObjectShare:             initRegex(h.ServicePrefix + "/object/.*/share$"),
		ObjectStream:            initRegex(h.ServicePrefix + "/object/.*/stream$"),
		ObjectStreamRevision:    initRegex(h.ServicePrefix + "/object/.*/history/.*/stream$"),
		ObjectSubscription:      initRegex(h.ServicePrefix + "/object/.*/subscribe$"),
		ListObjects:             initRegex(h.ServicePrefix + "/object/.*/list$"),
		ListObjectRevisions:     initRegex(h.ServicePrefix + "/object/.*/history$"),
		ListObjectShares:        initRegex(h.ServicePrefix + "/object/.*/shares$"),
		ListObjectSubscriptions: initRegex(h.ServicePrefix + "/object/.*/subscriptions$"),
		ListImages:              initRegex(h.ServicePrefix + "/images/.*/list$"),
		TrashObject:             initRegex(h.ServicePrefix + "/trash/.*"),
		StatsObject:             initRegex(h.ServicePrefix + "/stats$"),
		StaticFiles:             initRegex(h.ServicePrefix + "/static/(?P<path>.*)"),
	}
}

// Store this globally for now.  It could go into h.
var rx *StaticRx

// ServeHTTP handles the routing of requests
func (h AppServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	caller := GetCaller(r)

	// Load user from database, adding if they dont exist
	var user *models.ODUser
	var userRequested models.ODUser
	userRequested.DistinguishedName = caller.DistinguishedName
	user, err := h.DAO.GetUserByDistinguishedName(&userRequested)
	if err != nil || user.DistinguishedName != caller.DistinguishedName {
		// log.Printf("User was not found in database: %s", err.Error())
		// if err == sql.ErrNoRows || user.DistinguishedName != caller.DistinguishedName {
		// Doesn't exist yet, lets add this user
		userRequested.DistinguishedName = caller.DistinguishedName
		userRequested.DisplayName.String = caller.CommonName
		userRequested.DisplayName.Valid = true
		userRequested.CreatedBy = caller.DistinguishedName
		user, err = h.DAO.CreateUser(&userRequested)
		if err != nil {
			log.Printf("%s does not exist in database. Error creating: %s", caller.DistinguishedName, err.Error())
			http.Error(w, "Error accesing resource", 500)
			return
		}
		// } else {
		// 	log.Printf(err.Error())
		// }
	}

	if len(user.ModifiedBy) == 0 {
		fmt.Println("User does not have modified by set!")
		jsonData, err := json.MarshalIndent(user, "", "  ")
		if err != nil {
			panic(err)
		}
		jsonified := string(jsonData)
		fmt.Println(jsonified)
	}

	var uri = r.URL.Path

	log.Println("LOGGING APP SERVER CONFIG:")
	log.Println(h.ServicePrefix)
	log.Println("LOGGING URI: ")
	log.Println(r.Method, uri)

	//This will only compile the regexes once
	if rx == nil {
		rx = h.initRegex()
	}

	// TODO: use StripPrefix in handler?
	// https://golang.org/pkg/net/http/#StripPrefix
	switch r.Method {
	case "GET":
		switch {
		case rx.Home.MatchString(uri):
			h.home(w, r, caller)
		case uri == h.ServicePrefix+"/favicon.ico", uri == h.ServicePrefix+"//favicon.ico":
			h.favicon(w, r)
			// from longest to shortest...
		case rx.ObjectStreamRevision.MatchString(uri):
			h.getObjectStreamForRevision(w, r, caller)
		case rx.ObjectStream.MatchString(uri):
			h.getObjectStream(w, r, caller)
		case rx.ObjectProperties.MatchString(uri):
			h.getObject(w, r, caller)
		case rx.ObjectLinks.MatchString(uri):
			h.getRelationships(w, r, caller)
		case rx.Objects.MatchString(uri):
			h.listObjects(w, r, caller)
		case rx.ListObjects.MatchString(uri):
			h.listObjects(w, r, caller)
		case rx.Images.MatchString(uri), rx.ListImages.MatchString(uri):
			h.listObjectsImages(w, r, caller)
		case rx.ListObjectRevisions.MatchString(uri):
			h.listObjectRevisions(w, r, caller)
		case rx.ListObjectShares.MatchString(uri):
			h.listObjectShares(w, r, caller)
		case rx.ListObjectSubscriptions.MatchString(uri):
			h.listObjectsSubscriptions(w, r, caller)
			// single quick matchers
		case rx.Favorites.MatchString(uri):
			h.listFavorites(w, r, caller)
		case rx.Shared.MatchString(uri):
			h.listUserObjectsShared(w, r, caller)
		case rx.Shares.MatchString(uri):
			h.listUserObjectShares(w, r, caller)
			// TODO: Find out why this is showing up for /object//list
		case rx.Object.MatchString(uri):
			h.createObject(w, r, caller)
		case rx.Trash.MatchString(uri):
			h.listObjectsTrashed(w, r, caller)
		case rx.Query.MatchString(uri):
			h.query(w, r, caller)
		case rx.StatsObject.MatchString(uri):
			h.getStats(w, r, caller)
		case rx.StaticFiles.MatchString(uri):
			h.serveStatic(w, r, rx.StaticFiles, uri)
		case rx.Users.MatchString(uri):
			h.listUsers(w, r, caller)
		default:
			jurl, _ := json.MarshalIndent(r.URL, "", "  ")
			fmt.Println(string(jurl))

			msg := caller.DistinguishedName + " from address " + r.RemoteAddr + " using " + r.UserAgent() + " unhandled operation " + r.Method + " " + uri
			log.Println("WARN: " + msg)
			http.Error(w, "Resource not found", 404)
		}
	case "POST":
		switch {
		case rx.ObjectShare.MatchString(uri):
			h.addObjectShare(w, r, caller)
		case rx.ObjectSubscription.MatchString(uri):
			h.addObjectSubscription(w, r, caller)
		case rx.ObjectFavorite.MatchString(uri):
			h.addObjectToFavorites(w, r, caller)
		case rx.ObjectLink.MatchString(uri):
			h.addObjectToFolder(w, r, caller)
		case rx.Objects.MatchString(uri):
			log.Println("POST list objects")
			h.listObjects(w, r, caller)
		case rx.Folder.MatchString(uri):
			h.createFolder(w, r, caller)
		case rx.Object.MatchString(uri):
			h.createObject(w, r, caller)
		case rx.ListObjects.MatchString(uri):
			h.listObjects(w, r, caller)
		case rx.Query.MatchString(uri):
			h.query(w, r, caller)
		case rx.ObjectStream.MatchString(uri):
			h.updateObjectStream(w, r, caller)
		case rx.ObjectShare.MatchString(uri):
			h.addObjectShare(w, r, caller)
		//XXX This is a case that's necessary to have a very basic UI without JS.
		//Parameters hiding in POST urls is a problem without Javascript
		//But for now... we still need a UI to go with automated
		//testing (which does work fine)
		case rx.Shareto.MatchString(uri):
			h.addObjectShare(w, r, caller)
		default:
			msg := caller.DistinguishedName + " from address " + r.RemoteAddr + " using " + r.UserAgent() + " unhandled operation " + r.Method + " " + uri
			log.Println("WARN: " + msg)
			http.Error(w, "Resource not found", 404)
		}
	case "PUT":
		switch {
		case rx.ObjectChangeOwner.MatchString(uri):
			h.changeOwner(w, r, caller)
		case rx.ObjectMove.MatchString(uri):
			h.moveObject(w, r, caller)
		case rx.ObjectPermission.MatchString(uri):
			h.updateObjectPermissions(w, r, caller)
		case rx.ObjectProperties.MatchString(uri):
			h.updateObject(w, r, caller)
		default:
			msg := caller.DistinguishedName + " from address " + r.RemoteAddr + " using " + r.UserAgent() + " unhandled operation " + r.Method + " " + uri
			log.Println("WARN: " + msg)
			http.Error(w, "Resource not found", 404)
		}
	case "DELETE":
		switch {
		case rx.Object.MatchString(uri):
			h.deleteObject(w, r, caller)
		case rx.ObjectExpunge.MatchString(uri):
			h.deleteObjectForever(w, r, caller)
		case rx.ObjectFavorite.MatchString(uri):
			h.removeObjectFromFavorites(w, r, caller)
		case rx.ObjectLink.MatchString(uri):
			h.removeObjectFromFolder(w, r, caller)
		case rx.TrashObject.MatchString(uri):
			h.removeObjectFromTrash(w, r, caller)
		case rx.ObjectShare.MatchString(uri):
			h.removeObjectShare(w, r, caller)
		case rx.ObjectSubscription.MatchString(uri):
			h.removeObjectSubscription(w, r, caller)
		default:
			msg := caller.DistinguishedName + " from address " + r.RemoteAddr + " using " + r.UserAgent() + " unhandled operation " + r.Method + " " + uri
			log.Println("WARN: " + msg)
			http.Error(w, "Resource not found", 404)
		}
	}
}

// GetCaller populates a Caller object based upon request headers and peer
// certificates. Logically this is intended to work with or without NGINX as
// a front end
func GetCaller(r *http.Request) Caller {
	var localDebug = false
	var caller Caller
	caller.UserDistinguishedName = r.Header.Get("USER_DN")
	caller.ExternalSystemDistinguishedName = r.Header.Get("EXTERNAL_SYS_DN")
	if caller.UserDistinguishedName != "" {
		if localDebug {
			log.Println("Assigning distinguished name from USER_DN")
		}
		caller.DistinguishedName = caller.UserDistinguishedName
	} else {
		if len(r.TLS.PeerCertificates) > 0 {
			if localDebug {
				log.Println("Assigning distinguished name from peer certificate")
			}
			caller.DistinguishedName = config.GetDistinguishedName(r.TLS.PeerCertificates[0])
		} else {
			if localDebug {
				log.Println("WARNING: No distinguished name set!!!")
			}
		}
	}
	caller.DistinguishedName = config.GetNormalizedDistinguishedName(caller.DistinguishedName)
	caller.CommonName = config.GetCommonName(caller.DistinguishedName)
	return caller
}

func (h AppServer) xcheckAccess(dn string, clasKey string) bool {
	if h.AAC == nil {
		log.Printf("no aac checks for now")
		return true
	}
	//XXX XXX hack until I can reliably lookup real dns from whatever environment
	//i work on.  This is enough to at least exercise that the API works,
	//and will still work if it comes from a header
	//	dn = "CN=Holmes Jonathan,OU=People,OU=Bedrock,OU=Six 3 Systems,O=U.S. Government,C=US"
	//clasKey = "S"

	tokenType := "pki_dias"
	acmComplete := h.Classifications[clasKey]
	resp, err := h.AAC.CheckAccess(dn, tokenType, acmComplete)

	if err != nil {
		log.Printf("Error calling CheckAccess(): %v \n", err)
	}

	if resp.Success != true {
		log.Printf("Expected true, got %v \n", resp.Success)
		return false
	}

	if !resp.HasAccess {
		log.Printf("Expected resp.HasAccess to be true\n")
	}
	return true
}
