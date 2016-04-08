package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"

	"golang.org/x/net/context"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/performance"
	aac "decipher.com/oduploader/services/aac"
	audit "decipher.com/oduploader/services/audit/generated/auditservice_thrift"
	"decipher.com/oduploader/services/zookeeper"
	"decipher.com/oduploader/util"
)

// Constants serve as keys for setting values on a request-scoped Context.
const (
	CallerVal        = iota
	CaptureGroupsVal = iota
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
	// This encapsulates connectivity to long term storage behind the cache
	DrainProvider DrainProvider
	// ZKState is the current state of zookeeper
	ZKState zookeeper.ZKState
}

// AppError encapsulates an error with a desired http status code so that the server
// can issue the error code to the client.
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
	Code  int    //the http error code to return with the msg
	Error error  //an error that is ONLY for the log.  showing to the user may be sensitive.
	Msg   string //message to show to the user, and in log
	File  string //origin file
	Line  int    //origin line
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

// StaticRx is a bunch of static compiled regexes
type StaticRx struct {
	Home                   *regexp.Regexp
	HomeListObjects        *regexp.Regexp
	Favicon                *regexp.Regexp
	StatsObject            *regexp.Regexp
	StaticFiles            *regexp.Regexp
	Users                  *regexp.Regexp
	APIDocumentation       *regexp.Regexp
	Objects                *regexp.Regexp
	Object                 *regexp.Regexp
	ObjectProperties       *regexp.Regexp
	ObjectStream           *regexp.Regexp
	ObjectChangeOwner      *regexp.Regexp
	ObjectDelete           *regexp.Regexp
	ObjectUndelete         *regexp.Regexp
	ObjectExpunge          *regexp.Regexp
	ObjectMove             *regexp.Regexp
	Revisions              *regexp.Regexp
	RevisionStream         *regexp.Regexp
	SharedToMe             *regexp.Regexp
	SharedToOthers         *regexp.Regexp
	SharedObject           *regexp.Regexp
	SharedObjectShare      *regexp.Regexp
	Search                 *regexp.Regexp
	Trash                  *regexp.Regexp
	Favorites              *regexp.Regexp
	FavoriteObject         *regexp.Regexp
	LinkToObject           *regexp.Regexp
	ObjectLinks            *regexp.Regexp
	ObjectSubscribe        *regexp.Regexp
	Subscribed             *regexp.Regexp
	SubscribedSubscription *regexp.Regexp
	ObjectTypes            *regexp.Regexp
	ObjectType             *regexp.Regexp
}

// InitRegex compiles static regexes and initializes the AppServer Routes field.
func (h *AppServer) InitRegex() {
	h.Routes = &StaticRx{
		// UI
		Home:            regexp.MustCompile(h.ServicePrefix + "/ui/?$"),
		HomeListObjects: regexp.MustCompile(h.ServicePrefix + "/ui/listObjects/?$"),
		Favicon:         regexp.MustCompile(h.ServicePrefix + "/favicon.ico$"),
		StatsObject:     regexp.MustCompile(h.ServicePrefix + "/stats$"),
		StaticFiles:     regexp.MustCompile(h.ServicePrefix + "/static/(?P<path>.*)"),
		Users:           regexp.MustCompile(h.ServicePrefix + "/users$"),
		// Service operations
		APIDocumentation: regexp.MustCompile(h.ServicePrefix + "/?$"),
		// - objects
		Objects:          regexp.MustCompile(h.ServicePrefix + "/objects$"),
		Object:           regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})$"),
		ObjectProperties: regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})/properties$"),
		ObjectStream:     regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})/stream$"),
		// - actions on objects
		ObjectChangeOwner: regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})/owner/(?P<newOwner>.*)$"),
		ObjectDelete:      regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})/trash$"),
		ObjectUndelete:    regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})/untrash$"),
		ObjectExpunge:     regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})$"),
		ObjectMove:        regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})/move/(?P<folderId>[0-9a-fA-F]{32})$"),
		// - revisions
		Revisions:      regexp.MustCompile(h.ServicePrefix + "/revisions/(?P<objectId>[0-9a-fA-F]{32})$"),
		RevisionStream: regexp.MustCompile(h.ServicePrefix + "/revisions/(?P<objectId>[0-9a-fA-F]{32})/(?P<revisionId>.*)/stream$"),
		// - share
		SharedToMe:        regexp.MustCompile(h.ServicePrefix + "/shares$"),
		SharedToOthers:    regexp.MustCompile(h.ServicePrefix + "/shared$"),
		SharedObject:      regexp.MustCompile(h.ServicePrefix + "/shared/(?P<objectId>[0-9a-fA-F]{32})$"),
		SharedObjectShare: regexp.MustCompile(h.ServicePrefix + "/shared/(?P<objectId>[0-9a-fA-F]{32})/(?P<shareId>[0-9a-fA-F]{32})$"),
		// - search
		Search: regexp.MustCompile(h.ServicePrefix + "/search/(?P<searchPhrase>.*)$"),
		// - trash
		Trash: regexp.MustCompile(h.ServicePrefix + "/trashed$"),
		// - not yet implemented. future
		Favorites:              regexp.MustCompile(h.ServicePrefix + "/favorites$"),
		FavoriteObject:         regexp.MustCompile(h.ServicePrefix + "/favorites/(?P<objectId>[0-9a-fA-F]{32})$"),
		LinkToObject:           regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})/links/(?P<sourceObjectId>[0-9a-fA-F]{32})$"),
		ObjectLinks:            regexp.MustCompile(h.ServicePrefix + "/links/(?P<objectId>[0-9a-fA-F]{32})$"),
		ObjectSubscribe:        regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})/subscribe$"),
		Subscribed:             regexp.MustCompile(h.ServicePrefix + "/subscribed$"),
		SubscribedSubscription: regexp.MustCompile(h.ServicePrefix + "/subscribed/(?P<subscriptionId>[0-9a-fA-F]{32})$"),
		ObjectTypes:            regexp.MustCompile(h.ServicePrefix + "/objecttypes$"),
		ObjectType:             regexp.MustCompile(h.ServicePrefix + "/objecttypes/(?P<objectTypeId>[0-9a-fA-F]{32})$"),
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
		log.Printf("Creating user in database: %s", err.Error())
		userRequested.DistinguishedName = caller.DistinguishedName
		userRequested.DisplayName.String = caller.CommonName
		userRequested.DisplayName.Valid = true
		userRequested.CreatedBy = caller.DistinguishedName
		user, err = h.DAO.CreateUser(userRequested)
		if err != nil {
			log.Printf("%s does not exist in database. Error creating: %s", caller.DistinguishedName, err.Error())
			sendErrorResponse(&w, 500, nil, "Error accessing resource")
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

	//log.Printf("LOGGING APP SERVER CONFIG:%s URI:%s %s BY USER:%s", h.ServicePrefix, r.Method, uri, user.DistinguishedName)
	log.Printf("URI:%s %s USER:%s", r.Method, uri, user.DistinguishedName)

	// TODO: use StripPrefix in handler?
	// https://golang.org/pkg/net/http/#StripPrefix
	switch r.Method {
	case "GET":
		switch {
		// Development UI
		case h.Routes.Home.MatchString(uri):
			h.home(ctx, w, r)
		case h.Routes.HomeListObjects.MatchString(uri):
			h.homeListObjects(ctx, w, r)
		case h.Routes.Favicon.MatchString(uri):
			h.favicon(ctx, w, r)
		case h.Routes.StatsObject.MatchString(uri):
			h.getStats(ctx, w, r)
		case h.Routes.StaticFiles.MatchString(uri):
			h.serveStatic(w, r, h.Routes.StaticFiles, uri)
		case h.Routes.Users.MatchString(uri):
			h.listUsers(ctx, w, r)
		// API
		// - documentation
		case h.Routes.APIDocumentation.MatchString(uri):
			// TODO: route into serveStatic?
			h.docs(ctx, w, r)
		// - get object properties
		case h.Routes.ObjectProperties.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectProperties)
			h.getObject(ctx, w, r)
		// - get object stream
		case h.Routes.ObjectStream.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectStream)
			h.getObjectStream(ctx, w, r)
		// - list objects at root
		case h.Routes.Objects.MatchString(uri):
			h.listObjects(ctx, w, r)
		// - list objects of object
		case h.Routes.Object.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.Object)
			h.listObjects(ctx, w, r)
		// - list trash
		case h.Routes.Trash.MatchString(uri):
			h.listObjectsTrashed(ctx, w, r)
		// - list objects shared to me
		case h.Routes.SharedToMe.MatchString(uri):
			h.listUserObjectShares(ctx, w, r)
		// - list objects i've shared with others
		case h.Routes.SharedToOthers.MatchString(uri):
			h.listUserObjectsShared(ctx, w, r)
		// - list object revisions (array of get object properties)
		case h.Routes.Revisions.MatchString(uri):
			h.listObjectRevisions(ctx, w, r)
		// - get object revision stream
		case h.Routes.RevisionStream.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.RevisionStream)
			h.getObjectStreamForRevision(ctx, w, r)
		// - search
		case h.Routes.Search.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.Search)
			h.query(ctx, w, r)
		// FUTURE API, NOT YET IMPLEMENTED
		// - get relationships
		case h.Routes.ObjectLinks.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectLinks)
			h.getRelationships(ctx, w, r)
		// - list favorite / starred objects
		case h.Routes.Favorites.MatchString(uri):
			h.listFavorites(ctx, w, r)
		// - list subscribed objects
		case h.Routes.Subscribed.MatchString(uri):
			h.listObjectsSubscriptions(ctx, w, r)
		// - list object types
		case h.Routes.ObjectTypes.MatchString(uri):
			// TODO: h.listObjectTypes(ctx, w, r)
		default:
			do404(ctx, w, r)
		}
	case "POST":
		// API
		switch {
		// - create object
		case h.Routes.Objects.MatchString(uri):
			h.createObject(ctx, w, r)
		// - delete object (updates state)
		case h.Routes.ObjectDelete.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectDelete)
			h.deleteObject(ctx, w, r)
		// - undelete object (updates state)
		case h.Routes.ObjectUndelete.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectUndelete)
			h.removeObjectFromTrash(ctx, w, r)
		// - update object properties
		case h.Routes.ObjectProperties.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectProperties)
			h.updateObject(ctx, w, r)
		// - update object stream
		case h.Routes.ObjectStream.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectStream)
			h.updateObjectStream(ctx, w, r)
		// - create object share
		case h.Routes.SharedObject.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.SharedObject)
			h.addObjectShare(ctx, w, r)
		// - move object
		case h.Routes.ObjectMove.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectMove)
			h.moveObject(ctx, w, r)
		// - change owner
		case h.Routes.ObjectChangeOwner.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectChangeOwner)
			h.changeOwner(ctx, w, r)
		// - create favorite
		case h.Routes.FavoriteObject.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.FavoriteObject)
			h.addObjectToFavorites(ctx, w, r)
		// - create symbolic link from object to another folder
		case h.Routes.LinkToObject.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.LinkToObject)
			h.addObjectToFolder(ctx, w, r)
		// - create subscriptionId
		case h.Routes.ObjectSubscribe.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectSubscribe)
			h.addObjectSubscription(ctx, w, r)
		// - create object type
		case h.Routes.ObjectTypes.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectTypes)
			// TODO: h.addObjectType(ctx, w, r)
			// - update object type
		case h.Routes.ObjectType.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectType)
			// TODO: h.updateObjectType(ctx, w, r)
		default:
			do404(ctx, w, r)
		}
	case "DELETE":
		switch {
		// - delete object forever
		case h.Routes.ObjectExpunge.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectExpunge)
			h.deleteObjectForever(ctx, w, r)
			// - remove object share
		case h.Routes.SharedObjectShare.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.SharedObjectShare)
			h.removeObjectShare(ctx, w, r)
		// - remove object favorite
		case h.Routes.FavoriteObject.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.FavoriteObject)
			h.removeObjectFromFavorites(ctx, w, r)
		// - remove symbolic link
		case h.Routes.LinkToObject.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.LinkToObject)
			h.removeObjectFromFolder(ctx, w, r)
		// - remove subscription
		case h.Routes.SubscribedSubscription.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.SubscribedSubscription)
			h.removeObjectSubscription(ctx, w, r)
		// - remove all subscriptions
		case h.Routes.Subscribed.MatchString(uri):
			// TODO: h.deleteObjectSubscriptions(ctx, w, r)
			// - empty trash (expunge all in trash)
		case h.Routes.Trash.MatchString(uri):
			// TODO: h.emptyTrash(ctx, w, r)
		// - remove all shares on an object
		case h.Routes.SharedObject.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.SharedObject)
			// TODO: h.removeObjectShares(ctx, w, r)
		// - remove object type
		case h.Routes.ObjectType.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectType)
			// TODO: h.deleteObjectType(ctx, w, r)

		default:
			do404(ctx, w, r)
		}
	default:
		do404(ctx, w, r)
	}
	// TODO: Before returning, finalize any metrics, capturing time/error codes ?
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

func parseCaptureGroups(ctx context.Context, path string, regex *regexp.Regexp) context.Context {
	captured := util.GetRegexCaptureGroups(path, regex)
	return context.WithValue(ctx, CaptureGroupsVal, captured)
}

// CaptureGroupsFromContext extracts the capture groups from a context, if set
func CaptureGroupsFromContext(ctx context.Context) (map[string]string, bool) {
	captured, ok := ctx.Value(CaptureGroupsVal).(map[string]string)
	return captured, ok
}

func do404(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		caller = Caller{DistinguishedName: "UnknownUser"}
	}
	uri := r.URL.Path
	msg := caller.DistinguishedName + " from address " + r.RemoteAddr + " using " + r.UserAgent() + " unhandled operation " + r.Method + " " + uri
	log.Println("WARN: " + msg)
	sendErrorResponse(&w, 404, nil, "Resource not found")
	return
}
