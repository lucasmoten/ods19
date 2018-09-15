package audit

import (
	"testing"

	"bitbucket.di2e.net/greymatter/gov-go/audit/events_thrift"
)

// Don't import the GEM, fake it with a wrapper
type envelope struct{ E events_thrift.AuditEvent }

func TestAuditSetters(t *testing.T) {

	var en envelope

	en.E = WithAction(en.E, "ACCESS")
	en.E = WithType(en.E, "EventAccess")

	if en.E.Action == nil {
		t.Errorf("expected action to be set")
	}

	if *en.E.Action != "ACCESS" {
		t.Errorf("unexpected action: %v", en.E.Action)
	}
	if *en.E.Type != "EventAccess" {
		t.Errorf("unexpected type: %v", en.E.Type)
	}

	testPointer(t, en)
}

func testPointer(t *testing.T, en envelope) {
	if *en.E.Action != "ACCESS" {
		t.Errorf("unexpected action: %v", en.E.Action)
	}
	if *en.E.Type != "EventAccess" {
		t.Errorf("unexpected type: %v", en.E.Type)
	}
}
