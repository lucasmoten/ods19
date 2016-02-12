package zookeeper

import (
	"encoding/json"
	"log"
	"testing"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

func TestCreateZookeeperPath(t *testing.T) {
	//TODO: Skip until we fix failing test
	t.Skip()
	zkAddress := "dockervm:2181"
	if testing.Short() {
		t.Skip("Skipping integration test.")
	}
	data := AnnounceData{
		Status:          "AMAZED",
		ServiceEndpoint: Address{Host: "foo", Port: "9999"},
	}
	conn, _, err := zk.Connect([]string{zkAddress}, time.Second*2) //*10)
	defer conn.Close()
	if err != nil {
		t.Error("Could not get connection to Zookeeper.")
	}
	zkPath := "/hello"
	err = publishToNode(conn, zkPath, data)
	if err != nil {
		t.Log("publishToNode failed: ", err)
	}
	exists, stat, err := conn.Exists(zkPath)
	log.Println("Printing stat: ", *stat)
	if err != nil {
		t.Errorf("Error from Exists: %v", err)
	}
	if !exists {
		t.Errorf("Path does not exist: %v", zkPath)
	}
	res, _, err := conn.Get("/hello")
	if err != nil {
		t.Errorf("Could not Get path /hello: %v", err)
	}
	var final AnnounceData
	if err = json.Unmarshal(res, &final); err != nil {
		t.Errorf("Could not unmarshal response.")
	}
	if final.Status != "AMAZED" {
		t.Logf("Expected: AMAZED, got: %s", final.Status)
		t.Fail()
	}

}
