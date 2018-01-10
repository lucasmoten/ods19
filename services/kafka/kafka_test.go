package kafka

import (
	"os"
	"testing"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

func TestDiscoverKafka(t *testing.T) {
	if os.Getenv("OD_EVENT_KAFKA_ADDRS") == "" {
		t.Skip()
	}
	conn, _, err := zk.Connect([]string{"zk:2181"}, 5*time.Second)
	if err != nil {
		t.Errorf("connection error: %v", err)
	}

	type container struct{ producer *AsyncProducer }
	var appServer container
	var quit = make(chan bool)

	setter := func(p *AsyncProducer) {
		appServer.producer = p
		quit <- true
	}

	ap, err := DiscoverKafka(conn, "/brokers/ids", setter)
	if err != nil {
		t.Errorf("error from DiscoverKafka: %v", err)
	}
	appServer.producer = ap

	triggerEvent(t, conn, "/brokers/ids", quit)

	if appServer.producer == nil {
		t.Errorf("expected producer field to be set")
	}

}

// triggerEvent is an wacky routine to trigger a ZK event.
func triggerEvent(t *testing.T, conn *zk.Conn, path string, quit chan bool) {
	// normally, we get an immediate response - so having a large timeout is not a problem
	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-timeout:
			// Note: rjf - on my machine this almost always times out.
			t.Log("timeout exceeded")
			return
		case <-quit:
			return
		default:
			brokers, _, err := conn.Children(path)
			if err != nil {
				t.Errorf("error getting children: %v", err)
				t.FailNow()
			}
			for _, b := range brokers {
				t.Log("try create delete")
				pth := path + "/" + b
				data, _, err := conn.Get(pth)
				failNowOnErr(t, "could not get path", err)
				if len(data) == 0 {
					t.Errorf("no data at path")
				}
				conn.Delete(pth, -1)
				failNowOnErr(t, "could not delete path", err)
				s, err := conn.Create(pth, data, 0, zk.WorldACL(zk.PermAll))
				failNowOnErr(t, "could not create path", err)
				t.Log("s: " + s)
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func failNowOnErr(t *testing.T, msg string, err error) {
	if err != nil {
		t.Errorf("%s: %v", msg, err)
		t.FailNow()
	}
}
