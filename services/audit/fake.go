package audit

import (
	"decipher.com/object-drive-server/services/audit/generated/auditservice_thrift"
	"decipher.com/object-drive-server/services/audit/generated/events_thrift"
)

// FakeAudit mocks the Audit Service.
type FakeAudit struct {
	Err  error
	Resp auditservice_thrift.AuditResponse
}

func (fake *FakeAudit) SubmitAuditEvents(events []*events_thrift.AuditEvent) (*auditservice_thrift.AuditResponse, error) {
	return &fake.Resp, fake.Err
}
func (fake *FakeAudit) SubmitEventAccesses(events []*events_thrift.AuditEvent) (*auditservice_thrift.AuditResponse, error) {
	return &fake.Resp, fake.Err
}
func (fake *FakeAudit) SubmitEventAuthenticates(events []*events_thrift.AuditEvent) (*auditservice_thrift.AuditResponse, error) {
	return &fake.Resp, fake.Err
}
func (fake *FakeAudit) SubmitEventCreates(events []*events_thrift.AuditEvent) (*auditservice_thrift.AuditResponse, error) {
	return &fake.Resp, fake.Err
}
func (fake *FakeAudit) SubmitEventDeletes(events []*events_thrift.AuditEvent) (*auditservice_thrift.AuditResponse, error) {
	return &fake.Resp, fake.Err
}
func (fake *FakeAudit) SubmitEventExports(events []*events_thrift.AuditEvent) (*auditservice_thrift.AuditResponse, error) {
	return &fake.Resp, fake.Err
}
func (fake *FakeAudit) SubmitEventImports(events []*events_thrift.AuditEvent) (*auditservice_thrift.AuditResponse, error) {
	return &fake.Resp, fake.Err
}
func (fake *FakeAudit) SubmitEventModifies(events []*events_thrift.AuditEvent) (*auditservice_thrift.AuditResponse, error) {
	return &fake.Resp, fake.Err
}
func (fake *FakeAudit) SubmitEventSearchQrys(events []*events_thrift.AuditEvent) (*auditservice_thrift.AuditResponse, error) {
	return &fake.Resp, fake.Err
}
func (fake *FakeAudit) SubmitEventSystemActions(events []*events_thrift.AuditEvent) (*auditservice_thrift.AuditResponse, error) {
	return &fake.Resp, fake.Err
}
func (fake *FakeAudit) SubmitEventUnknowns(events []*events_thrift.AuditEvent) (*auditservice_thrift.AuditResponse, error) {
	return &fake.Resp, fake.Err
}
