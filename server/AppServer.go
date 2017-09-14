package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/karlseguin/ccache"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/autoscale"
	"decipher.com/object-drive-server/config"
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
	Logger
	SessionID
	DAO
	Groups
	Snippets
)

// AppServer is an http.Handler implementation that holds most service dependencies.
type AppServer struct {
	// Port is the TCP port that the web server listens on.
	Port string
	// Bind is the Network Address that the web server will use.
	Bind string
	// Addr is the combined network address and port the server listens on.
	Addr string
	// DAO is the interface contract with the database.
	RootDAO dao.DAO
	// Conf is the configuration passed to the application.
	Conf config.ServerSettingsConfiguration
	// ServicePrefix is the base url. Used when matching routes.
	ServicePrefix string
	// AAC is a handle to the security service.
	AAC aac.AacService
	// AACZK is a pointer to the cluster where we discover AAC. May be the same as DefaultZK.
	AACZK *zookeeper.ZKState
	// EventQueue is a Publisher interface we use to publish our main event stream.
	EventQueue events.Publisher
	// EventQueueZK is a pointer to the cluster where we discover Kafka. May be the same as DefaultZK.
	EventQueueZK *zookeeper.ZKState
	// Tracker captures metrics about upload/download throughput.
	Tracker *performance.JobReporters
	// TemplateCache holds HTML templates.
	TemplateCache *template.Template
	// StaticDir is the path of static web assets.
	StaticDir string
	// Routes holds the compiled regular expressions used when matching routes. See InitRegex method.
	Routes *StaticRx
	// DefaultZK wraps a connection to the ZK cluster we announce to, and holds state for odrive's registration.
	DefaultZK *zookeeper.ZKState
	// UsersLruCache contains a cache of users with support to purge those least recently used when filling. Up to 1000 users will be retained in memory
	UsersLruCache *ccache.Cache
	// AclWhitelist provides a list of distinguished names allowed to perform impersonation
	ACLImpersonationWhitelist []string
}

// NewAppServer creates an AppServer.
func NewAppServer(conf config.ServerSettingsConfiguration) (*AppServer, error) {

	var templates *template.Template
	var err error

	// If template path specified, ensure templates can be loaded
	if len(conf.PathToTemplateFiles) > 0 {
		templates, err = template.ParseGlob(filepath.Join(conf.PathToTemplateFiles, "*"))
		if err != nil {
			return nil, err
		}
	} else {
		templates = nil
	}

	usersLruCache := ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50))

	staticDir, err := resolvePath(conf.PathToStaticFiles)
	if err != nil {
		return nil, err
	}

	app := AppServer{
		Port:                      conf.ListenPort,
		Bind:                      conf.ListenBind,
		Addr:                      conf.ListenBind + ":" + conf.ListenPort,
		Conf:                      conf,
		Tracker:                   performance.NewJobReporters(1024),
		ServicePrefix:             config.RootURLRegex,
		TemplateCache:             templates,
		StaticDir:                 staticDir,
		UsersLruCache:             usersLruCache,
		ACLImpersonationWhitelist: conf.ACLImpersonationWhitelist,
	}

	app.InitRegex()

	return &app, nil
}

// InitRegex compiles static regexes and initializes the AppServer Routes field.
func (h *AppServer) InitRegex() {
	route := func(path string) StaticRxData {
		v := StaticRxData{
			Pattern:  path,
			RX:       regexp.MustCompile(h.ServicePrefix + path),
			TMGET:    metrics.NewTimer(),
			TMPOST:   metrics.NewTimer(),
			TMDELETE: metrics.NewTimer(),
		}
		return v
	}
	h.Routes = &StaticRx{
		Favicon:     route("/favicon.ico$"),
		StatsObject: route("/stats$"),
		StaticFiles: route("/static/(?P<path>.*)"),
		// Service operations
		APIDocumentation: route("/$"),
		UserStats:        route("/userstats$"),
		Ping:             route("/ping$"),
		// - objects
		Objects:          route("/objects$"),
		Object:           route("/objects/(?P<objectId>[0-9a-fA-F]{32})$"),
		ObjectProperties: route("/objects/(?P<objectId>[0-9a-fA-F]{32})/properties$"),
		ObjectStream:     route("/objects/(?P<objectId>[0-9a-fA-F]{32})/stream(\\.[0-9a-zA-Z]*)?$"),
		Ciphertext:       route("/ciphertext/(?P<zone>[0-9a-zA-Z_]*)?/(?P<rname>[0-9a-fA-F]{64})$"),
		BulkProperties:   route("/objects/properties$"),
		// - actions on objects
		ObjectChangeOwner:  route("/objects/(?P<objectId>[0-9a-fA-F]{32})/owner/(?P<newOwner>.*)$"),
		ObjectDelete:       route("/objects/(?P<objectId>[0-9a-fA-F]{32})/trash$"),
		ObjectUndelete:     route("/objects/(?P<objectId>[0-9a-fA-F]{32})/untrash$"),
		ObjectExpunge:      route("/objects/(?P<objectId>[0-9a-fA-F]{32})$"),
		ObjectMove:         route("/objects/(?P<objectId>[0-9a-fA-F]{32})/move/(?P<folderId>[0-9a-fA-F]{32})?$"),
		ObjectsMove:        route("/objects/move$"),
		ObjectsChangeOwner: route("/objects/owner/(?P<newOwner>.*)$"),
		// - revisions
		Revisions:      route("/revisions/(?P<objectId>[0-9a-fA-F]{32})$"),
		RevisionStream: route("/revisions/(?P<objectId>[0-9a-fA-F]{32})/(?P<revisionId>.*)/stream(\\.[0-9a-zA-Z]*)?$"),
		// - share
		SharedToMe:       route("/shares$"),
		SharedToOthers:   route("/shared$"),
		SharedToEveryone: route("/sharedpublic$"),
		SharedObject:     route("/shared/(?P<objectId>[0-9a-fA-F]{32})$"),
		GroupObjects:     route("/groupobjects/(?P<groupName>.*)$"),
		Groups:           route("/groups$"),
		// - search
		Search: route("/search/(?P<searchPhrase>.*)$"),
		// - trash
		Trash: route("/trashed$"),
		Zip:   route("/zip$"),
		// - not yet implemented. future
		Favorites:              route("/favorites$"),
		FavoriteObject:         route("/favorites/(?P<objectId>[0-9a-fA-F]{32})$"),
		LinkToObject:           route("/objects/(?P<objectId>[0-9a-fA-F]{32})/links/(?P<sourceObjectId>[0-9a-fA-F]{32})$"),
		ObjectLinks:            route("/links/(?P<objectId>[0-9a-fA-F]{32})$"),
		ObjectSubscribe:        route("/objects/(?P<objectId>[0-9a-fA-F]{32})/subscribe$"),
		Subscribed:             route("/subscribed$"),
		SubscribedSubscription: route("/subscribed/(?P<subscriptionId>[0-9a-fA-F]{32})$"),
		ObjectTypes:            route("/objecttypes$"),
		ObjectType:             route("/objecttypes/(?P<objectTypeId>[0-9a-fA-F]{32})$"),
	}
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
	var matched string
	beginTSInMS := util.NowMS()

	sessionID := newSessionID()
	w.Header().Add("sessionid", sessionID)

	caller := CallerFromRequest(r)
	logger := config.RootLogger.With(zap.String("session", sessionID))
	defer logCrashInServeHTTP(logger, w)

	// Authentication check GEM
	authGem := globalEventFromRequest(r)
	authGem.Action = "authenticate"
	authGem.Payload.SessionID = sessionID
	authGem.Payload.Audit = defaultAudit(r)
	authGem.Payload.Audit = audit.WithID(authGem.Payload.Audit, "guid", authGem.ID)
	authGem.Payload.UserDN = caller.DistinguishedName
	authGem.Payload.Audit = audit.WithType(authGem.Payload.Audit, "EventAuthenticate")
	authGem.Payload.Audit = audit.WithAction(authGem.Payload.Audit, "AUTHENTICATE")

	if err := caller.ValidateHeaders(h.ACLImpersonationWhitelist, r); err != nil {
		herr := NewAppError(401, err, err.Error())
		h.publishError(authGem, herr)
		sendErrorResponse(logger, &w, 401, err, err.Error())
		return
	}
	authGem.Payload.Audit = audit.WithActionResult(authGem.Payload.Audit, "SUCCESS")
	h.EventQueue.Publish(authGem)

	// Request GEM routed through
	gem := globalEventFromRequest(r)
	gem.Payload.Audit = defaultAudit(r)
	gem.Payload.Audit = audit.WithID(gem.Payload.Audit, "guid", gem.ID)
	gem.Payload.UserDN = caller.DistinguishedName
	gem.Payload.SessionID = sessionID
	gem.Payload.StreamUpdate = false

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
		zap.String("method", r.Method),
		zap.String("uri", r.RequestURI),
	)

	var uri = r.URL.Path
	var herr *AppError

	// CORS support - if it specifies an origin, then reflect back an access control origin
	if reqOrigin := r.Header.Get("Origin"); reqOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", reqOrigin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Vary", "Origin")

	// The following routes can be handled without calls to the database
	withoutDatabase := false
	switch r.Method {
	case "OPTIONS":
		// Handle the pre-flight request here
		herr = h.cors(ctx, w, r)
		withoutDatabase = true
	case "GET":
		switch {
		case h.Routes.Favicon.RX.MatchString(uri):
			herr = h.favicon(ctx, w, r)
			withoutDatabase = true
		case h.Routes.StatsObject.RX.MatchString(uri):
			matched = "StatsObject"
			herr = h.getStats(ctx, w, r)
			withoutDatabase = true
		case h.Routes.StaticFiles.RX.MatchString(uri):
			matched = "StaticFiles"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.StaticFiles.RX)
			herr = h.serveStatic(ctx, w, r)
			withoutDatabase = true
		// API documentation
		case h.Routes.APIDocumentation.RX.MatchString(uri):
			matched = "APIDocumentation"
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
		herr := NewAppError(500, err, "Error loading user")
		h.publishError(gem, herr)
		return
	}

	ctx = ContextWithUser(ctx, *user)
	groups, snippets, err := h.GetUserGroupsAndSnippets(ctx)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.HasPrefix(err.Error(), auth.ErrServiceNotSuccessful.Error()) {
			statusCode = http.StatusForbidden
		}
		sendErrorResponse(logger, &w, statusCode, err, "Error retrieving user snippets")
		herr := NewAppError(statusCode, err, "Error retrieving user snippets")
		h.publishError(gem, herr)
		return
	}

	// Adding the groups to Caller and into context
	caller.Groups = groups
	ctx = ContextWithCaller(ctx, caller)
	ctx = ContextWithSnippets(ctx, snippets)
	ctx = ContextWithGroups(ctx, groups)

	// Validate User AO Cache state, rebuilding as needed in the background
	if err := h.CheckUserAOCache(ctx); err != nil {
		statusCode := http.StatusInternalServerError
		sendErrorResponse(logger, &w, statusCode, err, "Error checking the user authorization object cache")
		herr := NewAppError(statusCode, err, "Error checking the user authorization object cache")
		h.publishError(gem, herr)
		return
	}

	switch r.Method {
	case "GET":
		switch {
		// - user profile usage information
		case h.Routes.UserStats.RX.MatchString(uri):
			matched = "UserStats"
			herr = h.userStats(ctx, w, r)
		// - get object properties
		case h.Routes.ObjectProperties.RX.MatchString(uri):
			matched = "ObjectProperties"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectProperties.RX)
			herr = h.getObject(ctx, w, r)
		// - get object stream
		case h.Routes.ObjectStream.RX.MatchString(uri):
			matched = "ObjectStream"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectStream.RX)
			herr = h.getObjectStream(ctx, w, r)
		// - get ciphertext
		case h.Routes.Ciphertext.RX.MatchString(uri):
			matched = "Ciphertext"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.Ciphertext.RX)
			herr = h.getCiphertext(ctx, w, r)
		// - list objects at root owned by the caller
		case h.Routes.Objects.RX.MatchString(uri):
			matched = "Objects"
			herr = h.listObjects(ctx, w, r)
		// - list objects at root owned by a group
		case h.Routes.GroupObjects.RX.MatchString(uri):
			matched = "GroupObjects"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.GroupObjects.RX)
			herr = h.listGroupObjects(ctx, w, r)
		// - list objects of object
		case h.Routes.Object.RX.MatchString(uri):
			matched = "Object"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.Object.RX)
			herr = h.listObjects(ctx, w, r)
		// - list trash
		case h.Routes.Trash.RX.MatchString(uri):
			matched = "Trash"
			herr = h.listObjectsTrashed(ctx, w, r)
		// - list objects shared to me
		case h.Routes.SharedToMe.RX.MatchString(uri):
			matched = "SharedToMe"
			herr = h.listUserObjectShares(ctx, w, r)
		// - list objects i've shared with others
		case h.Routes.SharedToOthers.RX.MatchString(uri):
			matched = "SharedToOthers"
			herr = h.listUserObjectsShared(ctx, w, r)
		// - list objects shared to everyone
		case h.Routes.SharedToEveryone.RX.MatchString(uri):
			matched = "SharedToEveryone"
			herr = h.listUserObjectsSharedToEveryone(ctx, w, r)
		// - list object revisions (array of get object properties)
		case h.Routes.Revisions.RX.MatchString(uri):
			matched = "Revisions"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.Revisions.RX)
			herr = h.listObjectRevisions(ctx, w, r)
		// - get object revision stream
		case h.Routes.RevisionStream.RX.MatchString(uri):
			matched = "RevisionStream"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.RevisionStream.RX)
			herr = h.getObjectStreamForRevision(ctx, w, r)
		// - search
		case h.Routes.Search.RX.MatchString(uri):
			matched = "Search"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.Search.RX)
			herr = h.query(ctx, w, r)
		// - my groups with objects
		case h.Routes.Groups.RX.MatchString(uri):
			matched = "Groups"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.Groups.RX)
			herr = h.listMyGroupsWithObjects(ctx, w, r)
		// FUTURE API, NOT YET IMPLEMENTED
		// - get relationships
		case h.Routes.ObjectLinks.RX.MatchString(uri):
			matched = "ObjectLinks"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectLinks.RX)
			herr = h.getRelationships(ctx, w, r)
		// - list favorite / starred objects
		case h.Routes.Favorites.RX.MatchString(uri):
			matched = "Favorites"
			herr = h.listFavorites(ctx, w, r)
		// - list subscribed objects
		case h.Routes.Subscribed.RX.MatchString(uri):
			matched = "Subscribed"
			herr = h.listObjectsSubscriptions(ctx, w, r)
		// - basic HTTP 200 health check
		case h.Routes.Ping.RX.MatchString(uri):
			matched = "Ping"
			herr = nil
		// - list object types
		case h.Routes.ObjectTypes.RX.MatchString(uri):
			matched = "ObjectTypes"
			herr = h.listObjectTypes(ctx, w, r)
		default:
			herr = do404(ctx, w, r)
			h.publishError(gem, herr)
		}

	case "POST":
		if h.RootDAO.IsReadOnly(false) {
			msg := "Service Unavailable. Operation not permitted until database schema upgrade completed. Service operating in read-only mode."
			herr = NewAppError(503, fmt.Errorf(msg), msg)
			break
		}
		// API
		switch {
		// - create object
		case h.Routes.Objects.RX.MatchString(uri):
			matched = "Objects"
			herr = h.createObject(ctx, w, r)
		// - delete object (updates state)
		case h.Routes.ObjectDelete.RX.MatchString(uri):
			matched = "ObjectDelete"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectDelete.RX)
			herr = h.deleteObject(ctx, w, r)
		// - undelete object (updates state)
		case h.Routes.ObjectUndelete.RX.MatchString(uri):
			matched = "ObjectUndelete"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectUndelete.RX)
			herr = h.removeObjectFromTrash(ctx, w, r)
		// - update object properties
		case h.Routes.ObjectProperties.RX.MatchString(uri):
			matched = "ObjectProperties"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectProperties.RX)
			herr = h.updateObject(ctx, w, r)
		// - update object stream
		case h.Routes.ObjectStream.RX.MatchString(uri):
			matched = "ObjectStream"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectStream.RX)
			herr = h.updateObjectStream(ctx, w, r)
		// - create object share
		case h.Routes.SharedObject.RX.MatchString(uri):
			matched = "SharedObject"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.SharedObject.RX)
			herr = h.addObjectShare(ctx, w, r)
		// - move object
		case h.Routes.ObjectMove.RX.MatchString(uri):
			matched = "ObjectMove"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectMove.RX)
			herr = h.moveObject(ctx, w, r)
		// - change owner
		case h.Routes.ObjectChangeOwner.RX.MatchString(uri):
			matched = "ObjectChangeOwner"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectChangeOwner.RX)
			herr = h.changeOwner(ctx, w, r)
		// - create favorite
		case h.Routes.FavoriteObject.RX.MatchString(uri):
			matched = "FavoriteObject"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.FavoriteObject.RX)
			herr = h.addObjectToFavorites(ctx, w, r)
		// - create symbolic link from object to another folder
		case h.Routes.LinkToObject.RX.MatchString(uri):
			matched = "LinkToObject"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.LinkToObject.RX)
			herr = h.addObjectToFolder(ctx, w, r)
		// - create subscriptionId
		case h.Routes.ObjectSubscribe.RX.MatchString(uri):
			matched = "ObjectSubscribe"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectSubscribe.RX)
			herr = h.addObjectSubscription(ctx, w, r)
		// - create zippost
		case h.Routes.Zip.RX.MatchString(uri):
			matched = "Zip"
			herr = h.postZip(ctx, w, r)
		// - create object type
		case h.Routes.ObjectType.RX.MatchString(uri):
			matched = "ObjectType"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectType.RX)
			herr = NewAppError(404, nil, "Not implemented")
		case h.Routes.BulkProperties.RX.MatchString(uri):
			matched = "BulkProperties"
			herr = h.getBulkProperties(ctx, w, r)
		case h.Routes.ObjectsMove.RX.MatchString(uri):
			matched = "ObjectsMove"
			herr = h.doBulkMove(ctx, w, r)
		// - change owner
		case h.Routes.ObjectsChangeOwner.RX.MatchString(uri):
			matched = "ObjectsChangeOwner"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectsChangeOwner.RX)
			herr = h.doBulkOwnership(ctx, w, r)
		default:
			herr = do404(ctx, w, r)
			h.publishError(gem, herr)
		}

	case "DELETE":
		if h.RootDAO.IsReadOnly(false) {
			msg := "Service Unavailable. Operation not permitted until database schema upgrade completed. Service operating in read-only mode."
			herr = NewAppError(503, fmt.Errorf(msg), msg)
			break
		}
		switch {
		// - delete object forever
		case h.Routes.ObjectExpunge.RX.MatchString(uri):
			matched = "ObjectExpunge"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectExpunge.RX)
			herr = h.deleteObjectForever(ctx, w, r)
			// - remove object share
		case h.Routes.SharedObject.RX.MatchString(uri):
			matched = "SharedObject"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.SharedObject.RX)
			herr = h.removeObjectShare(ctx, w, r)
		// - remove object favorite
		case h.Routes.FavoriteObject.RX.MatchString(uri):
			matched = "FavoriteObject"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.FavoriteObject.RX)
			herr = h.removeObjectFromFavorites(ctx, w, r)
		// - remove symbolic link
		case h.Routes.LinkToObject.RX.MatchString(uri):
			matched = "LinkToObject"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.LinkToObject.RX)
			herr = h.removeObjectFromFolder(ctx, w, r)
		// - remove subscription
		case h.Routes.SubscribedSubscription.RX.MatchString(uri):
			matched = "SubscribedSubscription"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.SubscribedSubscription.RX)
			herr = h.removeObjectSubscription(ctx, w, r)
		// - remove all subscriptions
		case h.Routes.Subscribed.RX.MatchString(uri):
			matched = "Subscribed"
			herr = NewAppError(404, nil, "Not implemented")
		// - Empty this user's trash
		case h.Routes.Trash.RX.MatchString(uri):
			matched = "Trash"
			herr = h.expungeDeleted(ctx, w, r)
		// - remove object type
		case h.Routes.ObjectType.RX.MatchString(uri):
			matched = "ObjectType"
			ctx = parseCaptureGroups(ctx, r.URL.Path, h.Routes.ObjectType.RX)
			herr = NewAppError(404, nil, "Not implemented")
			// TODO: h.deleteObjectType(ctx, w, r)
		case h.Routes.Objects.RX.MatchString(uri):
			matched = "Objects"
			herr = h.doBulkDelete(ctx, w, r)
		default:
			herr = do404(ctx, w, r)
			h.publishError(gem, herr)
		}
	default:
		herr = do404(ctx, w, r)
		h.publishError(gem, herr)
	}

	// TODO: Before returning, finalize any metrics, capturing time/error codes ?
	if herr != nil {
		sendAppErrorResponse(logger, &w, herr)
	} else {
		countOKResponse(logger)
	}

	endTSInMS := util.NowMS()
	if h.Tracker != nil {
		autoscale.CloudWatchTransaction(beginTSInMS, endTSInMS, h.Tracker)
	}

	if len(matched) > 0 {
		timerName := r.Method + "/" + matched
		tm := metrics.GetOrRegisterTimer(timerName, metrics.DefaultRegistry)
		tm.Update(time.Duration(endTSInMS-beginTSInMS) * time.Millisecond)
	}
}

func (h *AppServer) publishError(gem events.GEM, herr *AppError) {
	gem.Payload.Audit = audit.WithActionResult(gem.Payload.Audit, "FAILURE")
	gem.Payload.Audit = audit.WithActionTargetMessages(gem.Payload.Audit, strconv.Itoa(herr.Code))
	if herr.Error != nil {
		errMsg := herr.Error.Error()
		if len(errMsg) > 0 {
			gem.Payload.Audit = audit.WithActionTargetMessages(gem.Payload.Audit, errMsg)
		}
	}
	if len(herr.Msg) > 0 {
		gem.Payload.Audit = audit.WithActionTargetMessages(gem.Payload.Audit, herr.Msg)
	}
	gem.Payload.Audit = audit.WithACMCopies(gem.Payload.Audit)
	h.EventQueue.Publish(gem)
}
func (h *AppServer) publishSuccess(gem events.GEM, w http.ResponseWriter) {
	gem.Payload.Audit = audit.WithActionResult(gem.Payload.Audit, "SUCCESS")
	status := w.Header().Get("Status")
	if len(status) == 0 {
		status = "200"
	}
	gem.Payload.Audit = audit.WithActionTargetMessages(gem.Payload.Audit, status)
	gem.Payload.Audit = audit.WithACMCopies(gem.Payload.Audit)
	h.EventQueue.Publish(gem)
}

func newSessionID() string {
	return config.RandomID()
}

// ContextWithSession puts the sessionID on the context, used for log correlation
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

// ContextWithDAO puts the DAO on the context bound with a logger, so that SQL can be correlated
func ContextWithDAO(ctx context.Context, d dao.DAO) context.Context {
	return context.WithValue(ctx, DAO, d)
}

// ContextWithGroups puts the user's groups on the context object
func ContextWithGroups(ctx context.Context, groups []string) context.Context {
	return context.WithValue(ctx, Groups, groups)
}

// ContextWithSnippets puts the user's snippets from AAC on the context object
func ContextWithSnippets(ctx context.Context, snippets *acm.ODriveRawSnippetFields) context.Context {
	return context.WithValue(ctx, Snippets, snippets)
}

// DAOFromContext returns the DAO associated with the context
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

// SnippetsFromContext extracts the user's snippets from the context
func SnippetsFromContext(ctx context.Context) (*acm.ODriveRawSnippetFields, bool) {
	snippets, ok := ctx.Value(Snippets).(*acm.ODriveRawSnippetFields)
	return snippets, ok
}

// ContextWithLogger puts the logger on the context
func ContextWithLogger(ctx context.Context, logger zap.Logger) context.Context {
	return context.WithValue(ctx, Logger, logger)
}

// SessionIDFromContext extracts the session id from the context
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

// ContextWithUser puts the user object on the context and returns the modified context
func ContextWithUser(ctx context.Context, user models.ODUser) context.Context {
	return context.WithValue(ctx, UserVal, user)
}

// UserFromContext extracts the user from a context, if set
func UserFromContext(ctx context.Context) (models.ODUser, bool) {
	user, ok := ctx.Value(UserVal).(models.ODUser)
	return user, ok
}

func do404(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		caller = Caller{DistinguishedName: "UnknownUser"}
	}
	uri := r.URL.Path
	msg := caller.DistinguishedName + " from address " + r.RemoteAddr + " using " + r.UserAgent() + " unhandled operation " + r.Method + " " + uri
	return NewAppError(404, nil, fmt.Sprintf("Resource not found %s", msg))
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

func resolvePath(p string) (string, error) {
	if !path.IsAbs(p) {
		wd, err := os.Getwd()
		if err != nil {
			return p, err
		}
		return path.Clean(path.Join(wd, p)), nil
	}
	return p, nil
}

type StaticRxData struct {
	Pattern  string
	RX       *regexp.Regexp
	TMGET    metrics.Timer
	TMPOST   metrics.Timer
	TMDELETE metrics.Timer
}

// StaticRx statically references compiled regular expressions.
type StaticRx struct {
	Favicon                StaticRxData
	StatsObject            StaticRxData
	StaticFiles            StaticRxData
	Users                  StaticRxData
	APIDocumentation       StaticRxData
	UserStats              StaticRxData
	Objects                StaticRxData
	Object                 StaticRxData
	ObjectProperties       StaticRxData
	ObjectStream           StaticRxData
	Ciphertext             StaticRxData
	ObjectChangeOwner      StaticRxData
	ObjectDelete           StaticRxData
	ObjectUndelete         StaticRxData
	ObjectExpunge          StaticRxData
	ObjectMove             StaticRxData
	ObjectsChangeOwner     StaticRxData
	BulkProperties         StaticRxData
	Ping                   StaticRxData
	Revisions              StaticRxData
	RevisionStream         StaticRxData
	SharedToMe             StaticRxData
	SharedToOthers         StaticRxData
	SharedToEveryone       StaticRxData
	SharedObject           StaticRxData
	GroupObjects           StaticRxData
	Groups                 StaticRxData
	Search                 StaticRxData
	Trash                  StaticRxData
	Zip                    StaticRxData
	Favorites              StaticRxData
	FavoriteObject         StaticRxData
	LinkToObject           StaticRxData
	ObjectLinks            StaticRxData
	ObjectSubscribe        StaticRxData
	Subscribed             StaticRxData
	SubscribedSubscription StaticRxData
	ObjectTypes            StaticRxData
	ObjectType             StaticRxData
	ObjectsMove            StaticRxData
}
