package audit

import (
	auditservice "decipher.com/object-drive-server/services/audit/generated/auditservice_thrift"
	auditevents "decipher.com/object-drive-server/services/audit/generated/events_thrift"
	"github.com/samuel/go-thrift/thrift"
	"github.com/spacemonkeygo/openssl"
)

// Client embeds audit service client functionality from generated code.
type Client struct {
	auditservice.AuditServiceClient
}

// Event embeds the fields from the generated thrift code.
type Event struct {
	auditevents.AuditEvent
}

// NewAuditServiceClient creates an audit.Client given an openssl.Conn pointer.
func NewAuditServiceClient(conn *openssl.Conn) *Client {
	trns := thrift.NewTransport(
		thrift.NewFramedReadWriteCloser(conn, 0), thrift.BinaryProtocol)
	thriftClient := thrift.NewClient(trns, true)
	svc := auditservice.AuditServiceClient{Client: thriftClient}
	return &Client{svc}
}
