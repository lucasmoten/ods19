package zookeeper

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/samuel/go-zookeeper/zk"
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

func randomID() string {
	buf := make([]byte, 4)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}

// Put in a new level in the tree.
// this really only wraps up Create to handle non-existence cleanly.
func makeNewNode(conn *zk.Conn, pathType, prevPath, appendPath string, flags int32, data []byte) (string, error) {
	newPath := prevPath + "/" + appendPath
	exists, _, err := conn.Exists(newPath)
	if err != nil {
		return newPath, err
	}
	if !exists {
		log.Printf("zk: %s %s created", pathType, newPath)
		_, err = conn.Create(newPath, data, flags, PermissiveACL)
		if err != nil {
			return newPath, err
		}
	} else {
		log.Printf("zk: %s %s exists", pathType, appendPath)
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
	log.Printf("zk: connect to %s", zkAddress)
	addrs := strings.Split(zkAddress, ",")
	conn, _, err := zk.Connect(addrs, time.Second*2)
	if err != nil {
		return ZKState{}, err
	}

	//This is the mount point for our zookeeper data, and it should
	//be the same as where AAC mounts
	zkRoot := os.Getenv("ZKROOT")
	if len(zkRoot) == 0 {
		zkRoot = "/cte"
	}

	//Bundle up zookeeper context into a single object
	zkURI := zkRoot + uri
	zkState := ZKState{
		ZKAddress: zkAddress,
		Conn:      conn,
		Protocols: zkURI,
	}

	//Setup the environment for our version of the application
	parts := strings.Split(zkURI, "/")
	organization := parts[1]
	appType := parts[2]
	appName := parts[3]
	appVersion := parts[4]

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

// ServiceAnnouncement ensures that a node for this protocol exists
// and this member is represented with an announcement
//  It creates a node with protocol name and 8 random hex digits
//
//    https/a83e194d
//
// Containing the announcement.
// When our service dies, this node goes away.
//
func ServiceAnnouncement(zkState ZKState, protocol string, stat, host string, port int) error {

	//Turn this into a raw json announcement
	aData := AnnounceData{
		Status: stat,
		ServiceEndpoint: Address{
			Host: host,
			Port: port,
		},
	}

	//Marshall the announcement into bytes
	asBytes, err := json.Marshal(aData)
	if err != nil {
		log.Println("ServiceAnnouncement could not marshal AnnounceData to json: ", err)
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
			randomID(),
			zk.FlagEphemeral,
			asBytes,
		)
		log.Printf("zk: find us at: %s:%d", host, port)
	}
	return err
}
