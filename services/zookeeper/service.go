package zookeeper

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

// AnnounceData models the data written to a Zookeeper ephemeral node.
type AnnounceData struct {
	ServiceEndpoint Address `json:"serviceEndpoint"`
	Status          string
}

// Address models a host + port combination.
type Address struct {
	Host string `json:"host"`
	Port string `json:"string"`
}

// PerformServiceAnnounce ...
func PerformServiceAnnounce(zkAddress, zkPath string, data AnnounceData, quit chan bool) {

	var c *zk.Conn
	var err error

	// get connection to zk
	c, _, err = zk.Connect([]string{zkAddress}, time.Second*2) //*10)
	if err != nil {
		ticker := time.NewTicker(time.Millisecond * 500)

		go func() {
			for _ = range ticker.C {
				log.Println("Retrying Zookeeper connection at: ", zkAddress)
				c, _, err = zk.Connect([]string{zkAddress}, time.Second*2) //*10)
				if err == nil {
					ticker.Stop()
					return
				}

			}
		}()
	}

	// we have successfully connected?
	if err := publishToNode(c, zkPath, data); err != nil {
		log.Printf("Zookeeper connection established, but writing to path failed.\n\tPath: %s", zkPath)
	}

	// Loop to stay alive
	for {
		select {
		case msg := <-quit:
			// Try to delete node if quit message is received.
			_ = msg
		}
	}

}

func publishToNode(conn *zk.Conn, zkPath string, data AnnounceData) error {
	asBytes, err := json.Marshal(data)
	if err != nil {
		log.Println("PerformServiceAnnounce could not marshal AnnounceData to json: ", err)
	}
	acl := zk.WorldACL(zk.PermAll)
	// TODO: Do we need to pass flags besides 0 here?
	p, err := conn.Create(zkPath, asBytes, 0, acl)
	if err != nil {
		return errors.New("Error calling Create with Zookeeper conn: " + err.Error())
	}
	log.Println("Successfully registered at Zookeeper path: ", p)
	return nil
}
