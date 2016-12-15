package models

import "strings"

// ODAcmGrantee is the detailed fields of an Acm Share for an individual
// grantee, and associates to an object permission via the Grantee as a key
type ODAcmGrantee struct {
	// Grantee indicates the flattened representation of a user or group
	// referenced by a permission
	Grantee string `db:"grantee"`
	// ResourceString is the built up resource name as stored in the database
	ResourceString NullString `db:"resourceString"`
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

// String returns a serialized human readable representation of ODAcmGrantee
func (acmGrantee *ODAcmGrantee) String() string {
	return acmGrantee.ResourceName()
}

// ResourceName returns the built up resource name for a grantee based upon its composite parts
func (acmGrantee *ODAcmGrantee) ResourceName() string {
	if len(acmGrantee.DisplayName.String) > 0 {
		o := []string{}
		if len(acmGrantee.UserDistinguishedName.String) > 0 {
			o = append(o, "user")
			o = append(o, acmGrantee.UserDistinguishedName.String)
		} else if len(acmGrantee.GroupName.String) > 0 {
			o = append(o, "group")
			if len(acmGrantee.ProjectName.String) > 0 {
				o = append(o, acmGrantee.ProjectName.String)
				if len(acmGrantee.ProjectDisplayName.String) > 0 {
					o = append(o, acmGrantee.ProjectDisplayName.String)
				} else {
					o = append(o, acmGrantee.ProjectName.String)
				}
			}
			o = append(o, acmGrantee.GroupName.String)
		} else {
			o = append(o, "unknown")
		}
		o = append(o, acmGrantee.DisplayName.String)
		if len(o) > 0 {
			return strings.Join(o, "/")
		}
	}
	return ""
}

// NewODAcmGranteeFromResourceName instantiates an ODAcmGrantee object from parsing a resource string
func NewODAcmGranteeFromResourceName(resourceName string) ODAcmGrantee {
	if strings.HasPrefix(resourceName, "user/") {
		return newODAcmGranteeFromUserResource(resourceName)
	}
	if strings.HasPrefix(resourceName, "group/") {
		return newODAcmGranteeFromGroupResource(resourceName)
	}
	return ODAcmGrantee{}
}

func newODAcmGranteeFromUserResource(resource string) ODAcmGrantee {
	parts := strings.Split(strings.Replace(resource, "user/", "", 1), "/")
	grantee := ODAcmGrantee{}
	if len(parts) > 0 {
		grantee.UserDistinguishedName = ToNullString(parts[0])
		grantee.Grantee = AACFlatten(parts[0])
		grantee.DisplayName = ToNullString(parts[0])
		if len(parts) > 1 {
			grantee.DisplayName = ToNullString(parts[1])
		}
	}
	return grantee
}

func newODAcmGranteeFromGroupResource(resource string) ODAcmGrantee {
	parts := strings.Split(strings.Replace(resource, "group/", "", 1), "/")
	grantee := ODAcmGrantee{}
	switch len(parts) {
	case 1:
		grantee.GroupName = ToNullString(parts[0])
	case 2:
		grantee.GroupName = ToNullString(parts[1])
	default:
		grantee.ProjectName = ToNullString(parts[0])
		grantee.ProjectDisplayName = ToNullString(parts[1])
		grantee.GroupName = ToNullString(parts[2])
	}
	if len(grantee.ProjectDisplayName.String) > 0 {
		grantee.Grantee = AACFlatten(strings.TrimSpace(grantee.ProjectDisplayName.String + "_" + grantee.GroupName.String))
	} else {
		grantee.Grantee = AACFlatten(grantee.GroupName.String)
	}
	grantee.DisplayName = ToNullString(strings.TrimSpace(grantee.ProjectDisplayName.String + " " + grantee.GroupName.String))

	return grantee
}
