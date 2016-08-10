package zookeeper

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	globalconfig "decipher.com/object-drive-server/config"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/uber-go/zap"
)

var (
	logger = globalconfig.RootLogger
)

var PermissiveACL = zk.WorldACL(zk.PermAll)

// AnnouncementRequest is information required to re-invoke announcements
type AnnouncementRequest struct {
	protocol string
	stat     string
	host     string
	port     string
}

// ZKState is everything about zookeeper that we might need to know
type ZKState struct {
	// ZKAddress is the set of host:port that zk will try to connect to
	ZKAddress string
	// Conn is the open zookeeper connection
	Conn *zk.Conn
	// Protocols live under this path in zk
	Protocols string
	// Announcements
	AnnouncementRequests []AnnouncementRequest
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
		_, err = conn.Create(newPath, data, flags, PermissiveACL)
		if err != nil {
			return newPath, err
		}
	} else {
		zlogger.Info("zk exists")
	}
	return newPath, nil
}

// RegisterApplication registers object-drive directory heirarchy in zookeeper
// in parallel with the aac.
// Paths are structured:
//
//  /cte - where zk specific stuff to organization is for cte
//    /service - a type of thing being managed, service in this case
//      /object-drive - an application name
//        /1.0   - a version for the application
//
//  Under this mount point we should have service announcements (json data)
//  for each port that this version of the service exposes:
//
//    /https
//        /member_00000000  - includes some json that includes port and ip of member
//        /member_00000001  ...
//
//    {"host":"192.168.99.100", "port":"4430"}
//
//  The member nodes should be ephemeral so that they clean out when the service dies
//
func RegisterApplication(zkURI, zkAddress string) (*ZKState, error) {
	var err error

	//Get open zookeeper connection, and get a handle on closing it later
	addrs := strings.Split(zkAddress, ",")
	// TODO move this to AppConfiguration.go
	zkTimeout := globalconfig.GetEnvOrDefaultInt("OD_ZK_TIMEOUT", 5)

	//Because of the args to this function
	zlogger := logger.With(
		zap.String("uri", zkURI),
		zap.String("address", zkAddress),
	)

	zlogger.Info("zk connect", zap.Int("timeout", zkTimeout))

	conn, _, err := zk.Connect(addrs, time.Second*time.Duration(zkTimeout))
	if err != nil {
		return &ZKState{}, err
	}

	//Setup the environment for our version of the application
	parts := strings.Split(zkURI, "/")
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
	zkURI = "/" + organization + "/" + appType + "/" + appName + "/" + appVersion
	zlogger.Info("zk full URI setting", zap.String("zkuri", zkURI))

	zkState := ZKState{
		ZKAddress:            zkAddress,
		Conn:                 conn,
		Protocols:            zkURI,
		AnnouncementRequests: make([]AnnouncementRequest, 0),
	}

	//Create uncreated nodes, and log modifications we made
	//(it might not be right if we needed to make cte or service)
	var emptyData []byte
	var newPath string
	newPath, err = makeNewNode(conn, "organization", newPath, organization, 0, emptyData)
	if err == nil {
		newPath, err = makeNewNode(conn, "app type", newPath, appType, 0, emptyData)
		if err == nil {
			newPath, err = makeNewNode(conn, "app name", newPath, appName, 0, emptyData)
			if err == nil {
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
	ok := true
	for {
		zlogger.Info("zk announcement check")
		children, _, childrenEvents, err := z.Conn.ChildrenW(at)
		if err != nil {
			zlogger.Error(
				"zk watch child error",
				zap.String("err", err.Error()),
			)
			ok = false
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
				ok = false
			} else {
				var serviceAnnouncement AnnounceData
				json.Unmarshal(data, &serviceAnnouncement)
				announcements[thisChild] = serviceAnnouncement
			}
		}
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
		//Something is messed up.  Re-register our announcements to make things ok to try again.
		if !ok {
			doReAnnouncements(z, zlogger)
			ok = true
		}
	}
}

// try to fix it.
func doReAnnouncements(zkState *ZKState, logger zap.Logger) {
	for _, a := range zkState.AnnouncementRequests {
		err := ServiceReAnnouncement(zkState, a.protocol, a.stat, a.host, a.port)
		if err != nil {
			logger.Error(
				"zk re announce service", zap.Object("reannouncement", a), zap.String("err", err.Error()),
			)
		}
	}
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
	if err == nil {
		// Register a member with our data - we must use the randomID that was assigned on startup for odrive
		newPath, err = makeNewNode(zkState.Conn, "announcement", newPath, globalconfig.NodeID, zk.FlagEphemeral, asBytes)
		logger.Info("zk our address", zap.String("ip", host), zap.Int("port", intPort))
	}
	return err
}
