package client

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"decipher.com/object-drive-server/events"
	"github.com/Shopify/sarama"
	"github.com/wvanbergen/kafka/consumergroup"
)

type OdriveResponder struct {
	DebugMode bool
	Consumer  *consumergroup.ConsumerGroup
	Conf      Config
	Fetch     func(*OdriveResponder, *events.GEM) error
	Timeout   time.Duration
}

func NewOdriveResponder(
	cfg Config,
	groupName string,
	zkLocation string,
	fetch func(*OdriveResponder, *events.GEM) error,
) (*OdriveResponder, error) {
	cgconf := consumergroup.NewConfig()
	consumerGroup, err := consumergroup.JoinConsumerGroup(
		groupName,
		[]string{"odrive-event"},
		strings.Split(zkLocation, ","),
		cgconf,
	)
	if err != nil {
		return nil, err
	}
	c := &OdriveResponder{
		Conf:     cfg,
		Fetch:    fetch,
		Consumer: consumerGroup,
	}
	return c, nil
}

func (c *OdriveResponder) Note(msg string, args ...interface{}) {
	if c.DebugMode {
		log.Printf(msg, args...)
	}
}

func ParseGemEvent(msg *sarama.ConsumerMessage) (*events.GEM, error) {
	var gem events.GEM
	err := json.Unmarshal(msg.Value, &gem)
	if err != nil {
		return nil, err
	}
	return &gem, nil
}

// Parse a kafka action
func (c *OdriveResponder) Handle(msg *sarama.ConsumerMessage) error {
	gem, err := ParseGemEvent(msg)
	if err != nil {
		return err
	}
	if gem == nil {
		return nil
	}
	return c.Fetch(c, gem)
}

func (c *OdriveResponder) ConsumeKafka() error {
	// Try other partitions if there is nothing in this one
	timeout := time.After(c.Timeout)
	msgs := c.Consumer.Messages()
	for {
		select {
		case msg := <-msgs:
			c.Consumer.CommitUpto(msg)
			c.Handle(msg)
		case <-timeout:
			break
		}
	}
	return nil
}
