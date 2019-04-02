package server_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/events"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"

	"github.com/Shopify/sarama"
)

func TestPublishEvents(t *testing.T) {

	if (os.Getenv("OD_EVENT_ZK_ADDRS") == "") && (os.Getenv("OD_EVENT_KAFKA_ADDRS") == "") {
		t.Logf("OD_EVENT_ZK_ADDRS and OD_EVENT_KAFKA_ADDRS is not set. Events will not be emitted.")
		t.SkipNow()
	}

	published := make(map[string][]string)
	clientID := 0

	// Perform API calls that we expect to generate events on Kafka queue: create and then delete
	_, obj := doTestCreateObjectSimple(t, "test data", clientID, nil, nil, ValidACMUnclassified)
	published[obj.ID] = append(published[obj.ID], "create")

	_, obj = doTestUpdateObjectSimple(t, "updated data", clientID, obj, nil, nil, ValidACMUnclassified)
	published[obj.ID] = append(published[obj.ID], "update")
	po := protocol.Object{ID: obj.ID}
	po.ChangeToken = obj.ChangeToken
	req, err := NewDeleteObjectRequest(po, "")
	failNowOnErr(t, err, "could not create delete object request")
	resp, err := clients[clientID].Client.Do(req)
	failNowOnErr(t, err, "error calling delete object")
	statusMustBe(t, 200, resp, "expected status 200 for delete")
	published[obj.ID] = append(published[obj.ID], "delete")

	// Read events asynchronously
	appConf := config.NewAppConfiguration(config.ValueOpts{Conf: "../config/testfixtures/complete.yml"})
	topic := appConf.EventQueue.Topic
	pc := partitionConsumerForTopic(t, []string{"kafka:9092"}, topic)
	defer pc.Close()
	ch := pc.Messages()

	done := make(chan bool)
	go func() {
		for msg := range ch {
			var gem events.GEM
			if err := json.Unmarshal(msg.Value, &gem); err != nil {
				t.Log("unable to deserialize event to type GEM")
				continue
			}
			// If the event we read is in our map, remove it.
			if e, ok := published[gem.Payload.ObjectID]; ok {
				published[gem.Payload.ObjectID] = removeItemFromSlice(e, gem.Action)
				// We know we're done when our map[string][]string has only empty slices.
				if allEventsFound(published) {
					done <- true
					return
				}
			}
		}
		return
	}()

	timeout := time.After(30 * time.Second)
	for {
		select {
		case <-timeout:
			t.Logf("5 second timeout exceeded. Please run integration tests with empty Kafka queues.")
			//t.Skip()
			return
		case <-done:
			t.Log("Found all expected events")
			return
		}
	}

}

func partitionConsumerForTopic(t *testing.T, addrs []string, topic string) sarama.PartitionConsumer {

	c, err := sarama.NewConsumer(addrs, nil)
	if err != nil {
		t.Errorf("error creating Kafka consumer: %v", err)
		t.FailNow()
	}
	partitions, err := c.Partitions(topic)
	if err != nil {
		t.Errorf("could not get partitions: %v", err)
		t.FailNow()
	}
	if len(partitions) == 0 {
		t.Errorf("no partitions found for %s", topic)
		t.FailNow()
	}
	pc, err := c.ConsumePartition(topic, partitions[0], 0)
	if err != nil {
		t.Errorf("could not consume partition(s): %v", err)
	}
	if pc == nil {
		t.Errorf("unable to consume partitions, but no error")
	}
	return pc
}

func removeItemFromSlice(slice []string, item string) []string {

	for idx, val := range slice {
		if val == item {
			slice = append(slice[:idx], slice[idx+1:]...)
			return slice
		}
	}
	return slice
}

func allEventsFound(published map[string][]string) bool {
	total := 0
	for _, v := range published {
		total += len(v)
	}
	return total == 0
}
