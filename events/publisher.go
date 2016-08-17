package events

// Publisher is an interface for async events.
type Publisher interface {
	Publish(e Event)
	Errors() []error
}
