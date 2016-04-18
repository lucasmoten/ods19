package audit

import (
	audit "decipher.com/object-drive-server/services/audit/generated/auditservice_thrift"
)

// FakeAudit mocks the Audit Service.
type FakeAudit struct {
	Err error
	Msg string
}

// Ping for FakeAudit.
func (fake *FakeAudit) Ping() (string, error) {
	return fake.Msg, fake.Err
}

// SubmitAuditEvent for FakeAudit.
func (fake *FakeAudit) SubmitAuditEvent(
	req *audit.AuditServiceSubmitAuditEventRequest, res *audit.AuditServiceSubmitAuditEventResponse) error {
	return fake.Err
}
