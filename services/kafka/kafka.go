package kafka

import (
	"encoding/json"
	"errors"
	"fmt"

	"decipher.com/object-drive-server/events"

	"github.com/Shopify/sarama"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/uber-go/zap"
)

// AsyncProducer is a events.Publisher implementation for Kafka queues.
type AsyncProducer struct {
	producer       sarama.AsyncProducer
	logger         zap.Logger
	reconnect      bool
	successActions []string
	failureActions []string
}

// Publish implements the events.Publisher interface.
func (ap *AsyncProducer) Publish(e events.Event) {

	publishEvent := false
	if e.IsSuccessful() {
		publishEvent = publishEvent || stringInSlice("*", ap.successActions)
		publishEvent = publishEvent || stringInSlice(e.EventAction(), ap.successActions)
	} else {
		publishEvent = publishEvent || stringInSlice("*", ap.failureActions)
		publishEvent = publishEvent || stringInSlice(e.EventAction(), ap.failureActions)
	}
	if !publishEvent {
		return
	}

	msg := sarama.ProducerMessage{
		Topic: "odrive-event",
		Value: sarama.ByteEncoder(e.Yield()),
	}

	ap.producer.Input() <- &msg
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Reconnect implements the events.Publisher interface.
func (ap *AsyncProducer) Reconnect() bool {
	return ap.reconnect
}

// Opt sets an option on an AsyncProducer.
type Opt func(*AsyncProducer)

// WithLogger sets a custom logger on an AsyncProducer.
func WithLogger(logger zap.Logger) Opt {
	return func(ap *AsyncProducer) {
		ap.logger = logger
	}
}

// WithPublishActions sets success and failure actions that should be published on an AsyncProducer
func WithPublishActions(successActions []string, failureActions []string) Opt {
	return func(ap *AsyncProducer) {
		ap.successActions = successActions
		ap.failureActions = failureActions
	}
}

// NewAsyncProducer constructs an AsyncProducer with internal defaults and supplied options.
func NewAsyncProducer(brokerList []string, opts ...Opt) (*AsyncProducer, error) {

	producer, err := sarama.NewAsyncProducer(brokerList, nil)
	if err != nil {
		return nil, err
	}
	ap := AsyncProducer{producer: producer, reconnect: false}
	defaults(&ap)
	for _, opt := range opts {
		opt(&ap)
	}
	ap.start()

	return &ap, nil
}

func defaults(ap *AsyncProducer) {
	ap.logger = zap.New(zap.NewJSONEncoder())
}

// DiscoverKafka keeps a connection to Kafka alive. A discovered instance is returned early, and a setter callback
// is invoked when nodes in the cluster change.
func DiscoverKafka(conn *zk.Conn, path string, setter func(*AsyncProducer), opts ...Opt) (*AsyncProducer, error) {

	brokers := buildBrokers(conn, path)
	if len(brokers) < 1 {
		return nil, errors.New("no broker data found at Kafka path")
	}

	ap, err := NewAsyncProducer(brokers, opts...)
	if err != nil {
		return nil, fmt.Errorf("broker data found, but could not establish connection to Kafka")
	}

	// Get the chan zk.Event for changes to children
	_, _, events, err := conn.ChildrenW(path)
	if err != nil {
		return nil, err
	}
	l := ap.logger

	go func() {
		for e := range events {
			l.Info("zk event watching kafka path", zap.Object("event", e))
			if e.Type == zk.EventNodeChildrenChanged {
				brokers := buildBrokers(conn, path)
				if len(brokers) < 1 {
					l.Error("no kafka brokers found at zk path", zap.String("path", path))
				} else {
					p, err := NewAsyncProducer(brokers, opts...)
					if err != nil {
						l.Error("error re-creating Kafka connection", zap.Object("err", err))
						continue
					}
					l.Info("found kafka brokers", zap.Object("brokers", brokers))
					// invoke the callback with a new instance
					setter(p)
				}
			}
		}
	}()

	return ap, nil
}

// buildBrokers queries a zookeeper path and returns a string slice of host:port pairs
// suitable for the kafka library's constructor. Errors are ignored, because the caller
// can decide what to do if a zero-length list of brokers is returned.
func buildBrokers(conn *zk.Conn, path string) []string {

	var brokers []string

	children, _, _ := conn.Children(path)
	for _, c := range children {
		data, _, err := conn.Get(path + "/" + c)
		if err != nil {
			break
		}
		var a addr
		if err := json.Unmarshal(data, &a); err != nil {
			break
		}
		brokers = append(brokers, fmt.Sprintf("%s:%v", a.Host, a.Port))
	}
	return brokers

}

func (ap *AsyncProducer) start() {

	go func() {
		defer func() { ap.reconnect = true }()
		for err := range ap.producer.Errors() {
			ap.logger.Error("KAFKA ERROR", zap.Object("err", err))
			if requiresReconnect(err) {
				ap.reconnect = true
			}
		}
	}()

}

func requiresReconnect(err interface{}) bool {

	// From sarama docs: ProducerError is the type of error generated when the producer
	// fails to deliver a message. It contains the original ProducerMessage as well as
	// the actual error value.
	pe, ok := err.(*sarama.ProducerError)
	if !ok {
		return false
	}

	if v, ok := pe.Err.(sarama.KError); ok {
		switch v {
		// NOTE(cm): ErrUnknown (-1) is the only error seen in the logs so far
		case sarama.ErrUnknown,
			sarama.ErrClosedClient,
			sarama.ErrOffsetOutOfRange,
			sarama.ErrInvalidMessage,
			sarama.ErrUnknownTopicOrPartition,
			sarama.ErrInvalidMessageSize,
			sarama.ErrLeaderNotAvailable,
			sarama.ErrNotLeaderForPartition,
			sarama.ErrBrokerNotAvailable,
			sarama.ErrMessageSizeTooLarge,
			sarama.ErrStaleControllerEpochCode,
			sarama.ErrOffsetMetadataTooLarge,
			sarama.ErrNetworkException,
			sarama.ErrInvalidTopic,
			sarama.ErrMessageSetSizeTooLarge,
			sarama.ErrNotEnoughReplicas,
			sarama.ErrNotEnoughReplicasAfterAppend,
			sarama.ErrInvalidRequiredAcks,
			sarama.ErrInconsistentGroupProtocol,
			sarama.ErrInvalidGroupId,
			sarama.ErrUnknownMemberId,
			sarama.ErrRebalanceInProgress,
			sarama.ErrInvalidCommitOffsetSize,
			sarama.ErrTopicAuthorizationFailed,
			sarama.ErrGroupAuthorizationFailed,
			sarama.ErrClusterAuthorizationFailed,
			sarama.ErrInvalidTimestamp,
			sarama.ErrUnsupportedSASLMechanism,
			sarama.ErrIllegalSASLState,
			sarama.ErrUnsupportedVersion:
			return true
		case sarama.ErrInvalidSessionTimeout,
			sarama.ErrIllegalGeneration,
			sarama.ErrOffsetsLoadInProgress,
			sarama.ErrConsumerCoordinatorNotAvailable,
			sarama.ErrNotCoordinatorForConsumer,
			sarama.ErrRequestTimedOut,
			sarama.ErrReplicaNotAvailable,
			sarama.ErrNoError:
			return false

		}
	}

	return false
}

type addr struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}
