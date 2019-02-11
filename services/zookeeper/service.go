package zookeeper

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/config"
	"github.com/samuel/go-zookeeper/zk"
	"go.uber.org/zap"
)

var (
	logger     = config.RootLogger
	defaultACL = zk.WorldACL(zk.PermAll)
)

// AnnouncementRequest is information required to re-invoke announcements
type AnnouncementRequest struct {
	Protocol string
	Stat     string
	Host     string
	Port     string
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
	Timeout int64
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
func NewZKState(addrs []string, timeout int64) (*ZKState, error) {
	conn, _, err := zk.Connect(addrs, time.Second*time.Duration(timeout))
	if err != nil {
		return nil, err
	}
	zkState := ZKState{Conn: conn, Timeout: timeout}
	return &zkState, nil
}

// RegisterApplication registers object-drive in Zookeeper.
func RegisterApplication(originalPath string, zkAddress string, zkTimeout int64) (*ZKState, error) {
	var err error
	addrs := strings.Split(zkAddress, ",")

	//Because of the args to this function
	zlogger := logger.With(
		zap.String("uri", originalPath),
		zap.String("address", zkAddress),
		zap.Int64("timeout", zkTimeout),
	)

	zlogger.Info("zk connect attempt")
	conn, _, err := zk.Connect(addrs, time.Second*time.Duration(zkTimeout))
	if err != nil {
		return &ZKState{}, err
	}
	zlogger.Debug("zk connected to local pool")
	// Ensure the announcement path is in the correct format
	zkURI := originalPath
	if !strings.HasPrefix(zkURI, "/") {
		zkURI = "/" + zkURI
	}
	zkURI = strings.TrimRight(zkURI, "/")

	zlogger.Debug("zk full URI setting", zap.String("zkuri", zkURI))

	zkState := ZKState{
		ZKAddress:            zkAddress,
		Conn:                 conn,
		Protocols:            zkURI,
		AnnouncementRequests: make([]AnnouncementRequest, 0),
		registeredPath:       zkURI,
		Timeout:              zkTimeout,
	}

	// Create any uncreated nodes, logging modifications made
	var emptyData []byte
	var newPath string
	parts := strings.Split(zkURI, "/")
	for partnum, part := range parts {
		if partnum == 0 {
			continue
		}
		if part != "" {
			newPath, err = makeNewNode(conn, "part "+strconv.Itoa(partnum), newPath, part, 0, emptyData)
			if !isZKOk(err) {
				break
			}
		}
	}
	if err != nil {
		return &zkState, err
	}

	//return the closer, and zookeeper is running
	return &zkState, nil
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
				zap.Error(err),
			)
			//recover after all errors
			doZkRecovery(z, zlogger)
		} else {
			if exists {
				trackAnnouncementsLoop(z, at, handler)
			} else {
				zlogger.Info("zk mount check again")
				//it doesn't exist yet, and no error.  wait until this changes
				ev := <-existsEvents
				if ev.Err != nil {
					zlogger.Error(
						"zk event error",
						zap.Error(ev.Err),
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
			zap.Error(err),
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
				zap.Error(err),
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
					zap.Error(err),
				)
				ok = false
			} else {
				announcements := make(map[string]AnnounceData)
				for _, p := range children {
					thisChild := at + "/" + p
					data, _, err := z.Conn.Get(thisChild)
					if err != nil {
						zlogger.Warn(
							"zk error getting data on peer",
							zap.String("peer", p),
							zap.Error(err),
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
					DoReAnnouncements(z, zlogger)
				}
				if ok {
					zlogger.Info("zk membership change", zap.Any("announcements", announcements))
					if handler != nil {
						handler(at, announcements)
					}
					//blocks until it changes
					ev := <-childrenEvents
					if ev.Err != nil {
						zlogger.Warn(
							"zk event error",
							zap.Error(ev.Err),
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

func doZkRecovery(z *ZKState, zlogger *zap.Logger) bool {
	ok := false
	// Just try to re-announce - this almost always works (pauses and restarts of zk)
	zlogger.Info("zk recovery started")
	err := DoReAnnouncements(z, zlogger)
	if err != nil {
		zlogger.Warn(
			"zk re register error",
			zap.Error(err),
		)
		oldConnection := z.Conn
		zNew, err := RegisterApplication(z.registeredPath, z.ZKAddress, z.Timeout)
		if err != nil {
			zlogger.Warn(
				"zk re register error cant create connection",
				zap.Error(err),
			)
		} else {
			//Use the new connection
			*z = *zNew
			//Try to re-announce again.  If this fails, we still note that we are not ok, so it can be done again later.
			err := DoReAnnouncements(z, zlogger)
			if err != nil {
				zlogger.Warn(
					"zk re register error after create connection",
					zap.Error(err),
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
	switch err.Error() {
	case "zk: node already exists":
		return true
	case "zk: could not connect to a server":
		return false
	default:
		logger.Debug("isZKOk error", zap.String("error", err.Error()))
	}

	return false
}

// ServiceStop will shut down our zookeeper connections, so that we don't get new work.
func ServiceStop(zkState *ZKState, protocol string, logger *zap.Logger) {
	logger.Info("zk terminating")
	zkState.IsTerminated = true
	path := zkState.registeredPath + "/" + protocol + "/" + config.NodeID
	_, _, err := zkState.Conn.Exists(path)
	if err != nil {
		logger.Warn("zk exists node fail", zap.Error(err))
	}
	logger.Info("zk must remove its ephemeral node", zap.String("path", path))
	err = zkState.Conn.Delete(path, -1)
	if err != nil {
		logger.Warn("zk delete node fail", zap.Error(err))
	} else {
		logger.Info("zk terminated our ephemeral node")
	}
}

// DoReAnnouncements will try to fix it. if anything goes wrong, we try again.
func DoReAnnouncements(zkState *ZKState, logger *zap.Logger) error {
	if zkState.IsTerminated {
		return nil
	}

	var returnErr error
	for _, a := range zkState.AnnouncementRequests {
		err := ServiceReAnnouncement(zkState, a.Protocol, a.Stat, a.Host, a.Port)
		if isZKOk(err) == false {
			logger.Warn(
				"zk re announce service", zap.Any("reannouncement", a), zap.Error(err),
			)
			returnErr = err
		}
	}
	return returnErr
}

// ServiceAnnouncement is same as ServiceReAnnouncement with remembering for re-register later
func ServiceAnnouncement(zkState *ZKState, protocol string, stat, host string, port string) error {
	aReq := AnnouncementRequest{
		Protocol: protocol,
		Stat:     stat,
		Host:     host,
		Port:     port,
	}
	logger.Debug("zk service announcing", zap.Any("announcementrequest", aReq))
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
		logger.Error("ServiceAnnouncement could not marshal AnnounceData", zap.Error(err))
		return err
	}

	//Ensure that a node exists for our protocol - effectively permanent
	var emptyData []byte
	newPath, err := makeNewNode(zkState.Conn, "protocols", zkState.Protocols, protocol, 0, emptyData)
	if isZKOk(err) {
		// Register a member with our data - we must use the randomID that was assigned on startup for odrive
		newPath, err = makeNewNode(zkState.Conn, "announcement", newPath, config.NodeID, zk.FlagEphemeral, asBytes)
		if isZKOk(err) {
			err = nil
		}
		logger.Info("zk our address", zap.String("ip", host), zap.Int("port", intPort))
	}
	return err
}

// IsOnline returns a channel that will only receive data if a connection to Zookeeper can be established.
func IsOnline(addrs []string) chan bool {
	success := make(chan bool)
	go func() {
		for {
			log.Println("ZK try:", addrs)
			conn, _, err := zk.Connect(addrs, 5*time.Second)
			if err != nil {
				log.Println("ZK fail", err)
				time.Sleep(10 * time.Second)
				continue
			}
			conn.Close()
			success <- true
			return
		}
	}()

	return success
}
