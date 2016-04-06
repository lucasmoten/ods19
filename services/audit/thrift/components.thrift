/* Components of the enterprise audit event */

include "acm.thrift"

namespace * gov.ic.dodiis.dctc.bedrock.audit.thrift

struct ActionInitiator {
	1: optional string identity_type,
	2: optional string value
}

struct ActionLocation {
	1: optional string identifier,
	2: optional string value
}

struct ActionTarget {
	1: optional string identity_type,
	2: optional string value,
	3: optional acm.Acm acm
}

struct AdditionalInfo {
	1: optional string info_type,
	2: optional string login_failure
}

struct Creator {
	1: optional string identity_type,
	2: optional string value
}

struct Guide {
	1: optional string prefix,
	2: optional string number
}

struct ResponsibleEntity {
	1: optional string country,
	2: optional string organization,
}

struct Security {
	1: optional string owner_producer,
	2: optional string classified_by,
	3: optional string classification_reason,
	4: optional string declass_date,
	5: optional string derived_from
}

struct Edh {
	1: optional acm.Acm acm,
	2: optional Guide guide,
	3: optional ResponsibleEntity responsible_entity,
	4: optional Security security
}

struct Id {
	1: optional string identifier,
	2: optional string value
}

struct NTPInfo {
	1: optional string identity_type,
	2: optional string last_update,
	3: optional string server
}

struct MalwareService {
	1: optional string service
}

struct ResourceParent {
	1: optional string type,
	2: optional string sub_type,
	3: optional string location,
	4: optional string identifier
} 

struct ResourceName {
	1: optional string title,
	2: optional acm.Acm acm
}

struct ResourceContent {
	1: optional string content,
	2: optional acm.Acm acm,
	3: optional bool send
}

struct ResourceDescription {
	1: optional string description,
	2: optional acm.Acm acm
}

struct Resource {
	1: optional string object_type,
	2: optional ResourceName name,
	3: optional string location,
	4: optional i32 size,
	5: optional string sub_type,
	6: optional string type,
	7: optional string role,
	8: optional bool malware_check,
	9: optional string malware_check_status,
	
	10: optional ResourceContent content,
	
	11: optional ResourceDescription description,
	12: optional string identifier,
	13: optional list<MalwareService> malware_services,
	14: optional ResourceParent parent,
	15: optional acm.Acm acm
}

struct ModifiedResourcePair {
    1: optional Resource original,
    2: optional Resource modified
}

struct CollaborationFeature {
    1: optional string type,
    2: optional string value
}
struct Result {
    1: optional string type,
    2: optional string size,
    3: optional string value
}

struct Filter {
    1: optional string filter
}

struct Error {
    1: optional string type,
    2: optional string message
}

struct Workflow {
    1: optional bool complete,
    2: optional list<Error> errors,
    3: optional string id
}

struct Device {
    1: optional string location,
    2: optional string type
}