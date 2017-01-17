package main

import (
	"github.com/deciphernow/gm-fabric-go/audit/components_thrift"
	auditevent "github.com/deciphernow/gm-fabric-go/audit/events_thrift"
)

// ApplyAuditDefaults fills in the defaults for an AuditEvent and returns a copy of the event
// with values populated.
func ApplyAuditDefaults(ae auditevent.AuditEvent, conf AuditConsumerConfig) auditevent.AuditEvent {
	// var ret auditevent.AuditEvent

	// Defaultable top-level *string fields
	if ae.Type == nil {
		val := conf.Defaults["type"]
		ae.Type = stringPtr(val)
	}
	if ae.Action == nil {
		val := conf.Defaults["action"]
		ae.Action = stringPtr(val)
	}
	if ae.ActionMode == nil {
		val := conf.Defaults["action_mode"]
		ae.ActionMode = stringPtr(val)
	}
	if ae.ActionResult == nil {
		val := conf.Defaults["action_result"]
		ae.ActionResult = stringPtr(val)
	}
	if ae.Agency == nil {
		val := conf.Defaults["agency"]
		ae.Agency = stringPtr(val)
	}
	if ae.CreatedOn == nil {
		// we could do assignment here, but this is an upstream concern
	}
	if ae.HomeAgency == nil {
		val := conf.Defaults["home_agency"]
		ae.HomeAgency = stringPtr(val)
	}
	if ae.Network == nil {
		val := conf.Defaults["network"]
		ae.Network = stringPtr(val)
	}

	// Nested object defaults.
	// If an EDH is not set, make one.
	if ae.Edh == nil {
		ae.Edh = &components_thrift.Edh{}
	}
	if ae.Edh.Guide == nil {
		ae.Edh.Guide = &components_thrift.Guide{}
	}
	if ae.Edh.Guide.Number == nil {
		// set the guide number
		val := conf.Defaults["edh_guide_number"]
		ae.Edh.Guide.Number = stringPtr(val)
	}

	if ae.Creator == nil {
		ae.Creator = &components_thrift.Creator{}
	}
	if ae.Creator.Value == nil {
		val := conf.Defaults["creator_value"]
		ae.Creator.Value = stringPtr(val)
		val = conf.Defaults["creator_type"]
		ae.Creator.IdentityType = stringPtrDefault(val, "APPLICATION")
	}
	return ae
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringPtrDefault(s, val string) *string {
	if s == "" {
		return &val
	}
	return &s
}

// Our TODO list
/*
	ActionInitiator                *components_thrift.ActionInitiator        `thrift:"3" json:"action_initiator,omitempty"`
	ActionLocations                []*components_thrift.ActionLocation       `thrift:"4" json:"action_locations,omitempty"`
	ActionTargetMessages           []string                                  `thrift:"7" json:"action_target_messages,omitempty"`
	ActionTargetVersions           []string                                  `thrift:"8" json:"action_target_versions,omitempty"`
	ActionTargets                  []*components_thrift.ActionTarget         `thrift:"9" json:"action_targets,omitempty"`
	AdditionalInfo                 map[string]string                         `thrift:"10" json:"additional_info,omitempty"`
	CountriesOfCitizenship         []string                                  `thrift:"12" json:"countries_of_citizenship,omitempty"`
	Edh                            *components_thrift.Edh                    `thrift:"15" json:"edh,omitempty"`
	ResponsibleEntity              *components_thrift.ResponsibleEntity      `thrift:"16" json:"responsible_entity,omitempty"`
	Id                             *components_thrift.Id                     `thrift:"18" json:"id,omitempty"`
	NtpInfo                        *components_thrift.NTPInfo                `thrift:"20" json:"ntp_info,omitempty"`
	Resources                      []*components_thrift.Resource             `thrift:"21" json:"resources,omitempty"`
	// use string type for CrossDomain?
	CrossDomain                    *bool                                     `thrift:"22" json:"cross_domain,omitempty"`
    // not defaultable
	Relevancy                      *string                                   `thrift:"23" json:"relevancy,omitempty"`
	// not defaultable
	QueryString                    *string                                   `thrift:"24" json:"query_string,omitempty"`
	// not defaultable
	QueryType                      *string                                   `thrift:"25" json:"query_type,omitempty"`

	Results                        []*components_thrift.Result               `thrift:"26" json:"results,omitempty"`
	Filters                        []*components_thrift.Filter               `thrift:"27" json:"filters,omitempty"`
	ModifiedPairList               []*components_thrift.ModifiedResourcePair `thrift:"28" json:"modified_pair_list,omitempty"`
	CollaborationFeatures          []*components_thrift.CollaborationFeature `thrift:"29" json:"collaboration_features,omitempty"`
	SessionIds                     []string                                  `thrift:"30" json:"session_ids,omitempty"`
	WorkflowId                     *string                                   `thrift:"31" json:"workflow_id,omitempty"`
	AuthorizationServices          []string                                  `thrift:"32" json:"authorization_services,omitempty"`
	AuthorizationServiceTimePeriod *string                                   `thrift:"33" json:"authorization_service_time_period,omitempty"`
	Device                         *components_thrift.Device                 `thrift:"34" json:"device,omitempty"`
*/
