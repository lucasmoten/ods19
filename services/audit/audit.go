package audit

import (
	"log"

	"github.com/deciphernow/gm-fabric-go/audit/acm_thrift"
	"github.com/deciphernow/gm-fabric-go/audit/components_thrift"
	"github.com/deciphernow/gm-fabric-go/audit/events_thrift"
)

/* Create generic Event for now
-struct AuditEvent {
-	1: optional string type,
-	2: optional string action,
-	3: optional components.ActionInitiator action_initiator,
-	4: optional list<components.ActionLocation> action_locations,
-	5: optional string action_mode,
-	6: optional string action_result,
-	7: optional list<string> action_target_messages,
-	8: optional list<string> action_target_versions,
-	9: optional list<components.ActionTarget> action_targets,
-	10: optional map<string, string> additional_info,
-	11: optional string agency,
-	12: optional list<string> countries_of_citizenship,
-	13: optional string created_on,
-	14: optional components.Creator creator,
-	15: optional components.Edh edh,
-	16: optional components.ResponsibleEntity responsible_entity,
-	17: optional string home_agency,
-	18: optional components.Id id,
-	19: optional string network,
-	20: optional components.NTPInfo ntp_info,
-	21: optional list<components.Resource> resources,
-	22: optional bool cross_domain,
-	23: optional string relevancy,
-	24: optional string query_string,
-	25: optional string query_type,
-	26: optional list<components.Result> results,
-	27: optional list<components.Filter> filters,
-	28: optional list<components.ModifiedResourcePair> modified_pair_list,
-	29: optional list<components.CollaborationFeature> collaboration_features,
-	30: optional list<string> session_ids,
-	31: optional components.Workflow workflow,
-	32: optional list<string> authorization_services,
-	33: optional string authorization_service_time_period,
-	34: optional components.Device device
-}
*/

// NewModifiedResourcePair ...
func NewModifiedResourcePair(original components_thrift.Resource, modified components_thrift.Resource) components_thrift.ModifiedResourcePair {
	var pair components_thrift.ModifiedResourcePair

	// Set original
	pair.Original = &original

	// Set modified
	pair.Modified = &modified

	return pair
}

// NewResource creates a resource
func NewResource(Name string, Location string, Size int64, Type string, SubType string, Identifier string) components_thrift.Resource {
	resourceName := components_thrift.ResourceName{
		Title: stringPtr(Name),
	}
	resource := components_thrift.Resource{
		Name:     &resourceName,
		Location: stringPtr(Location),
		// TODO: Convert this later
		Size:       int32PtrOrZero(Size),
		SubType:    stringPtr(SubType),
		Type:       stringPtr(Type),
		Identifier: stringPtr(Identifier),
	}
	return resource
}

// WithCreator ...
func WithCreator(e events_thrift.AuditEvent, identityType, value string) events_thrift.AuditEvent {
	c := components_thrift.Creator{
		IdentityType: &identityType,
		Value:        &value,
	}
	e.Creator = &c
	return e
}

// WithType ...
func WithType(e events_thrift.AuditEvent, eventType string) events_thrift.AuditEvent {
	e.Type = stringPtr(eventType)
	return e
}

// WithAction ...
func WithAction(e events_thrift.AuditEvent, action string) events_thrift.AuditEvent {
	e.Action = stringPtr(action)
	return e
}

// WithActionInitiator ...
func WithActionInitiator(e events_thrift.AuditEvent, identityType, value string) events_thrift.AuditEvent {
	ai := components_thrift.ActionInitiator{
		IdentityType: stringPtr(identityType),
		Value:        stringPtr(value),
	}
	e.ActionInitiator = &ai
	return e
}

// WithActionLocations ...
func WithActionLocations(e events_thrift.AuditEvent, identifier, value string) events_thrift.AuditEvent {
	if e.ActionLocations == nil {
		e.ActionLocations = make([]*components_thrift.ActionLocation, 0)
	}
	al := components_thrift.ActionLocation{
		Identifier: stringPtr(identifier),
		Value:      stringPtr(value),
	}
	e.ActionLocations = append(e.ActionLocations, &al)
	return e
}

// WithActionMode ...
func WithActionMode(e events_thrift.AuditEvent, actionMode string) events_thrift.AuditEvent {
	e.ActionMode = stringPtr(actionMode)
	return e
}

// WithActionResult ...
func WithActionResult(e events_thrift.AuditEvent, actionResult string) events_thrift.AuditEvent {
	e.ActionResult = stringPtr(actionResult)
	return e
}

// WithActionTargetMessages ...
func WithActionTargetMessages(e events_thrift.AuditEvent, messages ...string) events_thrift.AuditEvent {
	if e.ActionTargetMessages == nil {
		e.ActionTargetMessages = make([]string, 0)
	}
	e.ActionTargetMessages = append(e.ActionTargetMessages, messages...)
	return e
}

// WithActionTargetVersions ...
func WithActionTargetVersions(e events_thrift.AuditEvent, versions ...string) events_thrift.AuditEvent {
	if e.ActionTargetVersions == nil {
		e.ActionTargetVersions = make([]string, 0)
	}
	e.ActionTargetVersions = append(e.ActionTargetVersions, versions...)
	return e
}

// WithActionTarget ...
func WithActionTarget(e events_thrift.AuditEvent, at components_thrift.ActionTarget) events_thrift.AuditEvent {
	if e.ActionTargets == nil {
		e.ActionTargets = make([]*components_thrift.ActionTarget, 0)
	}
	e.ActionTargets = append(e.ActionTargets, &at)
	return e
}

// WithActionTargetWithAcm ...
func WithActionTargetWithAcm(e events_thrift.AuditEvent, identityType, value string, acm acm_thrift.Acm) events_thrift.AuditEvent {
	at := components_thrift.ActionTarget{
		IdentityType: stringPtr(identityType),
		Value:        stringPtr(value),
		Acm:          &acm,
	}
	return WithActionTarget(e, at)
}

// WithActionTargetWithoutAcm ...
func WithActionTargetWithoutAcm(e events_thrift.AuditEvent, identityType, value string) events_thrift.AuditEvent {
	at := components_thrift.ActionTarget{
		IdentityType: stringPtr(identityType),
		Value:        stringPtr(value),
	}
	return WithActionTarget(e, at)
}

// WithAdditionalInfo ...
func WithAdditionalInfo(e events_thrift.AuditEvent, key, value string) events_thrift.AuditEvent {

	if e.AdditionalInfo == nil {
		e.AdditionalInfo = make(map[string]string)
	}

	if key == "" || value == "" {
		log.Println("NO-OP: empty string passed for WithAdditionalInfo key or value")
		return e
	}

	e.AdditionalInfo[key] = value
	return e
}

// WithAgency ...
func WithAgency(e events_thrift.AuditEvent, agency string) events_thrift.AuditEvent {
	e.Agency = stringPtr(agency)
	return e
}

// WithCountriesOfCitizenship ...
func WithCountriesOfCitizenship(e events_thrift.AuditEvent, countries ...string) events_thrift.AuditEvent {
	if e.CountriesOfCitizenship == nil {
		e.CountriesOfCitizenship = make([]string, 0)
	}
	e.CountriesOfCitizenship = append(e.CountriesOfCitizenship, countries...)
	return e
}

// WithCreatedOn ...
func WithCreatedOn(e events_thrift.AuditEvent, createdOn string) events_thrift.AuditEvent {
	e.CreatedOn = stringPtr(createdOn)
	return e
}

// WithEnterpriseDataHeader ...
func WithEnterpriseDataHeader(e events_thrift.AuditEvent, edh components_thrift.Edh) events_thrift.AuditEvent {
	e.Edh = &edh
	return e
}

// WithResponsibleEntity ...
func WithResponsibleEntity(e events_thrift.AuditEvent, country, org, subOrg string) events_thrift.AuditEvent {
	re := components_thrift.ResponsibleEntity{
		Country:      stringPtr(country),
		Organization: stringPtr(org),
	}
	e.ResponsibleEntity = &re
	return e
}

// WithHomeAgency ...
func WithHomeAgency(e events_thrift.AuditEvent, homeAgency string) events_thrift.AuditEvent {
	e.HomeAgency = stringPtr(homeAgency)
	return e
}

// WithID NOTE: this function is probably not needed. ID is a global setting in Transformation Service.
func WithID(e events_thrift.AuditEvent, identifier, value string) events_thrift.AuditEvent {
	id := components_thrift.Id{
		Identifier: stringPtr(identifier),
		Value:      stringPtr(value),
	}

	e.Id = &id
	return e
}

// WithNetwork ...
func WithNetwork(e events_thrift.AuditEvent, network string) events_thrift.AuditEvent {
	e.Network = stringPtr(network)
	return e
}

// WithNTPInfo NOTE: this function is probably not needed. ID is a global setting in Transformation Service.
func WithNTPInfo(e events_thrift.AuditEvent, identityType, lastUpdate, server string) events_thrift.AuditEvent {
	ntpInfo := components_thrift.NTPInfo{
		IdentityType: stringPtr(identityType),
		LastUpdate:   stringPtr(lastUpdate),
		Server:       stringPtr(server),
	}

	e.NtpInfo = &ntpInfo
	return e
}

// WithResources ...
func WithResources(e events_thrift.AuditEvent, resources ...components_thrift.Resource) events_thrift.AuditEvent {
	if e.Resources == nil {
		e.Resources = make([]*components_thrift.Resource, 0)
	}

	for _, r := range resources {
		e.Resources = append(e.Resources, &r)
	}
	return e
}

// WithResource adds a resource
func WithResource(e events_thrift.AuditEvent, Name string, Location string, Size int64, Type string, SubType string, Identifier string) events_thrift.AuditEvent {
	resource := NewResource(Name, Location, Size, Type, SubType, Identifier)
	return WithResources(e, resource)
}

// WithCrossDomain ...
func WithCrossDomain(e events_thrift.AuditEvent, crossDomain bool) events_thrift.AuditEvent {
	e.CrossDomain = boolPtr(crossDomain)
	return e
}

// WithRelevancy ...
func WithRelevancy(e events_thrift.AuditEvent, relevancy string) events_thrift.AuditEvent {
	e.Relevancy = stringPtr(relevancy)
	return e
}

// WithQueryString ...
func WithQueryString(e events_thrift.AuditEvent, query string) events_thrift.AuditEvent {
	e.QueryString = stringPtr(query)
	return e
}

// WithQueryType ...
func WithQueryType(e events_thrift.AuditEvent, queryType string) events_thrift.AuditEvent {
	e.QueryType = stringPtr(queryType)
	return e
}

// WithResult ...
func WithResult(e events_thrift.AuditEvent, resultType, size, value string) events_thrift.AuditEvent {
	if e.Results == nil {
		e.Results = make([]*components_thrift.Result, 0)
	}
	r := components_thrift.Result{
		Type:  stringPtr(resultType),
		Size:  stringPtr(size),
		Value: stringPtr(value),
	}
	e.Results = append(e.Results, &r)
	return e
}

// WithFilter ...
func WithFilter(e events_thrift.AuditEvent, filter string) events_thrift.AuditEvent {
	if e.Filters == nil {
		e.Filters = make([]*components_thrift.Filter, 0)
	}
	f := components_thrift.Filter{Filter: stringPtr(filter)}
	e.Filters = append(e.Filters, &f)
	return e
}

// WithModifiedPairList ...
func WithModifiedPairList(e events_thrift.AuditEvent, modifiedPairs ...components_thrift.ModifiedResourcePair) events_thrift.AuditEvent {
	if e.ModifiedPairList == nil {
		e.ModifiedPairList = make([]*components_thrift.ModifiedResourcePair, 0)
	}
	for _, mp := range modifiedPairs {
		e.ModifiedPairList = append(e.ModifiedPairList, &mp)
	}
	return e
}

// WithCollaborationFeature ...
func WithCollaborationFeature(
	e events_thrift.AuditEvent, cFeatureType, value string) events_thrift.AuditEvent {

	if e.CollaborationFeatures == nil {
		e.CollaborationFeatures = make([]*components_thrift.CollaborationFeature, 0)
	}

	cf := components_thrift.CollaborationFeature{
		Type:  stringPtr(cFeatureType),
		Value: stringPtr(value),
	}
	e.CollaborationFeatures = append(e.CollaborationFeatures, &cf)
	return e
}

// WithSessionIds ...
func WithSessionIds(e events_thrift.AuditEvent, sessionIds ...string) events_thrift.AuditEvent {
	if e.SessionIds == nil {
		e.SessionIds = make([]string, 0)
	}

	e.SessionIds = append(e.SessionIds, sessionIds...)
	return e
}

// WithAuthorizationServices ...
func WithAuthorizationServices(e events_thrift.AuditEvent, auths ...string) events_thrift.AuditEvent {
	if e.AuthorizationServices == nil {
		e.AuthorizationServices = make([]string, 0)
	}

	e.AuthorizationServices = append(e.AuthorizationServices, auths...)
	return e
}

// WithAuthorizationServiceTimePeriod ...
func WithAuthorizationServiceTimePeriod(e events_thrift.AuditEvent, timePeriod string) events_thrift.AuditEvent {
	e.AuthorizationServiceTimePeriod = stringPtr(timePeriod)
	return e
}

// WithDevice ...
func WithDevice(e events_thrift.AuditEvent, deviceLocation, deviceType string) events_thrift.AuditEvent {
	e.Device = &components_thrift.Device{
		Location: stringPtr(deviceLocation),
		Type:     stringPtr(deviceType),
	}
	return e
}

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
