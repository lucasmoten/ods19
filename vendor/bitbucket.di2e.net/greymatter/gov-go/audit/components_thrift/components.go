// This file is automatically generated. Do not modify.

package components_thrift

import (
	"fmt"

	"bitbucket.di2e.net/greymatter/gov-go/audit/acm_thrift"
)

var _ = fmt.Sprintf

type ActionInitiator struct {
	IdentityType *string `thrift:"1" json:"identity_type,omitempty"`
	Value        *string `thrift:"2" json:"value,omitempty"`
}

type ActionLocation struct {
	Identifier *string `thrift:"1" json:"identifier,omitempty"`
	Value      *string `thrift:"2" json:"value,omitempty"`
}

type ActionTarget struct {
	IdentityType *string         `thrift:"1" json:"identity_type,omitempty"`
	Value        *string         `thrift:"2" json:"value,omitempty"`
	Acm          *acm_thrift.Acm `thrift:"3" json:"acm,omitempty"`
}

type AdditionalInfo struct {
	InfoType     *string `thrift:"1" json:"info_type,omitempty"`
	LoginFailure *string `thrift:"2" json:"login_failure,omitempty"`
}

type CollaborationFeature struct {
	Type  *string `thrift:"1" json:"type,omitempty"`
	Value *string `thrift:"2" json:"value,omitempty"`
}

type Creator struct {
	IdentityType *string `thrift:"1" json:"identity_type,omitempty"`
	Value        *string `thrift:"2" json:"value,omitempty"`
}

type Device struct {
	Location *string `thrift:"1" json:"location,omitempty"`
	Type     *string `thrift:"2" json:"type,omitempty"`
}

type Edh struct {
	Acm               *acm_thrift.Acm    `thrift:"1" json:"acm,omitempty"`
	Guide             *Guide             `thrift:"2" json:"guide,omitempty"`
	ResponsibleEntity *ResponsibleEntity `thrift:"3" json:"responsible_entity,omitempty"`
	Security          *Security          `thrift:"4" json:"security,omitempty"`
}

type Error struct {
	Type    *string `thrift:"1" json:"type,omitempty"`
	Message *string `thrift:"2" json:"message,omitempty"`
}

type Filter struct {
	Filter *string `thrift:"1" json:"filter,omitempty"`
}

type Guide struct {
	Prefix *string `thrift:"1" json:"prefix,omitempty"`
	Number *string `thrift:"2" json:"number,omitempty"`
}

type Id struct {
	Identifier *string `thrift:"1" json:"identifier,omitempty"`
	Value      *string `thrift:"2" json:"value,omitempty"`
}

type MalwareService struct {
	MalwareService *string `thrift:"1" json:"malware_service,omitempty"`
}

type ModifiedResourcePair struct {
	Original *Resource `thrift:"1" json:"original,omitempty"`
	Modified *Resource `thrift:"2" json:"modified,omitempty"`
}

type NTPInfo struct {
	IdentityType *string `thrift:"1" json:"identity_type,omitempty"`
	LastUpdate   *string `thrift:"2" json:"last_update,omitempty"`
	Server       *string `thrift:"3" json:"server,omitempty"`
}

type Resource struct {
	ObjectType         *string              `thrift:"1" json:"object_type,omitempty"`
	Name               *ResourceName        `thrift:"2" json:"name,omitempty"`
	Location           *string              `thrift:"3" json:"location,omitempty"`
	Size               *int64               `thrift:"4" json:"size,omitempty"`
	SubType            *string              `thrift:"5" json:"sub_type,omitempty"`
	Type               *string              `thrift:"6" json:"type,omitempty"`
	Role               *string              `thrift:"7" json:"role,omitempty"`
	MalwareCheck       *bool                `thrift:"8" json:"malware_check,omitempty"`
	MalwareCheckStatus *string              `thrift:"9" json:"malware_check_status,omitempty"`
	Content            *ResourceContent     `thrift:"10" json:"content,omitempty"`
	Description        *ResourceDescription `thrift:"11" json:"description,omitempty"`
	Identifier         *string              `thrift:"12" json:"identifier,omitempty"`
	MalwareServices    []*MalwareService    `thrift:"13" json:"malware_services,omitempty"`
	Parent             *ResourceParent      `thrift:"14" json:"parent,omitempty"`
	Acm                *acm_thrift.Acm      `thrift:"15" json:"acm,omitempty"`
}

type ResourceContent struct {
	Content *string         `thrift:"1" json:"content,omitempty"`
	Acm     *acm_thrift.Acm `thrift:"2" json:"acm,omitempty"`
	Send    *bool           `thrift:"3" json:"send,omitempty"`
}

type ResourceDescription struct {
	Content *string         `thrift:"1" json:"content,omitempty"`
	Acm     *acm_thrift.Acm `thrift:"2" json:"acm,omitempty"`
}

type ResourceName struct {
	Title *string         `thrift:"1" json:"title,omitempty"`
	Acm   *acm_thrift.Acm `thrift:"2" json:"acm,omitempty"`
}

type ResourceParent struct {
	Type       *string `thrift:"1" json:"type,omitempty"`
	SubType    *string `thrift:"2" json:"sub_type,omitempty"`
	Location   *string `thrift:"3" json:"location,omitempty"`
	Identifier *string `thrift:"4" json:"identifier,omitempty"`
}

type ResponsibleEntity struct {
	Country      *string `thrift:"1" json:"country,omitempty"`
	Organization *string `thrift:"2" json:"organization,omitempty"`
}

type Result struct {
	Type  *string `thrift:"1" json:"type,omitempty"`
	Size  *string `thrift:"2" json:"size,omitempty"`
	Value *string `thrift:"3" json:"value,omitempty"`
}

type Security struct {
	OwnerProducer        *string `thrift:"1" json:"owner_producer,omitempty"`
	ClassifiedBy         *string `thrift:"2" json:"classified_by,omitempty"`
	ClassificationReason *string `thrift:"3" json:"classification_reason,omitempty"`
	DeclassDate          *string `thrift:"4" json:"declass_date,omitempty"`
	DerivedFrom          *string `thrift:"5" json:"derived_from,omitempty"`
}