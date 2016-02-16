package aac

type RPCClient interface {
	Call(method string, request interface{}, response interface{}) error
}