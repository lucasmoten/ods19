package zookeeper

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	globalconfig "decipher.com/object-drive-server/config"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/uber-go/zap"
)

var (
	logger     = globalconfig.RootLogger
	defaultACL = zk.WorldACL(zk.PermAll)
)

// AnnouncementRequest is information required to re-invoke announcements
type AnnouncementRequest struct {
	protocol string
	stat     string
	host     string
	port     string
}

// ZKState holds a ZK connection and other stateful attributes.
type ZKState struct {
	// ZKAddress is the set of host:port that zk will try to connect to
	ZKAddress string
	// Conn is the open zookeeper connection
	Conn *zk.Conn
	// Protocols live under this path in zk
	Protocols string
	// TODO(cm): document this field.
	// Announcements
	AnnouncementRequests []AnnouncementRequest
	// registeredPath is the original path we register in zookeeper
	registeredPath string
	// Whether we are terminated
	IsTerminated bool
	// The timeout on connections in seconds
	Timeout int
}

// AnnounceHandler is a callback when zk data changes
type AnnounceHandler func(mountPoint string, announcements map[string]AnnounceData)

// AnnounceData models the data written to a Zookeeper ephemeral node.
type AnnounceData struct {
	ServiceEndpoint Address `json:"serviceEndpoint"`
	Status          string  `json:"status"`
}

// Address models a host + port combination.
type Address struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Put in a new level in the tree.
// this really only wraps up Create to handle non-existence cleanly.
func makeNewNode(conn *zk.Conn, pathType, prevPath, appendPath string, flags int32, data []byte) (string, error) {
	newPath := prevPath + "/" + appendPath
	exists, _, err := conn.Exists(newPath)
	if err != nil {
		return newPath, err
	}
	zlogger := logger.With(
		zap.String("pathtype", pathType),
		zap.String("newpath", newPath),
		zap.String("appendpath", appendPath),
	)
	if !exists {
		zlogger.Info("zk create")
		_, err = conn.Create(newPath, data, flags, defaultACL)
		if err != nil {
			return newPath, err
		}
	} else {
		zlogger.Info("zk exists")
	}
	return newPath, nil
}

// NewZKState connects to a ZK cluster and returns an object that wraps a connection
// and other state about that cluster.
func NewZKState(addrs []string, timeout int) (*ZKState, error) {
	conn, _, err := zk.Connect(addrs, time.Second*time.Duration(timeout))
	if err != nil {
		return nil, err
	}
	zkState := ZKState{Conn: conn, Timeout: timeout}
	return &zkState, nil
}

// RegisterApplication registers object-drive in Zookeeper.
func RegisterApplication(originalPath, zkAddress string) (*ZKState, error) {
	var err error
	addrs := strings.Split(zkAddress, ",")
	// TODO move this to AppConfiguration.go
	zkTimeout := globalconfig.GetEnvOrDefaultInt("OD_ZK_TIMEOUT", 5)

	//Because of the args to this function
	zlogger := logger.With(
		zap.String("uri", originalPath),
		zap.String("address", zkAddress),
	)

	zlogger.Info("zk connect", zap.Int("timeout", zkTimeout))

	conn, _, err := zk.Connect(addrs, time.Second*time.Duration(zkTimeout))
	if err != nil {
		return &ZKState{}, err
	}

	// Setup the environment for our version of the application
	parts := strings.Split(originalPath, "/")
	// defaults (aligned with the defaults for the environment variables)
	organization := parts[1]
	appType := "service"
	appName := "object-drive"
	appVersion := "1.0"
	// overrides from URI
	if len(parts) != 5 {
		zlogger.Warn("zk base path may not be set correctly")
	}
	organization = assignPart(organization, parts, 1, "organization")
	appType = assignPart(appType, parts, 2, "app type")
	appName = assignPart(appName, parts, 3, "app name")
	appVersion = assignPart(appVersion, parts, 4, "version")

	// Rebuild zkURI
	zkURI := "/" + organization + "/" + appType + "/" + appName + "/" + appVersion
	zlogger.Info("zk full URI setting", zap.String("zkuri", zkURI))

	zkState := ZKState{
		ZKAddress:            zkAddress,
		Conn:                 conn,
		Protocols:            zkURI,
		AnnouncementRequests: make([]AnnouncementRequest, 0),
		registeredPath:       originalPath,
		Timeout:              zkTimeout,
	}

	//Create uncreated nodes, and log modifications we made
	//(it might not be right if we needed to make cte or service)
	var emptyData []byte
	var newPath string
	newPath, err = makeNewNode(conn, "organization", newPath, organization, 0, emptyData)
	if isZKOk(err) {
		newPath, err = makeNewNode(conn, "app type", newPath, appType, 0, emptyData)
		if isZKOk(err) {
			newPath, err = makeNewNode(conn, "app name", newPath, appName, 0, emptyData)
			if isZKOk(err) {
				newPath, err = makeNewNode(conn, "version", newPath, appVersion, 0, emptyData)
			}
		}
	}
	if err != nil {
		return &zkState, err
	}

	//return the closer, and zookeeper is running
	return &zkState, nil
}

func assignPart(defaultValue string, parts []string, idx int, partName string) string {
	if len(parts) > idx {
		ret := strings.TrimSpace(parts[idx])
		if len(ret) > 0 {
			return ret
		}
		logger.Warn("zk uri part empty.  using default.")
		return defaultValue
	}
	logger.Warn("Zookeeper URI not long enough")
	return defaultValue
}

//TrackAnnouncement will call handler every time there is a membership changes
// Ex:
//      aac/2.1/thrift -> [member_00000000 -> {192.168.2.3:9999,...}]
//      object-drive/1.0/https -> [e928923 -> {192.168.3.5:4430,...}]
//
// This will give the *full* membership for that entity, including ourselves.
//
func TrackAnnouncement(z *ZKState, at string, handler AnnounceHandler) {
	go trackMountLoop(z, at, handler)
}

//put a watch on the existence of this node
func trackMountLoop(z *ZKState, at string, handler AnnounceHandler) {
	zlogger := logger.With(zap.String("zk watch", at))
	//Whenever we exit back out to here, do another existence check before attempting
	//to trackAnnouncementsLoop
	for {
		zlogger.Info("zk mount check")
		exists, _, existsEvents, err := z.Conn.ExistsW(at)
		if err != nil {
			zlogger.Error(
				"zk watch exist error",
				zap.String("err", err.Error()),
			)
			//recover after all errors
			doZkRecovery(z, zlogger)
		} else {
			if exists {
				trackAnnouncementsLoop(z, at, handler)
			} else {
				zlogger.Info("zk mount check again")
				//it doesnt exist yet, and no error.  wait until this changes
				ev := <-existsEvents
				if ev.Err != nil {
					zlogger.Error(
						"zk event error",
						zap.String("err", ev.Err.Error()),
					)
					//recover after all errors
					doZkRecovery(z, zlogger)
				}
			}
		}
	}
}

//GetAnnouncements gets the most recent announcement
func GetAnnouncements(z *ZKState, at string) (map[string]AnnounceData, error) {
	zlogger := logger.With(zap.String("zk watch", at))
	children, _, err := z.Conn.Children(at)
	if err != nil {
		zlogger.Error(
			"zk watch child error",
			zap.String("err", err.Error()),
		)
		return nil, nil
	}
	announcements := make(map[string]AnnounceData)
	for _, p := range children {
		thisChild := at + "/" + p
		data, _, err := z.Conn.Get(thisChild)
		if err != nil {
			zlogger.Error(
				"error getting data on peer",
				zap.String("peer", p),
				zap.String("err", err.Error()),
			)
			return nil, err
		}
		var serviceAnnouncement AnnounceData
		json.Unmarshal(data, &serviceAnnouncement)
		announcements[thisChild] = serviceAnnouncement
	}
	return announcements, err
}

//Once the announce point exists, we can track it.
//When it returns, we still need to make sure that the zk node exists, as the error
//could be caused by a removed zk node
func trackAnnouncementsLoop(z *ZKState, at string, handler AnnounceHandler) {
	zlogger := logger.With(zap.String("zk watch", at))
	//if not ok, then we signal that we need recovery
	ok := true
	for {
		if ok {
			zlogger.Info("zk announcement check")
			children, _, childrenEvents, err := z.Conn.ChildrenW(at)
			if err != nil {
				zlogger.Error(
					"zk watch child error",
					zap.String("err", err.Error()),
				)
				ok = false
			} else {
				announcements := make(map[string]AnnounceData)
				for _, p := range children {
					thisChild := at + "/" + p
					data, _, err := z.Conn.Get(thisChild)
					if err != nil {
						zlogger.Error(
							"zk error getting data on peer",
							zap.String("peer", p),
							zap.String("err", err.Error()),
						)
						ok = false
					} else {
						var serviceAnnouncement AnnounceData
						json.Unmarshal(data, &serviceAnnouncement)
						announcements[thisChild] = serviceAnnouncement
						zlogger.Info("zk receive announcement", zap.String("child", thisChild))
					}
				}
				//If there are no children for odrive, then we may end up stuck in this state!
				if len(announcements) == 0 && strings.Contains(at, "object-drive") {
					zlogger.Info(
						"zk object-drive announcements are empty.  re-announcing.",
					)
					doReAnnouncements(z, logger)
				}
				if ok {
					zlogger.Info("zk membership change", zap.Object("announcements", announcements))
					if handler != nil {
						handler(at, announcements)
					}
					//blocks until it changes
					ev := <-childrenEvents
					if ev.Err != nil {
						zlogger.Error(
							"zk event error",
							zap.String("err", ev.Err.Error()),
						)
						ok = false
					}
				}
			}
		}
		//Something is messed up.  If this is our ephemeral node that is messed up, then we should re-register
		if !ok {
			ok = doZkRecovery(z, zlogger)
		}
	}
}

func doZkCleanup(oldConnection *zk.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Print("double close recover", r)
		}
	}()
	oldConnection.Close()
}

func doZkRecovery(z *ZKState, zlogger zap.Logger) bool {
	ok := false
	// Just try to re-announce - this almost always works (pauses and restarts of zk)
	zlogger.Error("zk recover")
	err := doReAnnouncements(z, zlogger)
	if err != nil {
		zlogger.Error(
			"zk re register error",
			zap.String("err", err.Error()),
		)
		oldConnection := z.Conn
		//Redoing zk is dire, and it will disturb aac connections in progress, but at least we will recover
		////Possibility: change the nodeid so that we look like a new instance, like this:
		//globalconfig.NodeID = globalconfig.RandomID()
		zNew, err := RegisterApplication(z.registeredPath, z.ZKAddress)
		if err != nil {
			zlogger.Error(
				"zk re register error cant create connection",
				zap.String("err", err.Error()),
			)
		} else {
			//Use the new connection
			*z = *zNew
			//Try to re-announce again.  If this fails, we still note that we are not ok, so it can be done again later.
			err := doReAnnouncements(z, zlogger)
			if err != nil {
				zlogger.Error(
					"zk re register error after create connection",
					zap.String("err", err.Error()),
				)
			}
			//Get rid of the old connection
			doZkCleanup(oldConnection)
			ok = true
		}
	} else {
		ok = true
	}
	return ok
}

//Node already exists is a sentinel value, not an error
//(We have same issue for deleting stuff that doesn't exist, and closing things that are closed)
func isZKOk(err error) bool {
	if err == nil {
		return true
	}
	if err.Error() == "zk: node already exists" {
		return true
	}
	return false
}

// ServiceStop will shut down our zookeeper connections, so that we don't get new work.
func ServiceStop(zkState *ZKState, protocol string, logger zap.Logger) {
	logger.Info("zk terminating")
	zkState.IsTerminated = true
	path := zkState.registeredPath + "/" + protocol + "/" + globalconfig.NodeID
	_, _, err := zkState.Conn.Exists(path)
	if err != nil {
		logger.Error("zk exists node fail", zap.String("err", err.Error()))
	}
	logger.Info("zk must remove its ephemeral node", zap.String("path", path))
	err = zkState.Conn.Delete(path, -1)
	if err != nil {
		logger.Error("zk delete node fail", zap.String("err", err.Error()))
	} else {
		logger.Info("zk terminated our ephemeral node")
	}
}

// try to fix it. if anything goes wrong, we try again.
func doReAnnouncements(zkState *ZKState, logger zap.Logger) error {
	if zkState.IsTerminated {
		return nil
	}

	var returnErr error
	for _, a := range zkState.AnnouncementRequests {
		err := ServiceReAnnouncement(zkState, a.protocol, a.stat, a.host, a.port)
		if isZKOk(err) == false {
			logger.Error(
				"zk re announce service", zap.Object("reannouncement", a), zap.String("err", err.Error()),
			)
			returnErr = err
		}
	}
	return returnErr
}

// ServiceAnnouncement is same as ServiceReAnnouncement with remembering for re-register later
func ServiceAnnouncement(zkState *ZKState, protocol string, stat, host string, port string) error {
	aReq := AnnouncementRequest{
		protocol: protocol,
		stat:     stat,
		host:     host,
		port:     port,
	}
	zkState.AnnouncementRequests = append(zkState.AnnouncementRequests, aReq)
	return ServiceReAnnouncement(zkState, protocol, stat, host, port)
}

// ServiceReAnnouncement ensures that a node for this protocol exists
// and this member is represented with an announcement
//  It creates a node with protocol name and 8 random hex digits
//
//    https/a83e194d
//
// Containing the announcement.
// When our service dies, this node goes away.
//
func ServiceReAnnouncement(zkState *ZKState, protocol string, stat, host string, port string) error {
	intPort, err := strconv.Atoi(port)
	if err != nil {
		return errors.New("port could not be parsed as int")
	}

	//Turn this into a raw json announcement
	aData := AnnounceData{
		Status: stat,
		ServiceEndpoint: Address{
			Host: host,
			Port: intPort,
		},
	}

	//Marshall the announcement into bytes
	asBytes, err := json.Marshal(aData)
	if err != nil {
		logger.Error("ServiceAnnouncement could not marshal AnnounceData", zap.String("err", err.Error()))
		return err
	}

	//Ensure that a node exists for our protocol - effectively permanent
	var emptyData []byte
	newPath, err := makeNewNode(zkState.Conn, "protocols", zkState.Protocols, protocol, 0, emptyData)
	if isZKOk(err) {
		// Register a member with our data - we must use the randomID that was assigned on startup for odrive
		newPath, err = makeNewNode(zkState.Conn, "announcement", newPath, globalconfig.NodeID, zk.FlagEphemeral, asBytes)
		if isZKOk(err) {
			err = nil
		}
		logger.Info("zk our address", zap.String("ip", host), zap.Int("port", intPort))
	}
	return err
}
