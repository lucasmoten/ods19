package zookeeper_test

import (
	"testing"
	"time"

	"decipher.com/object-drive-server/services/zookeeper"
)

func TestCreateServiceAnnouncement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test.")
	}

	zkAddress := "zk:2181"

	zkBasePath := "/cte/service/object-drive/1.0"

	zkState, err := zookeeper.RegisterApplication(zkBasePath, zkAddress)
	if err != nil {
		// zk was spawned before this test, so just try once more later
		t.Logf("sleeping a few seconds waiting for zk to settle")
		time.Sleep(10 * time.Second)
		zkState, err = zookeeper.RegisterApplication(zkBasePath, zkAddress)
		if err != nil {
			t.Errorf("could not create the directory for our app in zk:%v", err)
		}
	}
	defer zkState.Conn.Close()

	state := "ALIVE"
	host := "objectdrivedca1"
	port := "4430"
	err = zookeeper.ServiceAnnouncement(zkState, "https", state, host, port)
	if err != nil {
		t.Errorf("could not announce https node %s %s:%s: %v", state, host, port, err)
	}
}
