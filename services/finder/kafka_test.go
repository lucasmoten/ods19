package finder_test

import (
	"testing"
	"time"

	"github.com/Shopify/sarama"
)

func TestKafkaConnection(t *testing.T) {

	t.Skip()
	brokerList := []string{"localhost:9092"}

	t.Log("trying raw producer")
	p, err := sarama.NewAsyncProducer(brokerList, nil)
	if err != nil {
		t.Errorf("KAFKA ERROR: ", err)
	}

	p.Input() <- &sarama.ProducerMessage{
		Topic: "odrive_finder",
		Key:   sarama.StringEncoder("xyz"),
		Value: sarama.ByteEncoder([]byte("{\"hello\": \"world\"}")),
	}
	t.Logf("Sleeping")
	time.Sleep(2 * time.Second)
}
