/* The goal is to generate Thrift objects that transform into JSON acceptable to Enterprise-Audit-Exchange (EAE) and Bedrock core ACM.
   If making changes, ensure validity with Bedrock Core ACM, EAE and AuditXML.
   The naming conventions matter very much since that's how they'll be translated to JSON and serialized to EAE. */

namespace * gov.ic.dodiis.dctc.bedrock.audit.thrift

include "acm.thrift"
include "components.thrift"

/* Create generic Event for now */
struct AuditEvent {
	1: optional string type,
	2: optional string action,
	3: optional components.ActionInitiator action_initiator,
	4: optional list<components.ActionLocation> action_locations,
	5: optional string action_mode,
	6: optional string action_result,
	7: optional list<string> action_target_messages,
	8: optional list<string> action_target_versions,
	9: optional list<components.ActionTarget> action_targets,
	10: optional map<string, string> additional_info,
	11: optional string agency,
	12: optional list<string> countries_of_citizenship,
	13: optional string created_on,
	14: optional components.Creator creator,
	15: optional components.Edh edh,
	16: optional components.ResponsibleEntity responsible_entity,
	17: optional string home_agency,
	18: optional components.Id id,
	19: optional string network,
	20: optional components.NTPInfo ntp_info,
	21: optional list<components.Resource> resources,
	22: optional bool cross_domain,
	23: optional string relevancy,
	24: optional string query_string,
	25: optional string query_type,
	26: optional list<components.Result> results,
	27: optional list<components.Filter> filters,
	28: optional list<components.ModifiedResourcePair> modified_pair_list,
	29: optional list<components.CollaborationFeature> collaboration_features,
	30: optional list<string> session_ids,
	31: optional components.Workflow workflow,
	32: optional list<string> authorization_services,
	33: optional string authorization_service_time_period,
	34: optional components.Device device
}