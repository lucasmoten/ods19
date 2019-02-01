// This file is automatically generated. Do not modify.

package events_thrift

import (
	"fmt"

	"bitbucket.di2e.net/greymatter/gov-go/audit/components_thrift"
)

var _ = fmt.Sprintf

type AuditEvent struct {
	Type                           *string                                   `thrift:"1" json:"type,omitempty"`
	Action                         *string                                   `thrift:"2" json:"action,omitempty"`
	ActionInitiator                *components_thrift.ActionInitiator        `thrift:"3" json:"action_initiator,omitempty"`
	ActionLocations                []*components_thrift.ActionLocation       `thrift:"4" json:"action_locations,omitempty"`
	ActionMode                     *string                                   `thrift:"5" json:"action_mode,omitempty"`
	ActionResult                   *string                                   `thrift:"6" json:"action_result,omitempty"`
	ActionTargetMessages           []string                                  `thrift:"7" json:"action_target_messages,omitempty"`
	ActionTargetVersions           []string                                  `thrift:"8" json:"action_target_versions,omitempty"`
	ActionTargets                  []*components_thrift.ActionTarget         `thrift:"9" json:"action_targets,omitempty"`
	AdditionalInfo                 map[string]string                         `thrift:"10" json:"additional_info,omitempty"`
	Agency                         *string                                   `thrift:"11" json:"agency,omitempty"`
	CountriesOfCitizenship         []string                                  `thrift:"12" json:"countries_of_citizenship,omitempty"`
	CreatedOn                      *string                                   `thrift:"13" json:"created_on,omitempty"`
	Creator                        *components_thrift.Creator                `thrift:"14" json:"creator,omitempty"`
	Edh                            *components_thrift.Edh                    `thrift:"15" json:"edh,omitempty"`
	ResponsibleEntity              *components_thrift.ResponsibleEntity      `thrift:"16" json:"responsible_entity,omitempty"`
	HomeAgency                     *string                                   `thrift:"17" json:"home_agency,omitempty"`
	Id                             *components_thrift.Id                     `thrift:"18" json:"id,omitempty"`
	Network                        *string                                   `thrift:"19" json:"network,omitempty"`
	NtpInfo                        *components_thrift.NTPInfo                `thrift:"20" json:"ntp_info,omitempty"`
	Resources                      []*components_thrift.Resource             `thrift:"21" json:"resources,omitempty"`
	CrossDomain                    *bool                                     `thrift:"22" json:"cross_domain,omitempty"`
	Relevancy                      *string                                   `thrift:"23" json:"relevancy,omitempty"`
	QueryString                    *string                                   `thrift:"24" json:"query_string,omitempty"`
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
}