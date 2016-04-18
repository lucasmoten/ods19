package zookeeper

import (
	"testing"

	cfg "decipher.com/object-drive-server/config"
)

func TestCreateServiceAnnouncement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test.")
	}

	zkAddress := cfg.DockerVM + ":2181"

	zkState, err := RegisterApplication(cfg.RootURL, zkAddress)
	if err != nil {
		t.Errorf("could not create the directory for our app in zk:%v", err)
	}
	defer zkState.Conn.Close()

	state := "ALIVE"
	host := "objectdrivedca1"
	port := "4430"
	err = ServiceAnnouncement(zkState, "https", state, host, port)
	if err != nil {
		t.Errorf("could not announce https node %s %s:%s: %v", state, host, port, err)
	}
}
