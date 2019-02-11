FORMAT: 1A

# Object Drive 1.0 

<table style="width:100%;border:0px;padding:0px;border-spacing:0;border-collapse:collapse;font-family:Helvetica;font-size:10pt;vertical-align:center;"><tbody><tr><td style="padding:0px;font-size:10pt;">Version</td><td style="padding:0px;font-size:10pt;">--Version--</td><td style="width:20%;font-size:8pt;"> </td><td style="padding:0px;font-size:10pt;">Build</td><td style="padding:0px;font-size:10pt;">--BuildNumber--</td><td style="width:20%;font-size:8pt;"></td><td style="padding:0px;font-size:10pt;">Date</td><td style="padding:0px;font-size:10pt;">--BuildDate--</td></tr></tbody></table>

# Group Navigation

## Table of Contents

+ [Service Overview](../../)
+ [RESTful API documentation](rest.html)
+ [Emitted Events documentation](events.html)
+ [Environment](environment.html)
+ [Changelog](changelog.html)

# Group Emitted Events

Object Drive may be configured to emit events to a kafka broker for success and failure results for operations performed.


## Environment Variable Configuration

Object Drive publishes a single event stream for client applications. The following are the 
[environment variables](environment.html) for configuring Object Drive to publish events.

| Name | Description | Default |
| --- | --- | --- |
| OD_EVENT_KAFKA_ADDRS | A comma-separated list of **host:port** pairs.  These are Kafka brokers. | |
| OD_EVENT_ZK_ADDRS | A comma-separated list of **host:port** pairs. These are ZK nodes.  | |
| OD_EVENT_PUBLISH_FAILURE_ACTIONS | A comma delimited list of event action types that should be published to kafka if request failed. The default value * enables all failure events to be published. Permissible values are access, authenticate, create, delete, list, undelete, unknown, update, zip. | * |
| OD_EVENT_PUBLISH_SUCCESS_ACTIONS | A comma delimited list of event action types that should be published to kafka if request succeeded. The default value * enables all success events to be published. Permissible values are access, authenticate, create, delete, list, undelete, unknown, update, zip. | * |
| OD_EVENT_TOPIC | The name of the topic for which events will be published to. | odrive-event |

**NOTE:** If both the Kafka broker and ZooKeeper address options are blank, Object Drive will not publish events.

# Models

## Global Event Model

This is the global event "envelope" that all services use in this framework

| Field | Data Type | Description |
| --- | --- | --- |
| eventId | GUID | A software-generated GUID, unique for every published event. |
| eventChain | []GUID | An array of GUIDs. Will be empty if the event is never enriched. |
| schemaVersion | string | Represents the major and minor version of the event schema, e.g. `1.0`. |
| originatorToken | []string | Identifiers, usually consisting of subject distinguished name from X509 certificates or resource strings for the originators of the event. This consists of end user and/or system users and/or system impersonators. | 
| eventType | string | A globally unique string identifying the source system, e.g. `object-drive-event`. |
| timestamp | datetime | A unix timestamp, numerically represented in JSON. |
| xForwardedForIp | string | The IP address of the end user. Required for auditing. | 
| systemIp | string | The IP address of the system that emitted the event. |
| action | string | A string identifying the action. Values used are `access`, `authenticate`, `create`, `delete`, `list`, `undelete`, `update`, `zip`. |
| payload | ObjectDriveEvent | A custom payload structure with more information |

## Object Drive Event model

The Global Event Model emitted from Object Drive includes a payload that conforms to the following

| Field | Data Type | Since | Description |
| --- | --- | --- | --- |
| audit | Audit.AuditEvent | 1.0.1.13 | Embeds the ICS 500-27 schema necessary for transforming for audit delivery. |
| object_id | string | 1.0.1.13 | The unique identifier of the object hex encoded to a string. | 
| change_token | string | 1.0.1.13 | A change token assigned based upon hash of id, change count, and modified date. |
| stream_update | bool | 1.0.1.13 | Indicates whether this change includes an update to the content stream. |
| user_dn | string | 1.0.1.13 | The distinguished name of the user that triggered the action. |
| session_id | string | 1.0.1.13 | A random string generated for each http request. |
| createdBy | string | 1.0.12 | A representation of the user that created the referenced object. |
| modifiedBy | string | 1.0.12 | A representation of the user that modified the referenced object, triggering the action. | 
| deletedBy | string | 1.0.12 | A representation of the user that deleted the referenced object. | 
| changeCount | number | 1.0.12 | The number of times the object has been modified. |
| ownedBy | string | 1.0.12 | A representation of the current owner of the referenced object. |
| objectType | string | 1.0.12 | The name of the object type for this object. (e.g. `File`, `Folder`) |
| name | string | 1.0.12 | The name given to the object. (i.e., it's filename) |
| description | string | 1.0.12 | An abstract of the object or its contents. |
| parentId | string | 1.0.12 | A hex encoding of the object parent's unique identifier. |
| contentType | string | 1.0.12 | The mime-type for the object contents. |
| contentSize | number | 1.0.12 | The size of the content stream for the referenced object in bytes. |
| contentHash | string | 1.0.12 | A sha256 hash of the content stream | 
| containsUSPersonsData | string | 1.0.12 | Indicates whether the referenced object contains US Persons Data. Expected values are `Unknown`, `Yes`, and `No`. |
| exemptFromFOIA | string | 1.0.12 | Indicates whether the referenced object is exempt from Freedom of Information Act requests. Expected values are `Unknown`, `Yes`, and `No`. |
| breadcrumbs | []Breadcrumb | 1.0.12 | An array of breadcrumbs giving information about the chain of ancestors to the root for the referenced object. |

## Breadcrumb

A breadcrumb provides a minimal amount of reference data for linking from an object identifier to its parent.  An array of breadcrumbs can provide linkage from an object up to the root.

| Field | Data Type | Since | Description |
| --- | --- | --- | --- |
| id | string | 1.0.12 | The unique identifier of the object hex encoded to a string. |
| parentId | string | 1.0.12 | A hex encoding of the object parent's unique identifier. If this value as empty, it indicates that the parent is the root of the tree. |
| name | string | 1.0.12 | The name given to the object referenced by the id. |

# Audit Event Models

Structured models for holding Audit information in support of ICS 500-27

## Audit.ActionInitiator

| Field | Data Type |
| --- | --- | 
| identity_type | string |
| value | string |

## Audit.ActionLocation

| Field | Data Type |
| --- | --- | 
| identifier | string |
| value | string |

## Audit.ActionTarget

| Field | Data Type |
| --- | --- | 
| identity_type | string |
| value | string |
| acm | Acm.Acm |

## Audit.AuditEvent

| Field | Data Type |
| --- | --- | 
| type | string |
| action | string |
| action_initiator | Audit.ActionInitiator |
| action_locations | []Audit.ActionLocation |
| action_mode | string |
| action_result | string |
| action_target_messages | []string |
| action_target_versions | []string |
| action_targets | []Audit.ActionTarget |
| additional_info | map[string]string |
| agency | string |
| countries_of_citizenship | []string |
| created_on | string |
| creator | Audit.Creator |
| edh | Edh |
| responsible_entity | Audit.ResponsibleEntity |
| home_agency | string |
| id | Id |
| network | string |
| ntp_info | Audit.NTPInfo |
| resources | []Audit.Resource |
| cross_domain | bool |
| relevancy | string | 
| query_string | string |
| query_type | string |
| results | []Audit.Result |
| filters | []Audit.Filter |
| modified_pair_list | []Audit.ModifiedResourcePair |
| collaboration_features | []Audit.CollaborationFeature |
| session_ids | []string |
| workflow_id | string |
| authorization_services | []string |
| authorization_service_time_period | string | 
| device | Audit.Device |

## Audit.CollaborationFeature

| Field | Data Type |
| --- | --- | 
| type | string |
| value | string |

## Audit.Creator

| identity_type | string |
| value | string |

## Audit.Device

| location | string |
| type | string |

## Audit.Edh

| acm | Acm.Acm |
| guide | Audit.Guide |
| responsible_entity | Audit.ResponsibleEntity |
| security | Audit.Security |

## Audit.Filter

| Field | Data Type |
| --- | --- | 
| filter | string |

## Audit.Guide

| Field | Data Type |
| --- | --- | 
| prefix | string |
| number | string |

## Audit.Id

| Field | Data Type |
| --- | --- | 
| identifier | string |
| value | string | 

## Audit.MalwareService

| Field | Data Type |
| --- | --- | 
| malware_service | string |

## Audit.ModifiedResourcePair

| Field | Data Type |
| --- | --- | 
| original | Audit.Resource |
| modified | Audit.Resource |

## Audit.NTPInfo

| Field | Data Type |
| --- | --- | 
| identity_type | string |
| last_update | string |
| server | string | 

## Audit.Resource

| Field | Data Type |
| --- | --- | 
| object_type | string |
| name | Audit.ResourceName |
| location | string |
| size | number |
| sub_type | string |
| type | string |
| role | string |
| malware_check | bool | 
| malware_check_status | string |
| content | Audit.ResourceContent |
| description | Audit.ResourceDescription |
| identifier | string |
| malware_services | []Audit.MalwareService |
| parent | Audit.ResourceParent |
| acm | Acm.Acm |

## Audit.ResourceContent

| Field | Data Type |
| --- | --- | 
| content | string |
| acm | Acm.Acm |
| send | bool | 

## Audit.ResourceDescription

| Field | Data Type |
| --- | --- | 
| content | string |
| acm | Acm.Acm |

## Audit.ResourceName

| Field | Data Type |
| --- | --- | 
| title | string |
| acm | Acm.Acm | 

## Audit.ResourceParent

| Field | Data Type |
| --- | --- | 
| type | string |
| sub_type | string |
| location | string |
| identifier | string | 

## Audit.ResponsibleEntity

| Field | Data Type |
| --- | --- | 
| country | string |
| organization | string |

## Audit.Result

| Field | Data Type |
| --- | --- | 
| type | string |
| size | string | 
| value | string |

## Audit.Security

| Field | Data Type |
| --- | --- | 
| owner_producer | string |
| classified_by | string |
| classification_reason | string |
| declass_date | string |
| derived_from | string |

# ACM Models

## Acm.Accm

| Field | Data Type |
| --- | --- | 
| coi | string |
| disp_nm | string |
| coi_ctrls | []Acm.CoiControl |

## Acm.Acm

| Field | Data Type |
| --- | --- | 
| version | string |
| classif | string |
| owner_prod | []string |
| atom_energy | []string |
| non_us_ctrls | []string |
| non_ic | []string |
| sci_ctrls | []string |
| sar_id | []string |
| disponly_to | []string |
| disp_only | string |
| dissem_ctrls | []string |
| rel_to | []string |
| classif_rsn | string |
| classif_by | string |
| deriv_from | string |
| complies_with | string |
| classif_dt | string |
| declass_dt | string |
| declass_event | string |
| declass_ex | string |
| deriv_class_by | string |
| des_version | string |
| notice_rsn | string |
| poc | string |
| rsrc_elem | string |
| compil_rsn | string |
| ex_from_rollup | string |
| fgi_open | []string |
| fgi_protect | []string |
| portion | string |
| banner | string |
| dissem_countries | []string |
| oc_attribs | []Acm.OCAttribs |
| accms | []Acm.Accm |
| macs | []Acm.Accm |
| assigned_controls | Acm.AssignedControls |
| share | Acm.Share |
| f_clearance | []string |
| f_classif_rank | []string |
| f_sci_ctrls | []string |
| f_accms | []string |
| f_oc_org | []string |
| f_regions | []string |
| f_missions | []string |
| f_share | []string |
| f_sar_id | []string |
| f_atom_energy | []string |
| f_macs | []string |

## Acm.AssignedControls

| Field | Data Type |
| --- | --- | 
| coi | []string |
| coi_ctrls | []string |

## Acm.CoiControl

| Field | Data Type |
| --- | --- | 
| coi_ctrl | string |
| disp_nm | string |

## Acm.OCAttribs

| Field | Data Type |
| --- | --- | 
| missions | []string |
| regions | []string |
| orgs | []string |

## Acm.Project 

| Field | Data Type |
| --- | --- | 
| disp_nm | string |
| groups | []string |

## Acm.Share

| Field | Data Type |
| --- | --- | 
| users | []string |
| projects | map[string]Acm.Project |