package server

import (
	"encoding/hex"
	"encoding/json"
	"time"

	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/services/audit"
	"github.com/deciphernow/gm-fabric-go/audit/acm_thrift"
	"github.com/deciphernow/gm-fabric-go/audit/components_thrift"
	"github.com/deciphernow/gm-fabric-go/audit/events_thrift"
)

// Utilities for dealing with pointers to primitive types.
func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
func int32Ptr(i int32) *int32    { return &i }
func int64Ptr(i int64) *int64    { return &i }
func int32PtrOrZero(i int64) *int32 {
	var zero int32
	var x = int32(i)
	if zero > x {
		return &zero
	}
	return &x
}

// WithResourcesFromResultset writes lists of objects
func WithResourcesFromResultset(e events_thrift.AuditEvent, results models.ODObjectResultset) events_thrift.AuditEvent {
	for _, r := range results.Objects {
		e = audit.WithResources(e, NewResourceFromObject(r))
	}
	return e
}

// NewResourceFromObject creates an audit resource from a object drive object suitable as targetted resource, original or modified
func NewResourceFromObject(obj models.ODObject) components_thrift.Resource {
	resource := components_thrift.Resource{}
	resource.Name = &components_thrift.ResourceName{
		Title: stringPtr(obj.Name),
	}
	resource.Identifier = stringPtr(hex.EncodeToString(obj.ID))
	resource.Type = stringPtr("OBJECT")
	resource.SubType = stringPtr(obj.TypeName.String)
	resource.Description = &components_thrift.ResourceDescription{
		Content: stringPtr(obj.Description.String),
	}
	resource.Size = int64Ptr(obj.ContentSize.Int64)
	acm := NewAuditACMFromString(obj.RawAcm.String)
	resource.Acm = &acm
	return resource
}

// NewAuditACMFromString initializes an audit acm from the acm of an object
func NewAuditACMFromString(rawacm string) acm_thrift.Acm {
	var acm acm_thrift.Acm
	acmBytes := []byte(rawacm)
	json.Unmarshal(acmBytes, &acm)
	return acm
}

// NewAuditTargetForID creates an audit action target for an object id
func NewAuditTargetForID(ID []byte) components_thrift.ActionTarget {
	at := components_thrift.ActionTarget{}
	at.IdentityType = stringPtr("OBJECTID")
	at.Value = stringPtr(hex.EncodeToString(ID))
	return at
}

// ResetBulkItem clears fields specific to an event and assigns new id and time
func ResetBulkItem(gem events.GEM) events.GEM {
	gem.ID = newGUID()
	gem.Payload.Audit = audit.WithID(gem.Payload.Audit, "guid", gem.ID)
	gem.Payload.Audit = audit.WithCreatedOn(gem.Payload.Audit, time.Now().UTC().Format("2006-01-02T15:04:05.000Z"))
	gem.Payload.Audit.Resources = nil
	gem.Payload.Audit.ModifiedPairList = nil
	return gem
}