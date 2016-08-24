package protocol

// ObjectShare is the association of an object to one or more users and/or
// groups for specifying permissions that will either be granted or revoked.
// The referenced object is implicit in the URL of the request
type ObjectShare struct {

	// Share indicates the users, groups, or other identities for which the
	// permissions to an object will apply.
	//
	// An ACM compliant share may be expressed as an object. Example format:
	//  "share":{
	//     "users":[
	//        "cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
	//       ,"cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	//       ]
	//    ,"projects":{
	//        "jifct_twl":{
	//           "disp_nm":"JIFCT.TWL"
	//          ,"groups":[
	//              "SLE"
	//             ,"USER"
	//             ]
	//          }
	//       }
	//    }
	//
	Share interface{} `json:"share"`

	// AllowCreate indicates whether the users/groups in the share will be
	// granted permission to create child objects beneath the target of this
	// grant when adding permissions or revoking such when removing permissions
	AllowCreate bool `json:"allowCreate"`

	// AllowRead indicates whether the users/groups in the share will be
	// granted permission to read the object metadata and properties, object
	// stream or list its children when adding permissions or revoking such
	// when removing permissions
	AllowRead bool `json:"allowRead"`

	// AllowUpdate indicates whether the users/groups in the share will be
	// granted permission to make changes to the object metadata and properties
	// or its content stream when adding permissions or revoking such when
	// removing permissions
	AllowUpdate bool `json:"allowUpdate"`

	// AllowDelete indicates whether the users/groups in the share will be
	// granted permission to delete the object when adding permissions or
	// revoking such when removing permissions
	AllowDelete bool `json:"allowDelete"`

	// AllowShare indicates whether the users/groups in the share will be
	// granted permission to share the object to others when adding permissions
	// or revoking such capability when removing permissions
	AllowShare bool `json:"allowShare"`
}
