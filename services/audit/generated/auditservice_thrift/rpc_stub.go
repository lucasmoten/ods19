package auditservice_thrift

type RPCClient interface {
	Call(method string, request interface{}, response interface{}) error
}
