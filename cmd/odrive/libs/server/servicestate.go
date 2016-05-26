package server

import (
	"time"

	"github.com/uber-go/zap"
)

// ServiceState represents the state of an online dependency, such as
// an external service or database.
type ServiceState struct {
	Name    string
	Retries int
	Status  string
	Updated time.Time
}

// ServiceStates is a mapping of services
type ServiceStates map[string]ServiceState

// MarshalLog is a map of service states
func (s ServiceStates) MarshalLog(kv zap.KeyValue) error {
	for k, v := range s {
		kv.AddMarshaler(k, v)
	}
	return nil
}

// MarshalLog is used to marshal the service state to json in the logs
func (s ServiceState) MarshalLog(kv zap.KeyValue) error {
	kv.AddString("name", s.Name)
	kv.AddInt("retries", s.Retries)
	kv.AddString("status", s.Status)
	kv.AddString("updated", s.Updated.String())
	return nil
}

// MaxDelayMillis controls the maximum duration that will be returned from Delay func.
var MaxDelayMillis = 60000

// Delay computes a time.Duration from retries, exponentially backing off until a max delay.
func (s ServiceState) Delay(retries int) time.Duration {
	millis := retries ^ 2
	if millis > MaxDelayMillis {
		return time.Duration(MaxDelayMillis) * time.Millisecond
	}
	return time.Duration(millis) * time.Millisecond
}
