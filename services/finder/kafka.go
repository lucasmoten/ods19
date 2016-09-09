package finder

import (
	"crypto/tls"
	"log"
	"time"

	"decipher.com/object-drive-server/events"

	"github.com/Shopify/sarama"
	"github.com/uber-go/zap"
)

// AsyncKafkaProducer is a events.Publisher implementation for Kafka queues.
type AsyncKafkaProducer struct {
	producer sarama.AsyncProducer
	logger   zap.Logger
}

// Publish implements the Publisher interface, along with Errors.
func (akp *AsyncKafkaProducer) Publish(e events.Event) {

	msg := sarama.ProducerMessage{
		Topic: "odrive-event",
		Value: sarama.ByteEncoder(e.Yield()),
	}

	akp.producer.Input() <- &msg
}

// Errors implements the events.Publisher interface, alone with Publish.
func (akp *AsyncKafkaProducer) Errors() []error {

	// TODO
	results := make([]error, 0)
	return results
}

// NewAsyncKafkaProducer constructs an AsyncKafkaProducer with internal defaults.
func NewAsyncKafkaProducer(logger zap.Logger, brokerList []string, tlsConfig *tls.Config) *AsyncKafkaProducer {

	config := sarama.NewConfig()
	if tlsConfig != nil {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
	}
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.Flush.Frequency = 500 * time.Millisecond

	producer, err := sarama.NewAsyncProducer(brokerList, config)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}

	go func() {
		for err := range producer.Errors() {
			logger.Error("KAFKA ERROR", zap.Object("err", err))
		}
	}()

	return &AsyncKafkaProducer{producer, logger}
}

// FakeAsyncKafkaProducer is a null implementation of Publisher.
type FakeAsyncKafkaProducer struct {
	logger zap.Logger
}

// NewFakeAsyncKafkaProducer returns a null Kafka Publisher implementation and logs
func NewFakeAsyncKafkaProducer(logger zap.Logger) *FakeAsyncKafkaProducer {
	if logger == nil {
		logger = zap.NewJSON(zap.Output(zap.Discard), zap.ErrorOutput(zap.Discard))
	}
	logger.Info("Using FakeAsyncKafkaProducer")
	return &FakeAsyncKafkaProducer{logger}
}

// Publish implements the Publisher interface, along with Errors.
func (fake *FakeAsyncKafkaProducer) Publish(e events.Event) {
	fake.logger.Debug("Publish event on fake queue", zap.Object("event", e))
}

// Errors implements the events.Publisher interface, alone with Publish.
func (fake *FakeAsyncKafkaProducer) Errors() []error {
	return nil
}