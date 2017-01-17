package main

import (
	"testing"

	"decipher.com/object-drive-server/services/audit"

	"github.com/deciphernow/gm-fabric-go/audit/components_thrift"
	auditevent "github.com/deciphernow/gm-fabric-go/audit/events_thrift"
)

func TestUnmarshal(t *testing.T) {
	events := []string{
		`{ 
            "payload": 
            { 
                "audit_event": 
                { 
                    "type": "FOO" 
                } 
            } 
        }
        `,
		`{ 
            "payloadz": 
            { 
                "audit_event": 
                { 
                    "type": "FOO"
                } 
            } 
        }
        `,
		`{
            "audit_event": 
            { 
                "type": "FOO" 
            } 
        }
        `,
	}
	ae, err := unmarshal([]byte(events[0]))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if *ae.Type != "FOO" {
		t.Errorf("error in unmarshal, expected type FOO got %s", *ae.Type)
	}
	_, err = unmarshal([]byte(events[1]))
	if err == nil {
		t.Errorf("error should not be nil")
	}
}

func TestApplyAuditDefaults(t *testing.T) {
	testconf, err := NewAuditConsumerConfig("testfixtures/testconfig.json")
	if err != nil {
		t.Error("error loading testconfig.json")
		t.FailNow()
	}
	t.Log("ApplyAuditDefaults func should set defaulted fields that are unset")
	ae := auditevent.AuditEvent{}
	ae = ApplyAuditDefaults(ae, testconf)
	if *ae.Agency != "Castle Black" {
		t.Error("Expected Agency to be set by defaults")
	}
	if *ae.Edh.Guide.Number != "5555" {
		t.Error("Expected inner EDH Guide Number to be set by defaults")
	}

	t.Log("ApplyAuditDefaults func should not overwrite already-set fields")
	ae = auditevent.AuditEvent{}
	ae = audit.WithEnterpriseDataHeader(ae, components_thrift.Edh{Guide: &components_thrift.Guide{Number: stringPtr("4444")}})
	ae = ApplyAuditDefaults(ae, testconf)
	if *ae.Edh.Guide.Number != "4444" {
		t.Error("ApplyAuditDefaults should not override already-set inner EDH Guide Number")
	}
	if *ae.Creator.IdentityType != "APPLICATION" {
		t.Error("ApplyAuditDefaults should set Creator.IdentityType")
	}
}
