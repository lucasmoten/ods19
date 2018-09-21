// Copyright 2017 Decipher Technology Studios LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gk

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"
	"time"

	"github.com/deciphernow/gm-fabric-go/zkutil"

	"github.com/samuel/go-zookeeper/zk"
)

// -----------------------------------------------------------------------------
// The JSON blob stored in ZooKeeper.
// These correspond to the following parts of the GateKeeper config:
//
//   zookeeper.jsonPath.host=serviceEndpoint.host
//   zookeeper.jsonPath.port=serviceEndpoint.port
//
// Here's an example of the JSON:
//
//	{
//		"serviceEndpoint": {
//			"host": "127.0.0.1",
//			"port": 8080
//		},
//		"status": "ALIVE"
//	}
//
// TODO: The JSON schema should be handled dynamically, such that one can
// specify the jsonPath to use, rather than baking that into the AnnounceData
// type.

// AnnounceData models the data written to a ZooKeeper ephemeral node.
type AnnounceData struct {
	ServiceEndpoint Address `json:"serviceEndpoint"`
	Status          status  `json:"status"`
}

// Address models a host + port combination.
type Address struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// don't export; force the user to use one of the constants
type status string

const Alive = status("ALIVE")

// -----------------------------------------------------------------------------

// GetIP returns an IPv4 Address in string format suitable for Gatekeeper to reach us at
func GetIP() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	if len(hostname) <= 0 {
		return "", errors.New("could not find our hostname")
	}
	myIPs, err := net.LookupIP(hostname)
	if err != nil {
		return "", errors.New("could not get a set of IPs for our hostname")
	}
	if len(myIPs) <= 0 {
		return "", errors.New("could not find IPv4 address")
	}
	for a := range myIPs {
		if myIPs[a].To4() != nil {
			return myIPs[a].String(), nil
		}
	}
	return "", errors.New("could not find IPv4 address")
}

type Registration struct {
	Path   string
	Status status
	Host   string
	Port   int
}

func (self *Registration) toAnn() AnnounceData {
	json := AnnounceData{
		Status: self.Status,
		ServiceEndpoint: Address{
			Host: self.Host,
			Port: self.Port,
		},
	}
	return json
}

// Register the service announcement with ZooKeeper.
// Should look something like:
//
//	cancel := gk.Announce([]string{"localhost:2181"}, &gk.Registration{
//		Path:   "/cte/service/category/1.0.0",
//		Host:   "127.0.0.1",
//		Port:   8090,
//		Status: gk.Alive,
//	})
//	defer cancel()
func Announce(servers []string, reg *Registration) (cancel func()) {
	done := make(chan struct{})
	cancel = func() {
		close(done)
	}

	annJson := reg.toAnn()
	annBytes, _ := json.Marshal(annJson)

	go func() {
		// Announce until cancelled.
		for doAnn(done, annBytes, servers, reg) {
			select {
			case <-done:
				return
			case <-time.After(time.Second * 2):
				// Retry.
			}
		}
	}()

	return cancel
}

func doAnn(done chan struct{}, annBytes []byte, servers []string, reg *Registration) bool {
	conn, evCh, err := zk.Connect(servers, 2*time.Second)
	if err != nil {
		log.Println("Zookeeper connection failed:", err)
		// Time to reconnect.
		return true
	}

	defer conn.Close()

	expired := true

create:
	_, err = zkutil.CreateRecursive(conn, reg.Path+"/member_", annBytes, zk.FlagEphemeral|zk.FlagSequence, zkutil.DefaultACLs)
	if err == nil {
		// Mark that we successfully created the node.
		expired = false
	} else {
		log.Println("Gatekeeper announcement failed:", err)
	}

	// Wait until we're cancelled or the connection fails.
	for {
		select {
		case <-done:
			// Bail.
			return false
		case ev := <-evCh:
			// The zk.Conn will attempt to reconnect repeatedly upon disconnect,
			// and as long as a connection is established within the session timeout,
			// the ephemeral nodes will continue to exist.
			// On the other hand, if the session expires, we need to recreate the
			// node.
			if ev.Err != nil {
				log.Println("Gatekeeper announcement failed:", err)
				return true
			} else if ev.State == zk.StateExpired {
				log.Println("Gatekeeper announcement expired")
				expired = true
			} else if expired && ev.State == zk.StateHasSession {
				log.Println("Re-announcing service")
				goto create
			}
		}
	}
}
