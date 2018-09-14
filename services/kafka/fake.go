package kafka

import (
	"bitbucket.di2e.net/dime/object-drive-server/events"
	"go.uber.org/zap"
)

// FakeAsyncProducer is a null implementation of events.Publisher.
type FakeAsyncProducer struct {
	logger *zap.Logger
}

// NewFakeAsyncProducer returns a null Kafka events.Publisher implementation.
func NewFakeAsyncProducer(logger *zap.Logger) *FakeAsyncProducer {
	if logger == nil {
		logger = zap.NewNop()
	}
	logger.Info("using fakeasyncproducer")
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
