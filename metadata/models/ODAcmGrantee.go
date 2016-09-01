package models

// ODAcmGrantee is the detailed fields of an Acm Share for an individual
// grantee, and associates to an object permission via the Grantee as a key
type ODAcmGrantee struct {
	// Grantee indicates the flattened representation of a user or group
	// referenced by a permission
	Grantee string `db:"grantee"`
	// ProjectName contains the project key portion of an AcmShare if this
	// grantee represents a group
	ProjectName NullString `db:"projectName"`
	// ProjectDisplayName contains the disp_nm portion of an AcmShare if this
	// grantee represents a group
	ProjectDisplayName NullString `db:"projectDisplayName"`
	// GroupName contains the group value portion of an AcmShare if this
	// grantee represents a group
	GroupName NullString `db:"groupName"`
	// UserDistinguishedName contains a user value portion of an AcmShare
	// if this grantee represnts a user
	UserDistinguishedName NullString `db:"userDistinguishedName"`
	// DisplayName is an optional display representation of the user or
	// group for user interfaces
	DisplayName NullString `db:"displayName"`
}
