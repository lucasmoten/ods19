package audit

import (
	"log"

	"decipher.com/object-drive-server/services/audit/generated/acm_thrift"
	"decipher.com/object-drive-server/services/audit/generated/components_thrift"
	"decipher.com/object-drive-server/services/audit/generated/events_thrift"
)

// NewModifiedResourcePair ...
func NewModifiedResourcePair() *components_thrift.ModifiedResourcePair {
	var pair components_thrift.ModifiedResourcePair

	// Set original

	// Set modified

	return &pair
}

// WithCreator ...
func WithCreator(e *events_thrift.AuditEvent, identityType, value string) {
	c := components_thrift.Creator{
		IdentityType: &identityType,
		Value:        &value,
	}
	e.Creator = &c
}

// WithType ...
func WithType(e *events_thrift.AuditEvent, eventType string) {
	e.Type = stringPtr(eventType)
}

// WithAction ...
func WithAction(e *events_thrift.AuditEvent, action string) {
	e.Action = stringPtr(action)
}

// WithActionInitiator ...
func WithActionInitiator(e *events_thrift.AuditEvent, identityType, value string) {
	ai := components_thrift.ActionInitiator{
		IdentityType: stringPtr(identityType),
		Value:        stringPtr(value),
	}
	e.ActionInitiator = &ai
}

// WithActionLocations ...
func WithActionLocations(e *events_thrift.AuditEvent, identifier, value string) {
	if e.ActionLocations == nil {
		e.ActionLocations = make([]*components_thrift.ActionLocation, 0)
	}
	al := components_thrift.ActionLocation{
		Identifier: stringPtr(identifier),
		Value:      stringPtr(value),
	}
	e.ActionLocations = append(e.ActionLocations, &al)
}

// WithActionMode ...
func WithActionMode(e *events_thrift.AuditEvent, actionMode string) {
	e.ActionMode = stringPtr(actionMode)
}

// WithActionResult ...
func WithActionResult(e *events_thrift.AuditEvent, actionResult string) {
	e.ActionResult = stringPtr(actionResult)
}

// WithActionTargetMessages ...
func WithActionTargetMessages(e *events_thrift.AuditEvent, messages ...string) {
	if e.ActionTargetMessages == nil {
		e.ActionTargetMessages = make([]string, 0)
	}
	e.ActionTargetMessages = append(e.ActionTargetMessages, messages...)
}

// WithActionTargetVersions ...
func WithActionTargetVersions(e *events_thrift.AuditEvent, versions ...string) {
	if e.ActionTargetVersions == nil {
		e.ActionTargetVersions = make([]string, 0)
	}
	e.ActionTargetVersions = append(e.ActionTargetVersions, versions...)
}

// WithActionTarget ...
func WithActionTarget(e *events_thrift.AuditEvent, identityType, value string, acm *acm_thrift.Acm) {
	if e.ActionTargets == nil {
		e.ActionTargets = make([]*components_thrift.ActionTarget, 0)
	}
	at := components_thrift.ActionTarget{
		IdentityType: stringPtr(identityType),
		Value:        stringPtr(value),
		Acm:          acm,
	}
	e.ActionTargets = append(e.ActionTargets, &at)
}

// WithAdditionalInfo ...
func WithAdditionalInfo(e *events_thrift.AuditEvent, key, value string) {

	if e.AdditionalInfo == nil {
		e.AdditionalInfo = make(map[string]string)
	}

	if key == "" || value == "" {
		log.Println("NO-OP: empty string passed for WithAdditionalInfo key or value")
		return
	}

	e.AdditionalInfo[key] = value
}

// WithAgency ...
func WithAgency(e *events_thrift.AuditEvent, agency string) {
	e.Agency = stringPtr(agency)
}

// WithCountriesOfCitizenship ...
func WithCountriesOfCitizenship(e *events_thrift.AuditEvent, countries ...string) {
	if e.CountriesOfCitizenship == nil {
		e.CountriesOfCitizenship = make([]string, 0)
	}
	e.CountriesOfCitizenship = append(e.CountriesOfCitizenship, countries...)
}

// WithCreatedOn ...
func WithCreatedOn(e *events_thrift.AuditEvent, createdOn string) {
	e.CreatedOn = stringPtr(createdOn)
}

// WithEnterpriseDataHeader ...
func WithEnterpriseDataHeader(e *events_thrift.AuditEvent, edh *components_thrift.Edh) {
	e.Edh = edh
}

// WithResponsibleEntity ...
func WithResponsibleEntity(e *events_thrift.AuditEvent, country, org, subOrg string) {
	re := components_thrift.ResponsibleEntity{
		Country:         stringPtr(country),
		Organization:    stringPtr(org),
		SubOrganization: stringPtr(subOrg),
	}
	e.ResponsibleEntity = &re
}

// WithHomeAgency ...
func WithHomeAgency(e *events_thrift.AuditEvent, homeAgency string) {
	e.HomeAgency = stringPtr(homeAgency)
}

// WithID ...
func WithID(e *events_thrift.AuditEvent, identifier, value string) {
	id := components_thrift.Id{
		Identifier: stringPtr(identifier),
		Value:      stringPtr(value),
	}

	e.Id = &id
}

// WithNetwork ...
func WithNetwork(e *events_thrift.AuditEvent, network string) {
	e.Network = stringPtr(network)
}

// WithNTPInfo ...
func WithNTPInfo(e *events_thrift.AuditEvent, identityType, lastUpdate, server string) {
	ntpInfo := components_thrift.NTPInfo{
		IdentityType: stringPtr(identityType),
		LastUpdate:   stringPtr(lastUpdate),
		Server:       stringPtr(server),
	}

	e.NtpInfo = &ntpInfo
}

// WithResources ...
func WithResources(e *events_thrift.AuditEvent, resources ...*components_thrift.Resource) {
	if e.Resources == nil {
		e.Resources = make([]*components_thrift.Resource, 0)
	}

	e.Resources = append(e.Resources, resources...)
}

// WithCrossDomain ...
func WithCrossDomain(e *events_thrift.AuditEvent, crossDomain bool) {
	e.CrossDomain = boolPtr(crossDomain)
}

// WithRelevancy ...
func WithRelevancy(e *events_thrift.AuditEvent, relevancy string) {
	e.Relevancy = stringPtr(relevancy)
}

// WithQueryString ...
func WithQueryString(e *events_thrift.AuditEvent, query string) {
	e.QueryString = stringPtr(query)
}

// WithQueryType ...
func WithQueryType(e *events_thrift.AuditEvent, queryType string) {
	e.QueryType = stringPtr(queryType)
}

// WithResult ...
func WithResult(e *events_thrift.AuditEvent, resultType, size, value string) {
	if e.Results == nil {
		e.Results = make([]*components_thrift.Result, 0)
	}
	r := components_thrift.Result{
		Type:  stringPtr(resultType),
		Size:  stringPtr(size),
		Value: stringPtr(value),
	}
	e.Results = append(e.Results, &r)
}

// WithFilter ...
func WithFilter(e *events_thrift.AuditEvent, filter string) {
	if e.Filters == nil {
		e.Filters = make([]*components_thrift.Filter, 0)
	}
	f := components_thrift.Filter{Filter: stringPtr(filter)}
	e.Filters = append(e.Filters, &f)
}

// WithModifiedPairList ...
func WithModifiedPairList(e *events_thrift.AuditEvent, modifiedPairs ...*components_thrift.ModifiedResourcePair) {

	if e.ModifiedPairList == nil {
		e.ModifiedPairList = make([]*components_thrift.ModifiedResourcePair, 0)
	}

	e.ModifiedPairList = append(e.ModifiedPairList, modifiedPairs...)
}

// WithCollaborationFeature ...
func WithCollaborationFeature(
	e *events_thrift.AuditEvent, cFeatureType, value string) {

	if e.CollaborationFeatures == nil {
		e.CollaborationFeatures = make([]*components_thrift.CollaborationFeature, 0)
	}

	cf := components_thrift.CollaborationFeature{
		Type:  stringPtr(cFeatureType),
		Value: stringPtr(value),
	}
	e.CollaborationFeatures = append(e.CollaborationFeatures, &cf)
}

// WithSessionIds ...
func WithSessionIds(e *events_thrift.AuditEvent, sessionIds ...string) {
	if e.SessionIds == nil {
		e.SessionIds = make([]string, 0)
	}

	e.SessionIds = append(e.SessionIds, sessionIds...)
}

// WithWorkflow ...
func WithWorkflow(e *events_thrift.AuditEvent, wf components_thrift.Workflow) {
	e.Workflow = &wf
}

// WithAuthorizationServices ...
func WithAuthorizationServices(e *events_thrift.AuditEvent, auths ...string) {
	if e.AuthorizationServices == nil {
		e.AuthorizationServices = make([]string, 0)
	}

	e.AuthorizationServices = append(e.AuthorizationServices, auths...)
}

// WithAuthorizationServiceTimePeriod ...
func WithAuthorizationServiceTimePeriod(e *events_thrift.AuditEvent, timePeriod string) {
	e.AuthorizationServiceTimePeriod = stringPtr(timePeriod)
}

// WithDevice ...
func WithDevice(e *events_thrift.AuditEvent, deviceLocation, deviceType string) {
	e.Device = &components_thrift.Device{
		Location: stringPtr(deviceLocation), Type: stringPtr(deviceType),
	}
}

// Utilities for dealing with pointers to primitive types.
func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
