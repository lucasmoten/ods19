package audit

import (
	auditservice "decipher.com/oduploader/services/audit/generated/auditservice_thrift"
)

// Service embeds audit service functionality from generated code.
type Service interface {
	auditservice.AuditService
}
