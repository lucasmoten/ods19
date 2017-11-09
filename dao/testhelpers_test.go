package dao_test

import (
	"fmt"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util"
)

var ValidACMUnclassified = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

// NewObjectWithPermissionsAndProperties creates a single minimally populated
// object with random properties and full permissions.
func NewObjectWithPermissionsAndProperties(username, objectType string) models.ODObject {

	var obj models.ODObject
	randomName, err := util.NewGUID()
	if err != nil {
		panic(err)
	}

	obj.Name = randomName
	obj.CreatedBy = username
	obj.TypeName.String, obj.TypeName.Valid = objectType, true
	obj.RawAcm.String = ValidACMUnclassified
	permissions := make([]models.ODObjectPermission, 1)
	permissions[0].Grantee = models.AACFlatten(obj.CreatedBy)
	permissions[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, obj.CreatedBy)
	permissions[0].AcmGrantee.Grantee = permissions[0].Grantee
	permissions[0].AcmGrantee.UserDistinguishedName.String = obj.CreatedBy
	permissions[0].AcmGrantee.UserDistinguishedName.Valid = true
	permissions[0].AllowCreate = true
	permissions[0].AllowRead = true
	permissions[0].AllowUpdate = true
	permissions[0].AllowDelete = true
	obj.Permissions = permissions
	properties := make([]models.ODObjectPropertyEx, 1)
	properties[0].Name = "Test Property for " + randomName
	properties[0].Value.String = "Property Val for " + randomName
	properties[0].Value.Valid = true
	properties[0].ClassificationPM.String = "UNCLASSIFIED"
	properties[0].ClassificationPM.Valid = true
	obj.Properties = properties

	return obj
}

// CreateParentChildObjectRelationship sets the ParentID of child to the ID of parent.
// If parent has no ID, a []byte GUID is generated.
func CreateParentChildObjectRelationship(parent, child models.ODObject) (models.ODObject, models.ODObject, error) {

	if len(parent.ID) == 0 {
		id, err := util.NewGUIDBytes()
		if err != nil {
			return parent, child, err
		}
		parent.ID = id
	}
	child.ParentID = parent.ID
	return parent, child, nil
}
