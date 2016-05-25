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

// ZKState is everything about zookeeper that we might need to know
type ZKState struct {
	// ZKAddress is the set of host:port that zk will try to connect to
	ZKAddress string
	// Conn is the open zookeeper connection
	Conn *zk.Conn
	// Protocols live under this path in zk
	Protocols string
}

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
func RegisterApplication(uri, zkAddress string) (ZKState, error) {
	var err error

	//Get open zookeeper connection, and get a handle on closing it later
	addrs := strings.Split(zkAddress, ",")
	//This is the mount point for our zookeeper data, and it should
	//be the same as where AAC mounts
	zkRoot := globalconfig.GetEnvOrDefault("OD_ZK_ROOT", "/cte")
	zkTimeout := globalconfig.GetEnvOrDefaultInt("OD_ZK_TIMEOUT", 5)

	//Because of the args to this function
	zlogger := logger.With(
		zap.String("uri", uri),
		zap.String("address", zkAddress),
		zap.String("zkroot", zkRoot),
	)

	zlogger.Info("zk connect", zap.Int("timeout", zkTimeout))

	conn, _, err := zk.Connect(addrs, time.Second*time.Duration(zkTimeout))
	if err != nil {
		return ZKState{}, err
	}

	//Bundle up zookeeper context into a single object
	zkURI := zkRoot + uri

	//Setup the environment for our version of the application
	parts := strings.Split(zkURI, "/")
	// defaults (aligned with the defaults for the environment variables)
	organization := zkRoot[1:]
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
		ZKAddress: zkAddress,
		Conn:      conn,
		Protocols: zkURI,
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
		return zkState, err
	}

	//return the closer, and zookeeper is running
	return zkState, nil
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

// ServiceAnnouncement ensures that a node for this protocol exists
// and this member is represented with an announcement
//  It creates a node with protocol name and 8 random hex digits
//
//    https/a83e194d
//
// Containing the announcement.
// When our service dies, this node goes away.
//
func ServiceAnnouncement(zkState ZKState, protocol string, stat, host string, port string) error {

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
	newPath, err := makeNewNode(
		zkState.Conn,
		"protocols",
		zkState.Protocols,
		protocol,
		0,
		emptyData,
	)
	if err == nil {
		//Register a member with our data - ephemeral so that data disappears when we die
		newPath, err = makeNewNode(
			zkState.Conn,
			"announcement",
			newPath,
			globalconfig.NodeID,
			zk.FlagEphemeral,
			asBytes,
		)
		logger.Info("zk our address", zap.String("ip", host), zap.Int("port", intPort))
	}
	return err
}
