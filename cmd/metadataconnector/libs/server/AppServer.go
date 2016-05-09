package server

import (
	"html/template"
	"log"
	"net/http"
	"regexp"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/dao"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
	"decipher.com/object-drive-server/performance"
	"decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/services/audit"
	"decipher.com/object-drive-server/services/zookeeper"
	"decipher.com/object-drive-server/util"
)

// Constants serve as keys for setting values on a request-scoped Context.
const (
	CallerVal        = iota
	CaptureGroupsVal = iota
	UserVal          = iota
	UserSnippetsVal  = iota
)

// AppServer contains definition for the metadata server
type AppServer struct {
	// Port is the TCP port that the web server listens on
	Port string
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
	Auditor audit.Auditor
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
	// Users contains a cache of users
	Users *UserCache
	// Snippets contains a cache of snippets
	Snippets *SnippetCache
	// AclWhitelist provides a list of distinguished names allowed to perform impersonation
	AclImpersonationWhitelist []string
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
		APIDocumentation: regexp.MustCompile(h.ServicePrefix + "/$"),
		UserStats:        regexp.MustCompile(h.ServicePrefix + "/userstats$"),
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
	err := caller.ValidateHeaders(h.AclImpersonationWhitelist, w, r)
	// Log state consistent with AclRestFilter
	if err != nil {
		log.Printf("Transaction: "+caller.TransactionType+" INVALID!"+caller.GetMessage()+" %s", err.Error())
		sendErrorResponse(&w, 401, err, err.Error())
		return
	}
	log.Printf("Transaction: " + caller.TransactionType + " VALID! UserAuthentication.current: " + caller.UserDistinguishedName + " " + caller.GetMessage())

	// Prepare a Context to propagate to request handlers
	var ctx context.Context

	// Set the caller as a value on the Context. Background() creates a new context.
	// Subsequent calls should pass the same ctx instead of initiliazing a new context.
	ctx = ContextWithCaller(context.Background(), caller)

	// For routing, examine just the path portion of the URL
	var uri = r.URL.Path

	// Log the request. This is predominately diagnostic, as full logging of the
	// request with status codes and sizing is done in nginx fronting this service
	log.Printf("URI:%s %s USER:%s", r.Method, uri, caller.DistinguishedName)

	// The following routes can be handled without calls to the database
	switch r.Method {
	case "GET":
		switch {
		// Development UI
		case h.Routes.Home.MatchString(uri):
			h.home(ctx, w, r)
			return
		case h.Routes.HomeListObjects.MatchString(uri):
			h.homeListObjects(ctx, w, r)
			return
		case h.Routes.Favicon.MatchString(uri):
			h.favicon(ctx, w, r)
			return
		case h.Routes.StatsObject.MatchString(uri):
			h.getStats(ctx, w, r)
			return
		case h.Routes.StaticFiles.MatchString(uri):
			h.serveStatic(w, r, h.Routes.StaticFiles, uri)
			return
		// API
		// - documentation
		case h.Routes.APIDocumentation.MatchString(uri):
			h.docs(ctx, w, r)
			return
		}
	}

	// Retrieve user
	user, err := h.FetchUser(ctx)
	if err != nil {
		sendErrorResponse(&w, 500, nil, "Error loading user")
		return
	}
	// And put on context
	ctx = PutUserOnContext(ctx, *user)

	// TODO: use StripPrefix in handler?
	// https://golang.org/pkg/net/http/#StripPrefix
	switch r.Method {
	case "GET":
		switch {
		// Development UI
		case h.Routes.Users.MatchString(uri):
			h.listUsers(ctx, w, r)
			// API
			// - user profile usage information
		case h.Routes.UserStats.MatchString(uri):
			h.userStats(ctx, w, r)
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
	// TODO: Before returning, lets capture changes placed back on the context and push into the cache
	// TODO: UserSnippetsFromContext
	// TODO: UserSnippetSQL

	// TODO: Before returning, finalize any metrics, capturing time/error codes ?
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

// PutUserOnContext puts the user object on the context and returns the modified context
func PutUserOnContext(ctx context.Context, user models.ODUser) context.Context {
	return context.WithValue(ctx, UserVal, user)
}

// UserFromContext extracts the user from a context, if set
func UserFromContext(ctx context.Context) (models.ODUser, bool) {
	user, ok := ctx.Value(UserVal).(models.ODUser)
	return user, ok
}

// PutUserSnippetsOnContext puts the user snippet object on the context and returns the modified context
func PutUserSnippetsOnContext(ctx context.Context, snippets acm.ODriveRawSnippetFields) context.Context {
	return context.WithValue(ctx, UserSnippetsVal, snippets)
}

// UserSnippetsFromContext extracts the user snippets from a context, if set
func UserSnippetsFromContext(ctx context.Context) (acm.ODriveRawSnippetFields, bool) {
	snippets, ok := ctx.Value(UserSnippetsVal).(acm.ODriveRawSnippetFields)
	return snippets, ok
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

// StaticRx statically references compiled regular expressions.
type StaticRx struct {
	Home                   *regexp.Regexp
	HomeListObjects        *regexp.Regexp
	Favicon                *regexp.Regexp
	StatsObject            *regexp.Regexp
	StaticFiles            *regexp.Regexp
	Users                  *regexp.Regexp
	APIDocumentation       *regexp.Regexp
	UserStats              *regexp.Regexp
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
