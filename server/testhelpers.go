package server

import (
	"fmt"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util"
)

// ValidACMs
const (
	// TODO: add "share" and set with users or project/groups
	ValidACMUnclassified = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	ValidACMUnclassifiedEmptyDissemCountries = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":[""],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	ValidACMUnclassifiedEmptyDissemCountriesEmptyFShare = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":[""],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[""],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	// TODO: Need to figure out what the actual result is and put into f_share
	ValidACMUnclassifiedWithFShare = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":["cntesttester01oupeopleoudaeouchimeraou_s_governmentcus"],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	ValidACMUnclassifiedFOUO = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":["FOUO"],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U//FOUO","banner":"UNCLASSIFIED//FOUO","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	ValidACMTopSecretSITK = `{"version":"2.1.0","classif":"TS","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":["si","tk"],"disponly_to":[""],"dissem_ctrls":[""],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"TS//SI/TK","banner":"TOP SECRET//SI/TK","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["ts"],"f_sci_ctrls":["si","tk"],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	ValidACMUnclassifiedFOUOSharedToTester01 = `{"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"f_clearance":["u"],"f_share":["cntesttester01oupeopleoudaeouchimeraou_s_governmentcus"],"portion":"U//FOUO","share":{"users":["cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`

	ValidACMUnclassifiedFOUOSharedToTester10 = `{"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"f_clearance":["u"],"f_share":["cntesttester01oupeopleoudaeouchimeraou_s_governmentcus"],"portion":"U//FOUO","share":{"users":["cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`

	ValidACMTopSecretSharedToTester01 = `{"fgi_open":[],"rel_to":[],"sci_ctrls":[],"owner_prod":[],"portion":"TS","disp_only":"","disponly_to":[],"banner":"TOP SECRET","non_ic":[],"classif":"TS","atom_energy":[],"dissem_ctrls":[],"sar_id":[],"version":"2.1.0","fgi_protect":[],"share":{"users":["cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"f_clearance":[],"dissem_countries":["USA"],"isShared":true}
`
	ValidACMUnclassifiedFOUOSharedToTester01And02 = `{"accms":[],"atom_energy":[],"banner":"UNCLASSIFIED//FOUO","classif":"U","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":["cntesttester01oupeopleoudaeouchimeraou_s_governmentcus","cntesttester02oupeopleoudaeouchimeraou_s_governmentcus"],"fgi_open":[],"fgi_protect":[],"macs":[],"non_ic":[],"oc_attribs":[{"missions":[],"orgs":[],"regions":[]}],"owner_prod":[],"portion":"U//FOUO","rel_to":[],"sar_id":[],"sci_ctrls":[],"share":{"users":["cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us","cn=test tester02,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`
	ValidACMUnclassifiedFOUOSharedToTester01And10 = `{"accms":[],"atom_energy":[],"banner":"UNCLASSIFIED//FOUO","classif":"U","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":["cntesttester01oupeopleoudaeouchimeraou_s_governmentcus","cntesttester10oupeopleoudaeouchimeraou_s_governmentcus"],"fgi_open":[],"fgi_protect":[],"macs":[],"non_ic":[],"oc_attribs":[{"missions":[],"orgs":[],"regions":[]}],"owner_prod":[],"portion":"U//FOUO","rel_to":[],"sar_id":[],"sci_ctrls":[],"share":{"users":["cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us","cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`

	//This is a plausible user that hasn't yet had a visit to odrive yet
	ValidACMUnclassifiedFOUOSharedToTester11    = `{"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"portion":"U//FOUO","share":{"users":["cn=test tester11,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`
	ValidACMUnclassifiedFOUOSharedToTester12    = `{"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"portion":"U//FOUO","share":{"users":["cn=test tester12,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`
	ValidACMUnclassifiedFOUOSharedToTester13    = `{"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"portion":"U//FOUO","share":{"users":["cn=test tester13,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`
	Tester10DN                                  = "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	Tester11DN                                  = "cn=test tester11,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	Tester12DN                                  = "cn=test tester12,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	Tester13DN                                  = "cn=test tester13,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	ValidACMUnclassifiedFOUOSharedToDAOTester11 = `{"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"portion":"U//FOUO","share":{"users":["CN=[DAOTEST]test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US","CN=[DAOTEST]test tester'1, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"]},"version":"2.1.0"}`
)

// Snippets
const (
	SnippetTP01 = "{\"f_macs\":\"{\\\"field\\\":\\\"f_macs\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"tide\\\",\\\"bir\\\",\\\"watchdog\\\"]}\",\"f_oc_org\":\"{\\\"field\\\":\\\"f_oc_org\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"dia\\\"]}\",\"f_accms\":\"{\\\"field\\\":\\\"f_accms\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[]}\",\"f_sap\":\"{\\\"field\\\":\\\"f_sar_id\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_clearance\":\"{\\\"field\\\":\\\"f_clearance\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"u\\\"]}\",\"f_regions\":\"{\\\"field\\\":\\\"f_regions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_missions\":\"{\\\"field\\\":\\\"f_missions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_share\":\"{\\\"field\\\":\\\"f_share\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"cntesttester01oupeopleoudaeouchimeraou_s_governmentcus\\\",\\\"cusou_s_governmentouchimeraoudaeoupeoplecntesttester01\\\"]}\",\"f_aea\":\"{\\\"field\\\":\\\"f_atom_energy\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_sci_ctrls\":\"{\\\"field\\\":\\\"f_sci_ctrls\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"g\\\",\\\"hcs\\\",\\\"hcs_p\\\",\\\"kdk\\\",\\\"rsv\\\",\\\"si\\\",\\\"tk\\\"]}\",\"dissem_countries\":\"{\\\"field\\\":\\\"dissem_countries\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"USA\\\"]}\"}"
	SnippetTP02 = "{\"f_macs\":\"{\\\"field\\\":\\\"f_macs\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"tide\\\",\\\"bir\\\",\\\"watchdog\\\"]}\",\"f_oc_org\":\"{\\\"field\\\":\\\"f_oc_org\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"dia\\\"]}\",\"f_accms\":\"{\\\"field\\\":\\\"f_accms\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[]}\",\"f_sap\":\"{\\\"field\\\":\\\"f_sar_id\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_clearance\":\"{\\\"field\\\":\\\"f_clearance\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"u\\\"]}\",\"f_regions\":\"{\\\"field\\\":\\\"f_regions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_missions\":\"{\\\"field\\\":\\\"f_missions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_share\":\"{\\\"field\\\":\\\"f_share\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"cntesttester02oupeopleoudaeouchimeraou_s_governmentcus\\\",\\\"cusou_s_governmentouchimeraoudaeoupeoplecntesttester02\\\"]}\",\"f_aea\":\"{\\\"field\\\":\\\"f_atom_energy\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_sci_ctrls\":\"{\\\"field\\\":\\\"f_sci_ctrls\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"g\\\",\\\"hcs\\\",\\\"hcs_p\\\",\\\"kdk\\\",\\\"rsv\\\",\\\"si\\\",\\\"tk\\\"]}\",\"dissem_countries\":\"{\\\"field\\\":\\\"dissem_countries\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"USA\\\"]}\"}"
	SnippetTP10 = "{\"f_macs\":\"{\\\"field\\\":\\\"f_macs\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"tide\\\",\\\"bir\\\",\\\"watchdog\\\"]}\",\"f_oc_org\":\"{\\\"field\\\":\\\"f_oc_org\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"dia\\\"]}\",\"f_accms\":\"{\\\"field\\\":\\\"f_accms\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[]}\",\"f_sap\":\"{\\\"field\\\":\\\"f_sar_id\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_clearance\":\"{\\\"field\\\":\\\"f_clearance\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"ts\\\",\\\"s\\\",\\\"c\\\",\\\"u\\\"]}\",\"f_regions\":\"{\\\"field\\\":\\\"f_regions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_missions\":\"{\\\"field\\\":\\\"f_missions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_share\":\"{\\\"field\\\":\\\"f_share\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"dctc_up2_dctc_manager\\\",\\\"dctc_up2_dctc_supervisor\\\",\\\"dctc_up2_dctc\\\",\\\"dctc_up2_aprc_supervisor\\\",\\\"dctc_up2_aprc_manager\\\",\\\"dctc_up2_aprc\\\",\\\"dctc_up2_administrator\\\",\\\"dctc_watchdog_fle\\\",\\\"dctc_watchdog_sle\\\",\\\"dctc_watchdog_fdo\\\",\\\"dctc_watchdog_user\\\",\\\"dctc_watchdog_administrator\\\",\\\"cntesttester10oupeopleoudaeouchimeraou_s_governmentcus\\\",\\\"cusou_s_governmentouchimeraoudaeoupeoplecntesttester10\\\"]}\",\"f_aea\":\"{\\\"field\\\":\\\"f_atom_energy\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_sci_ctrls\":\"{\\\"field\\\":\\\"f_sci_ctrls\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"hcs_p\\\",\\\"kdk\\\",\\\"rsv\\\"]}\",\"dissem_countries\":\"{\\\"field\\\":\\\"dissem_countries\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"USA\\\"]}\"}"
)

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
