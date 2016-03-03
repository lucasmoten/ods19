package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/context"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/performance"
	aac "decipher.com/oduploader/services/aac"
	audit "decipher.com/oduploader/services/audit/generated/auditservice_thrift"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Constants serve as keys for setting values on a request-scoped Context.
const (
	CallerVal = iota
)

// AppServer contains definition for the metadata server
type AppServer struct {
	// Port is the TCP port that the web server listens on
	Port int
	// Bind is the Network Address that the web server will use.
	Bind string
	// Addr is the combined network address and port the server listens on
	Addr string
	// DAO is the interface contract with the database.
	DAO dao.DAO
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
	AAC aac.AacService
	// Audit Service is for remote logging for compliance.
	Auditer audit.AuditService
	// MasterKey is the secret passphrase used in scrambling keys
	MasterKey string
	// Tracker captures metrics about upload/download begin and end time and transfer bytes
	Tracker *performance.JobReporters
	// TemplateCache is location of HTML templates used by server
	TemplateCache *template.Template
	// StaticDir is location of static objects like javascript
	StaticDir string
	// Routes holds the routes.
	Routes *StaticRx
}

// At points where a goroutine originating from a ServerHTTP call
// must stop and issue an error to the client and stop any further information in the
// connection.  This AppError is *not* recoverable in any way, because the connection
// is already considered dead at this point.  At best, intermediate handlers may need
// to handle surrounding cleanup that wasn't already done with a defer.
//
//  If we are not in the top level handler, we should always just stop and quietly throw
//  the error up:
//
//    if a,b,herr,err := h.acceptUpload(......); herr != nil {
//      return herr
//    }
//
//  And the top level ServeHTTP (or as high as possible) needs to handle it for real, and stop.
//
//     if herr != nil {
//         h.sendError(herr.Code, herr.Err, herr.Msg)
//         return //DO NOT RECOVER.  THE HTTP ERROR CODES HAVE BEEN SENT!
//     }
//
type AppError struct {
	Code int    //the http error code to return with the msg
	Err  error  //an error that is ONLY for the log.  showing to the user may be sensitive.
	Msg  string //message to show to the user, and in log
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
	HomeListObjects         *regexp.Regexp
	Images                  *regexp.Regexp
	ObjectCreate            *regexp.Regexp
	Query                   *regexp.Regexp
	Shared                  *regexp.Regexp
	Shares                  *regexp.Regexp
	Shareto                 *regexp.Regexp
	Trash                   *regexp.Regexp
	Users                   *regexp.Regexp
	Object                  *regexp.Regexp
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

// InitRegex compiles static regexes and initializes the AppServer Routes field.
func (h *AppServer) InitRegex() {
	h.Routes = &StaticRx{
		// These regular expressions to match uri patterns
		Favorites:               initRegex(h.ServicePrefix + "/favorites$"),
		Folder:                  initRegex(h.ServicePrefix + "/folder$"),
		Home:                    initRegex(h.ServicePrefix + "/?$"),
		HomeListObjects:         initRegex(h.ServicePrefix + "/home/listObjects/?$"),
		Images:                  initRegex(h.ServicePrefix + "/images$"),
		ObjectCreate:            initRegex(h.ServicePrefix + "/object$"),
		Query:                   initRegex(h.ServicePrefix + "/query/.*"),
		Shared:                  initRegex(h.ServicePrefix + "/shared$"),
		Shares:                  initRegex(h.ServicePrefix + "/shares$"),
		Trash:                   initRegex(h.ServicePrefix + "/trash$"),
		Users:                   initRegex(h.ServicePrefix + "/users$"),
		Object:                  initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)$"),
		ObjectChangeOwner:       initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/changeowner/.*"),
		ObjectExpunge:           initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/expunge$"),
		ObjectFavorite:          initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/favorite$"),
		ObjectLink:              initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/link/([0-9a-fA-F]*)"),
		ObjectLinks:             initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/links$"),
		ObjectMove:              initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/move/([0-9a-fA-F]*)"),
		ObjectPermission:        initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/permission/([0-9a-fA-F]*)"),
		ObjectProperties:        initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/properties$"),
		Objects:                 initRegex(h.ServicePrefix + "/objects$"),
		ObjectShare:             initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/share$"),
		ObjectStream:            initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/stream$"),
		ObjectStreamRevision:    initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/history/.*/stream$"),
		ObjectSubscription:      initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/subscribe$"),
		ListObjects:             initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/list$"),
		ListObjectRevisions:     initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/history$"),
		ListObjectShares:        initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/shares$"),
		ListObjectSubscriptions: initRegex(h.ServicePrefix + "/object/([0-9a-fA-F]*)/subscriptions$"),
		ListImages:              initRegex(h.ServicePrefix + "/images/([0-9a-fA-F]*)/list$"),
		TrashObject:             initRegex(h.ServicePrefix + "/trash/([0-9a-fA-F]*)"),
		StatsObject:             initRegex(h.ServicePrefix + "/stats$"),
		StaticFiles:             initRegex(h.ServicePrefix + "/static/(?P<path>.*)"),
	}
}

// ServeHTTP handles the routing of requests
func (h AppServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	caller := GetCaller(r)

	// Prepare a Context to propagate to request handlers
	var ctx context.Context

	// Load user from database, adding if they dont exist
	var user models.ODUser
	var userRequested models.ODUser
	userRequested.DistinguishedName = caller.DistinguishedName
	user, err := h.DAO.GetUserByDistinguishedName(userRequested)
	if err != nil || user.DistinguishedName != caller.DistinguishedName {
		log.Printf("User was not found in database: %s", err.Error())
		userRequested.DistinguishedName = caller.DistinguishedName
		userRequested.DisplayName.String = caller.CommonName
		userRequested.DisplayName.Valid = true
		userRequested.CreatedBy = caller.DistinguishedName
		user, err = h.DAO.CreateUser(userRequested)
		if err != nil {
			log.Printf("%s does not exist in database. Error creating: %s", caller.DistinguishedName, err.Error())
			http.Error(w, "Error accesing resource", 500)
			return
		}
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

	// Set the caller as a value on the Context. Background() creates a new context.
	// Subsequent calls should pass the same ctx instead of initiliazing a new context.
	ctx = ContextWithCaller(context.Background(), caller)

	var uri = r.URL.Path

	log.Printf("LOGGING APP SERVER CONFIG:%s URI:%s %s BY USER:%s", h.ServicePrefix, r.Method, uri, user.DistinguishedName)

	// TODO: use StripPrefix in handler?
	// https://golang.org/pkg/net/http/#StripPrefix
	switch r.Method {
	case "GET":
		switch {
		case h.Routes.Home.MatchString(uri):
			h.home(w, r, caller)
		case h.Routes.HomeListObjects.MatchString(uri):
			h.homeListObjects(w, r, caller)
		case uri == h.ServicePrefix+"/favicon.ico", uri == h.ServicePrefix+"//favicon.ico", strings.HasSuffix(uri, "/favicon.ico"):
			h.favicon(w, r)
			// from longest to shortest...
		case h.Routes.ObjectStreamRevision.MatchString(uri):
			h.getObjectStreamForRevision(w, r, caller)
		case h.Routes.ObjectStream.MatchString(uri):
			h.getObjectStream(w, r, caller)
		case h.Routes.ObjectProperties.MatchString(uri):
			h.getObject(w, r, caller)
		case h.Routes.ObjectLinks.MatchString(uri):
			h.getRelationships(w, r, caller)
		case h.Routes.Objects.MatchString(uri):
			h.listObjects(w, r, caller)
		case h.Routes.ListObjects.MatchString(uri):
			h.listObjects(w, r, caller)
		case h.Routes.Images.MatchString(uri), h.Routes.ListImages.MatchString(uri):
			h.listObjectsImages(w, r, caller)
		case h.Routes.ListObjectRevisions.MatchString(uri):
			h.listObjectRevisions(w, r, caller)
		case h.Routes.ListObjectShares.MatchString(uri):
			h.listObjectShares(ctx, w, r)
		case h.Routes.ListObjectSubscriptions.MatchString(uri):
			h.listObjectsSubscriptions(w, r, caller)
			// single quick matchers
		case h.Routes.Favorites.MatchString(uri):
			h.listFavorites(w, r, caller)
		case h.Routes.Shared.MatchString(uri):
			h.listUserObjectsShared(ctx, w, r)
		case h.Routes.Shares.MatchString(uri):
			h.listUserObjectShares(w, r, caller)
		case h.Routes.Trash.MatchString(uri):
			h.listObjectsTrashed(ctx, w, r)
		case h.Routes.Query.MatchString(uri):
			h.query(w, r, caller)
		case h.Routes.StatsObject.MatchString(uri):
			h.getStats(w, r, caller)
		case h.Routes.StaticFiles.MatchString(uri):
			h.serveStatic(w, r, h.Routes.StaticFiles, uri)
		case h.Routes.Users.MatchString(uri):
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
		case h.Routes.ObjectShare.MatchString(uri):
			h.addObjectShare(w, r, caller)
		case h.Routes.ObjectSubscription.MatchString(uri):
			h.addObjectSubscription(w, r, caller)
		case h.Routes.ObjectFavorite.MatchString(uri):
			h.addObjectToFavorites(w, r, caller)
		case h.Routes.ObjectLink.MatchString(uri):
			h.addObjectToFolder(w, r, caller)
		case h.Routes.Objects.MatchString(uri):
			log.Println("POST list objects")
			h.listObjects(w, r, caller)
		case h.Routes.Folder.MatchString(uri):
			h.createFolder(w, r, caller)
		case h.Routes.ObjectCreate.MatchString(uri):
			h.createObject(w, r, caller)
		case h.Routes.ListObjects.MatchString(uri):
			h.listObjects(w, r, caller)
		case h.Routes.Query.MatchString(uri):
			h.query(w, r, caller)
		case h.Routes.ObjectStream.MatchString(uri):
			h.updateObjectStream(w, r, caller)
		default:
			msg := caller.DistinguishedName + " from address " + r.RemoteAddr + " using " + r.UserAgent() + " unhandled operation " + r.Method + " " + uri
			log.Println("WARN: " + msg)
			http.Error(w, "Resource not found", 404)
		}
	case "PUT":
		switch {
		case h.Routes.ObjectChangeOwner.MatchString(uri):
			h.changeOwner(w, r, caller)
		case h.Routes.ObjectMove.MatchString(uri):
			h.moveObject(w, r, caller)
		case h.Routes.ObjectPermission.MatchString(uri):
			h.updateObjectPermissions(w, r, caller)
		case h.Routes.ObjectProperties.MatchString(uri):
			h.updateObject(w, r, caller)
		default:
			msg := caller.DistinguishedName + " from address " + r.RemoteAddr + " using " + r.UserAgent() + " unhandled operation " + r.Method + " " + uri
			log.Println("WARN: " + msg)
			http.Error(w, "Resource not found", 404)
		}
	case "DELETE":
		switch {
		case h.Routes.Object.MatchString(uri):
			h.deleteObject(w, r, caller)
		case h.Routes.ObjectExpunge.MatchString(uri):
			h.deleteObjectForever(w, r, caller)
		case h.Routes.ObjectFavorite.MatchString(uri):
			h.removeObjectFromFavorites(w, r, caller)
		case h.Routes.ObjectLink.MatchString(uri):
			h.removeObjectFromFolder(w, r, caller)
		case h.Routes.TrashObject.MatchString(uri):
			h.removeObjectFromTrash(w, r, caller)
		case h.Routes.ObjectShare.MatchString(uri):
			h.removeObjectShare(w, r, caller)
		case h.Routes.ObjectSubscription.MatchString(uri):
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

// ContextWithCaller returns a new Context object with a Caller value set. The const CallerVal acts
// as the key that maps to the caller value.
func ContextWithCaller(ctx context.Context, caller Caller) context.Context {
	return context.WithValue(ctx, CallerVal, caller)
}

// CallerFromContext extracts a Caller from a context, if set.
func CallerFromContext(ctx context.Context) (Caller, bool) {
	// ctx.Value returns nil if ctx has no value for the key
	// the Caller type assertion returns ok=false for nil.
	caller, ok := ctx.Value(CallerVal).(Caller)
	return caller, ok
}
