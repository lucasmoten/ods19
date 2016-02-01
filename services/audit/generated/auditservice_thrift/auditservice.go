// This file is automatically generated. Do not modify.

package auditservice_thrift

import (
	"events_thrift"
	"fmt"
)

var _ = fmt.Sprintf

type Int int32

type AuditResponse struct {
	Success  bool     `thrift:"1,required" json:"success"`
	Messages []string `thrift:"2,required" json:"messages"`
}

type AuditServiceException struct {
	Message string `thrift:"1,required" json:"message"`
}

func (e *AuditServiceException) Error() string {
	return fmt.Sprintf("AuditServiceException{Message: %+v}", e.Message)
}

type InvalidInputException struct {
	Message string `thrift:"1,required" json:"message"`
}

func (e *InvalidInputException) Error() string {
	return fmt.Sprintf("InvalidInputException{Message: %+v}", e.Message)
}

type AuditService interface {
	Ping() (string, error)
	SubmitAuditEvent(event *events_thrift.AuditEvent) (*AuditResponse, error)
}

type AuditServiceServer struct {
	Implementation AuditService
}

func (s *AuditServiceServer) Ping(req *AuditServicePingRequest, res *AuditServicePingResponse) error {
	val, err := s.Implementation.Ping()
	res.Value = &val
	return err
}

func (s *AuditServiceServer) SubmitAuditEvent(req *AuditServiceSubmitAuditEventRequest, res *AuditServiceSubmitAuditEventResponse) error {
	val, err := s.Implementation.SubmitAuditEvent(req.Event)
	switch e := err.(type) {
	case *InvalidInputException:
		res.Ex1 = e
		err = nil
	case *AuditServiceException:
		res.Ex2 = e
		err = nil
	}
	res.Value = val
	return err
}

type AuditServicePingRequest struct {
}

type AuditServicePingResponse struct {
	Value *string `thrift:"0" json:"value,omitempty"`
}

type AuditServiceSubmitAuditEventRequest struct {
	Event *events_thrift.AuditEvent `thrift:"1,required" json:"event"`
}

type AuditServiceSubmitAuditEventResponse struct {
	Value *AuditResponse         `thrift:"0" json:"value,omitempty"`
	Ex1   *InvalidInputException `thrift:"1" json:"ex1,omitempty"`
	Ex2   *AuditServiceException `thrift:"2" json:"ex2,omitempty"`
}

type AuditServiceClient struct {
	Client RPCClient
}

func (s *AuditServiceClient) Ping() (ret string, err error) {
	req := &AuditServicePingRequest{}
	res := &AuditServicePingResponse{}
	err = s.Client.Call("ping", req, res)
	if err == nil && res.Value != nil {
		ret = *res.Value
	}
	return
}

func (s *AuditServiceClient) SubmitAuditEvent(event *events_thrift.AuditEvent) (ret *AuditResponse, err error) {
	req := &AuditServiceSubmitAuditEventRequest{
		Event: event,
	}
	res := &AuditServiceSubmitAuditEventResponse{}
	err = s.Client.Call("submitAuditEvent", req, res)
	if err == nil {
		switch {
		case res.Ex1 != nil:
			err = res.Ex1
		case res.Ex2 != nil:
			err = res.Ex2
		}
	}
	if err == nil {
		ret = res.Value
	}
	return
}
