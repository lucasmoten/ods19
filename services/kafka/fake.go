package kafka

import (
	"decipher.com/object-drive-server/events"
	"github.com/uber-go/zap"
)

// FakeAsyncProducer is a null implementation of events.Publisher.
type FakeAsyncProducer struct {
	logger zap.Logger
}

// NewFakeAsyncProducer returns a null Kafka events.Publisher implementation.
func NewFakeAsyncProducer(logger zap.Logger) *FakeAsyncProducer {
	if logger == nil {
		logger = zap.New(zap.NewJSONEncoder(), zap.Output(zap.Discard), zap.ErrorOutput(zap.Discard))
	}
	logger.Info("Using FakeAsyncProducer")
	return &FakeAsyncProducer{logger}
}

// Publish implements the events.Publisher interface.
func (fake *FakeAsyncProducer) Publish(e events.Event) {
	// no-op
}

// Reconnect implements the events.Publisher interface.
func (fake *FakeAsyncProducer) Reconnect() bool {
	return false
}
