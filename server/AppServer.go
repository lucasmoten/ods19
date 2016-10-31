package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/karlseguin/ccache"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/autoscale"
	globalconfig "decipher.com/object-drive-server/config"
	configx "decipher.com/object-drive-server/configx"
	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
	"decipher.com/object-drive-server/performance"
	"decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/services/audit"
	"decipher.com/object-drive-server/services/zookeeper"
	"decipher.com/object-drive-server/util"
	"golang.org/x/net/context"
)

// Constants serve as keys for setting values on a request-scoped Context.
const (
	CallerVal = iota
	CaptureGroupsVal
	GEMVal
	UserVal
	UserSnippetsVal
	AuditEventVal
	Logger
	SessionID
	DAO
	Groups
	Snippets
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
	RootDAO dao.DAO
	// Conf is the configuration passed to the application
	Conf configx.ServerSettingsConfiguration
	// ServicePrefix is the base RootURL for all public operations of web server
	ServicePrefix string
	// AAC is a handle to the Authorization and Access Control client
	AAC aac.AacService
	// AACZK is a pointer to the cluster where we discover AAC. May be set to DefaultZK.
	AACZK *zookeeper.ZKState
	// Audit Service is for remote logging for compliance.
	Auditor audit.Auditor
	// EventQueue is a Publisher interface we use to publish our main event stream.
	EventQueue events.Publisher
	// EventQueueZK is a pointer to the cluster where we discover Kafka. May be set to DefaultZK.
	EventQueueZK *zookeeper.ZKState
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
	// DefaultZK wraps a connection to the ZK cluster we announce to, and holds state for odrive's registration.
	DefaultZK *zookeeper.ZKState
	// UsersLruCache contains a cache of users with support to purge those least recently used when filling. Up to 1000 users will be retained in memory
	UsersLruCache *ccache.Cache
	// Snippets contains a cache of snippets
	Snippets *SnippetCache
	// AclWhitelist provides a list of distinguished names allowed to perform impersonation
	AclImpersonationWhitelist []string
	// ServiceRegistry is a map of services we depend on that reports on their state.
	ServiceRegistry ServiceStates
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
		ObjectStream:     regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})/stream(\\.[0-9a-zA-Z]*)?$"),
		Ciphertext:       regexp.MustCompile(h.ServicePrefix + "/ciphertext/(?P<selector>[0-9a-zA-Z_]*)?/(?P<rname>[0-9a-fA-F]{64})$"),
		// - actions on objects
		ObjectChangeOwner: regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})/owner/(?P<newOwner>.*)$"),
		ObjectDelete:      regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})/trash$"),
		ObjectUndelete:    regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})/untrash$"),
		ObjectExpunge:     regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})$"),
		ObjectMove:        regexp.MustCompile(h.ServicePrefix + "/objects/(?P<objectId>[0-9a-fA-F]{32})/move/(?P<folderId>[0-9a-fA-F]{32})?$"),
		// - revisions
		Revisions:      regexp.MustCompile(h.ServicePrefix + "/revisions/(?P<objectId>[0-9a-fA-F]{32})$"),
		RevisionStream: regexp.MustCompile(h.ServicePrefix + "/revisions/(?P<objectId>[0-9a-fA-F]{32})/(?P<revisionId>.*)/stream(\\.[0-9a-zA-Z]*)?$"),
		// - share
		SharedToMe:       regexp.MustCompile(h.ServicePrefix + "/shares$"),
		SharedToOthers:   regexp.MustCompile(h.ServicePrefix + "/shared$"),
		SharedToEveryone: regexp.MustCompile(h.ServicePrefix + "/sharedpublic$"),
		SharedObject:     regexp.MustCompile(h.ServicePrefix + "/shared/(?P<objectId>[0-9a-fA-F]{32})$"),
		// - search
		Search: regexp.MustCompile(h.ServicePrefix + "/search/(?P<searchPhrase>.*)$"),
		// - trash
		Trash: regexp.MustCompile(h.ServicePrefix + "/trashed$"),
		Zip:   regexp.MustCompile(h.ServicePrefix + "/zip$"),
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

// newLogger instantiates a logger for our object service with basic fields pre-populated.
func newLogger(logger zap.Logger, sessionID, cn string, r *http.Request) zap.Logger {
	return logger.
		With(zap.String("session", sessionID)).
		With(zap.String("cn", cn)).
		With(zap.String("method", r.Method)).
		With(zap.String("uri", r.RequestURI)).
		With(zap.String("date", fmt.Sprintf("%v", time.Now())))
}

//When there is a panic, all deferred functions get executed.
func logCrashInServeHTTP(logger zap.Logger, w http.ResponseWriter) {
	if r := recover(); r != nil {
		logger.Error("odrive crash", zap.Object("context", r), zap.String("stack", string(debug.Stack())))
		w.WriteHeader(http.StatusInternalServerError)
		//Note: even if follow "let it crash" and explicitly return an error code,
		//we should log this and return a 500 if we plan on doing a system exit on internal 5xx errors.
	}
}

// ServeHTTP handles the routing of requests
func (h AppServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	beginTSInMS := util.NowMS()

	sessionID := newSessionID()
	w.Header().Add("sessionid", sessionID)

	caller := CallerFromRequest(r)
	logger := newLogger(globalconfig.RootLogger, sessionID, caller.CommonName, r)
	defer logCrashInServeHTTP(logger, w)

	gem := globalEventFromRequest(r)

	if err := caller.ValidateHeaders(h.AclImpersonationWhitelist, w, r); err != nil {
		sendErrorResponse(logger, &w, 401, err, err.Error())
		return
	}

	ctx := context.Background()
	ctx = ContextWithLogger(ctx, logger)
	ctx = ContextWithCaller(ctx, caller)
	ctx = ContextWithSession(ctx, sessionID)
	ctx = ContextWithDAO(ctx, h.RootDAO)
	ctx = ContextWithGEM(ctx, gem)

	logger.Info(
		"transaction start",
		zap.String("dn", caller.DistinguishedName),
		zap.String("cn", caller.CommonName),
		zap.String("xdn", caller.ExternalSystemDistinguishedName),
		zap.String("sdn", caller.SSLClientSDistinguishedName),
		zap.String("udn", caller.UserDistinguishedName),
		zap.String("dtime", time.Now().String()),
	)

	var uri = r.URL.Path
	var herr *AppError

	// The following routes can be handled without calls to the database
	withoutDatabase := false
	switch r.Method {
	case "GET":
		switch {
		// Development UI
		case h.Routes.Home.MatchString(uri):
			herr = h.home(ctx, w, r)
			withoutDatabase = true
		case h.Routes.HomeListObjects.MatchString(uri):
			herr = h.homeListObjects(ctx, w, r)
			withoutDatabase = true
		case h.Routes.Favicon.MatchString(uri):
			herr = h.favicon(ctx, w, r)
			withoutDatabase = true
		case h.Routes.StatsObject.MatchString(uri):
			herr = h.getStats(ctx, w, r)
			withoutDatabase = true
		case h.Routes.StaticFiles.MatchString(uri):
			herr = h.serveStatic(ctx, w, r)
			withoutDatabase = true
		// API documentation
		case h.Routes.APIDocumentation.MatchString(uri):
			herr = h.docs(ctx, w, r)
			withoutDatabase = true
		}
	}
	if withoutDatabase {
		if herr != nil {
			sendAppErrorResponse(logger, &w, herr)
		} else {
			countOKResponse(logger)
		}
		return
	}

	user, err := h.FetchUser(ctx)
	if err != nil {
		sendErrorResponse(logger, &w, 500, err, "Error loading user")
		return
	}

	ctx = PutUserOnContext(ctx, *user)
	groups, snippets, err := h.GetUserGroupsAndSnippets(ctx)
	if err != nil {
		sendErrorResponse(logger, &w, 500, err, "Error retrieving user snippets")
		return
	}

	// NOTE: Adding the groups to Caller
	caller.Groups = groups
	ctx = ContextWithCaller(ctx, caller)
	ctx = ContextWithSnippets(ctx, snippets)
	ctx = ContextWithGroups(ctx, groups)

	switch r.Method {
	case "GET":
		switch {
		// Development UI
		case h.Routes.Users.MatchString(uri):
			herr = h.listUsers(ctx, w, r)
			// API
			// - user profile usage information
		case h.Routes.UserStats.MatchString(uri):
			herr = h.userStats(ctx, w, r)
		// - get object properties
		case h.Routes.ObjectProperties.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectProperties)
			herr = h.getObject(ctx, w, r)
		// - get object stream
		case h.Routes.ObjectStream.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectStream)
			herr = h.getObjectStream(ctx, w, r)
		// - get ciphertext
		case h.Routes.Ciphertext.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.Ciphertext)
			herr = h.getCiphertext(ctx, w, r)
		// - list objects at root
		case h.Routes.Objects.MatchString(uri):
			herr = h.listObjects(ctx, w, r)
		// - list objects of object
		case h.Routes.Object.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.Object)
			herr = h.listObjects(ctx, w, r)
		// - list trash
		case h.Routes.Trash.MatchString(uri):
			herr = h.listObjectsTrashed(ctx, w, r)
		// - list objects shared to me
		case h.Routes.SharedToMe.MatchString(uri):
			herr = h.listUserObjectShares(ctx, w, r)
		// - list objects i've shared with others
		case h.Routes.SharedToOthers.MatchString(uri):
			herr = h.listUserObjectsShared(ctx, w, r)
		// - list objects shared to everyone
		case h.Routes.SharedToEveryone.MatchString(uri):
			herr = h.listUserObjectsSharedToEveryone(ctx, w, r)
		// - list object revisions (array of get object properties)
		case h.Routes.Revisions.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.Revisions)
			herr = h.listObjectRevisions(ctx, w, r)
		// - get object revision stream
		case h.Routes.RevisionStream.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.RevisionStream)
			herr = h.getObjectStreamForRevision(ctx, w, r)
		// - search
		case h.Routes.Search.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.Search)
			herr = h.query(ctx, w, r)
		// FUTURE API, NOT YET IMPLEMENTED
		// - get relationships
		case h.Routes.ObjectLinks.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectLinks)
			herr = h.getRelationships(ctx, w, r)
		// - list favorite / starred objects
		case h.Routes.Favorites.MatchString(uri):
			herr = h.listFavorites(ctx, w, r)
		// - list subscribed objects
		case h.Routes.Subscribed.MatchString(uri):
			herr = h.listObjectsSubscriptions(ctx, w, r)
		// - list object types
		case h.Routes.ObjectTypes.MatchString(uri):
			// TODO: h.listObjectTypes(ctx, w, r)
			herr = NewAppError(404, nil, "Not matched")
		default:
			herr = do404(ctx, w, r)
		}

	case "POST":
		// API
		switch {
		// - create object
		case h.Routes.Objects.MatchString(uri):
			herr = h.createObject(ctx, w, r)
		// - delete object (updates state)
		case h.Routes.ObjectDelete.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectDelete)
			herr = h.deleteObject(ctx, w, r)
		// - undelete object (updates state)
		case h.Routes.ObjectUndelete.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectUndelete)
			herr = h.removeObjectFromTrash(ctx, w, r)
		// - update object properties
		case h.Routes.ObjectProperties.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectProperties)
			herr = h.updateObject(ctx, w, r)
		// - update object stream
		case h.Routes.ObjectStream.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectStream)
			herr = h.updateObjectStream(ctx, w, r)
		// - create object share
		case h.Routes.SharedObject.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.SharedObject)
			herr = h.addObjectShare(ctx, w, r)
		// - move object
		case h.Routes.ObjectMove.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectMove)
			herr = h.moveObject(ctx, w, r)
		// - change owner
		case h.Routes.ObjectChangeOwner.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectChangeOwner)
			herr = h.changeOwner(ctx, w, r)
		// - create favorite
		case h.Routes.FavoriteObject.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.FavoriteObject)
			herr = h.addObjectToFavorites(ctx, w, r)
		// - create symbolic link from object to another folder
		case h.Routes.LinkToObject.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.LinkToObject)
			herr = h.addObjectToFolder(ctx, w, r)
		// - create subscriptionId
		case h.Routes.ObjectSubscribe.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectSubscribe)
			herr = h.addObjectSubscription(ctx, w, r)
		// - create zippost
		case h.Routes.Zip.MatchString(uri):
			herr = h.postZip(ctx, w, r)
		// - create object type
		case h.Routes.ObjectType.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectType)
			// TODO: h.updateObjectType(ctx, w, r)
			herr = NewAppError(404, nil, "Not implemented")
		default:
			herr = do404(ctx, w, r)
		}

	case "DELETE":
		switch {
		// - delete object forever
		case h.Routes.ObjectExpunge.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectExpunge)
			herr = h.deleteObjectForever(ctx, w, r)
			// - remove object share
		case h.Routes.SharedObject.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.SharedObject)
			herr = h.removeObjectShare(ctx, w, r)
		// - remove object favorite
		case h.Routes.FavoriteObject.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.FavoriteObject)
			herr = h.removeObjectFromFavorites(ctx, w, r)
		// - remove symbolic link
		case h.Routes.LinkToObject.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.LinkToObject)
			herr = h.removeObjectFromFolder(ctx, w, r)
		// - remove subscription
		case h.Routes.SubscribedSubscription.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.SubscribedSubscription)
			herr = h.removeObjectSubscription(ctx, w, r)
		// - remove all subscriptions
		case h.Routes.Subscribed.MatchString(uri):
			herr = NewAppError(404, nil, "Not implemented")
			// TODO: h.deleteObjectSubscriptions(ctx, w, r)
			// - empty trash (expunge all in trash)
		case h.Routes.Trash.MatchString(uri):
			herr = NewAppError(404, nil, "Not implemented")
			// TODO: h.emptyTrash(ctx, w, r)
		// - remove object type
		case h.Routes.ObjectType.MatchString(uri):
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectType)
			herr = NewAppError(404, nil, "Not implemented")
			// TODO: h.deleteObjectType(ctx, w, r)

		default:
			herr = do404(ctx, w, r)
		}
	default:
		herr = do404(ctx, w, r)
	}

	// TODO: Before returning, lets capture changes placed back on the context and push into the cache
	// TODO: UserSnippetsFromContext
	// TODO: UserSnippetSQL

	// TODO: Before returning, finalize any metrics, capturing time/error codes ?
	if herr != nil {
		sendAppErrorResponse(logger, &w, herr)
	} else {
		countOKResponse(logger)
	}

	if h.Tracker != nil {
		autoscale.CloudWatchTransaction(beginTSInMS, util.NowMS(), h.Tracker)
	}
}

func newSessionID() string {
	return globalconfig.RandomID()
}

// Before setting the logger, give the context an identity for log correlation
func ContextWithSession(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, SessionID, sessionID)
}

// ContextWithCaller returns a new Context object with a Caller value set. The const CallerVal acts
// as the key that maps to the caller value.
func ContextWithCaller(ctx context.Context, caller Caller) context.Context {
	return context.WithValue(ctx, CallerVal, caller)
}

// ContextWithGEM attaches a GEM to the context object.
func ContextWithGEM(ctx context.Context, gem events.GEM) context.Context {
	return context.WithValue(ctx, GEMVal, gem)
}

// Bind a DAO with our logger, so that SQL can be correlated
func ContextWithDAO(ctx context.Context, genericDAO dao.DAO) context.Context {
	logger := LoggerFromContext(ctx)
	return context.WithValue(ctx, DAO, dao.NewDerivedDAO(genericDAO, logger))
}

func ContextWithGroups(ctx context.Context, groups []string) context.Context {
	return context.WithValue(ctx, Groups, groups)
}

func ContextWithSnippets(ctx context.Context, snippets *acm.ODriveRawSnippetFields) context.Context {
	return context.WithValue(ctx, Snippets, snippets)
}

func DAOFromContext(ctx context.Context) dao.DAO {
	d, ok := ctx.Value(DAO).(dao.DAO)
	if !ok {
		//Should be *completely* impossible as setting these up are preconditions setup in an obvious location
		LoggerFromContext(ctx).Error("cannot get dao from context")
	}
	return d
}

// CallerFromContext extracts a Caller from a context, if set.
func CallerFromContext(ctx context.Context) (Caller, bool) {
	// ctx.Value returns nil if ctx has no value for the key
	// the Caller type assertion returns ok=false for nil.
	caller, ok := ctx.Value(CallerVal).(Caller)
	return caller, ok
}

// GEMFromContext extracts a GEM from a context, if set.
func GEMFromContext(ctx context.Context) (events.GEM, bool) {
	gem, ok := ctx.Value(GEMVal).(events.GEM)
	return gem, ok
}

// GroupsFromContext extracts a list of groups as strings from a context, if set.
func GroupsFromContext(ctx context.Context) ([]string, bool) {
	groups, ok := ctx.Value(Groups).([]string)
	return groups, ok
}

func SnippetsFromContext(ctx context.Context) (*acm.ODriveRawSnippetFields, bool) {
	snippets, ok := ctx.Value(Snippets).(*acm.ODriveRawSnippetFields)
	return snippets, ok
}

func ContextWithLogger(ctx context.Context, logger zap.Logger) context.Context {
	return context.WithValue(ctx, Logger, logger)
}

func SessionIDFromContext(ctx context.Context) string {
	sessionID, ok := ctx.Value(SessionID).(string)
	if !ok {
		return "unknown"
	}
	return sessionID
}

// LoggerFromContext gets an uber zap logger from our context
func LoggerFromContext(ctx context.Context) zap.Logger {
	logger, ok := ctx.Value(Logger).(zap.Logger)
	if !ok {
		log.Print("!!! Any ctx object you get should have a logger set on it")
	}
	return logger
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

func do404(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		caller = Caller{DistinguishedName: "UnknownUser"}
	}
	uri := r.URL.Path
	msg := caller.DistinguishedName + " from address " + r.RemoteAddr + " using " + r.UserAgent() + " unhandled operation " + r.Method + " " + uri
	log.Println("WARN: " + msg)
	return NewAppError(404, nil, "Resource not found")
}

// resolve the ip address once only
var ipString string

func resolveOurIP() string {
	//If our IP changes, a lot more than this breaks (zk for one thing, then nobody can reach us, and cloudwatch reboots us).
	//So, it's a constant from startup of odrive.
	if ipString != "" {
		return ipString
	}
	hostname, err := os.Hostname()
	if err != nil {
		globalconfig.RootLogger.Error("unable to resolve our own hostname")
		return ""
	}
	myIPs, err := net.LookupIP(hostname)
	if err != nil {
		globalconfig.RootLogger.Error("could not lookup IP for hostname")
		return ""
	}
	for _, addr := range myIPs {
		if addr.To4() != nil {
			ipString = addr.String()
			globalconfig.RootLogger.Info("resolved our IP", zap.String("ip", ipString))
			return addr.String()
		}
	}
	return ""
}

// jsonResponse writes a response, and should be called for all HTTP handlers that return JSON.
func jsonResponse(w http.ResponseWriter, i interface{}) {
	w.Header().Set("Content-Type", "application/json")
	jsonData, _ := json.MarshalIndent(i, "", "  ")
	w.Write(jsonData)
}

// newGUID is a helper that ignores the error from util.NewGUID. If that function ever returns
// an error, something is seriously wrong with underlying hardware.
func newGUID() string {
	guid, err := util.NewGUID()
	if err != nil {
		log.Printf("could not create GUID: %s", err.Error())
	}
	return guid
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
	Ciphertext             *regexp.Regexp
	ObjectChangeOwner      *regexp.Regexp
	ObjectDelete           *regexp.Regexp
	ObjectUndelete         *regexp.Regexp
	ObjectExpunge          *regexp.Regexp
	ObjectMove             *regexp.Regexp
	Revisions              *regexp.Regexp
	RevisionStream         *regexp.Regexp
	SharedToMe             *regexp.Regexp
	SharedToOthers         *regexp.Regexp
	SharedToEveryone       *regexp.Regexp
	SharedObject           *regexp.Regexp
	Search                 *regexp.Regexp
	Trash                  *regexp.Regexp
	Zip                    *regexp.Regexp
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
