package audit

/*
func NewThriftAuditor(conn *openssl.Conn) *ThriftAuditClient {

	// TODO should this return an error?
	eventsChan := make(chan *auditevents.AuditEvent, 100)

	trns := thrift.NewTransport(
		thrift.NewFramedReadWriteCloser(conn, 0), thrift.BinaryProtocol)
	thriftClient := thrift.NewClient(trns, true)
	svc := auditservice.AuditServiceClient{Client: thriftClient}
	payloads := NewPayloads(DefaultMaxRequestArraySize)
	c := &ThriftAuditClient{
		Svc:          &svc,
		events:       eventsChan,
		PayloadQueue: payloads,
	}

	return c
}
*/
