package audit_test

import (
	"testing"

	"decipher.com/object-drive-server/services/audit"
	auditevents "decipher.com/object-drive-server/services/audit/generated/events_thrift"
)

func TestAddToPayloads(t *testing.T) {

	data := &auditevents.AuditEvent{}
	data2 := &auditevents.AuditEvent{}
	data3 := &auditevents.AuditEvent{}

	p := audit.NewPayloads(10)

	p.Add("EventAccess", data)
	p.Add("EventAccess", data2)
	p.Add("EventFoo", data3)

	q := p.M["EventAccess"]

	if q == nil {
		t.Errorf("Expected *Queue not nil")
	}

}
