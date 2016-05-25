package server

import "time"

// ServiceState represents the state of an online dependency, such as
// an external service or database.
type ServiceState struct {
	Name    string
	Retries int
	Status  string
	Updated time.Time
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
