package mapping

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/services/audit/generated/acm_thrift"
	"decipher.com/object-drive-server/services/audit/generated/components_thrift"
	"decipher.com/object-drive-server/services/audit/generated/events_thrift"
)

// ODObjectToAuditorResource converts ODObject to an audit Resource. A pointer
// to the existing event is required to set up fields that duplicate some data,
// such as Location-type fields.
func ODObjectToAuditorResource(
	o *models.ODObject, event *events_thrift.AuditEvent) (*components_thrift.Resource, error) {

	var res components_thrift.Resource

	res.ObjectType = determineObjectType(event)

	converted, err := RawAcmToThriftAcm(o.RawAcm.String)
	if err != nil {
		return nil, err
	}

	res.Name = &components_thrift.ResourceName{
		Title: stringPtr(o.Name),
		Acm:   converted,
	}

	// This might need to be a URL. Using ID here.
	res.Location = stringPtr(o.ODID.String())
	res.Size = nullInt64ToInt32Ptr(o.ContentSize)
	res.SubType = nil // no equivalent field
	res.Type = determineType(o.Name)
	res.Role = stringPtr("GENERAL_USER")
	res.MalwareCheck = boolPtr(false)
	// TODO these might need to be populated.
	res.MalwareCheckStatus = nil
	res.MalwareServices = nil
	res.Content = nil
	res.Description = &components_thrift.ResourceDescription{
		Description: stringPtr(o.Description.String),
		Acm:         converted,
	}
	// Using ID again.
	res.Identifier = stringPtr(o.ODID.String())
	res.MalwareServices = nil
	res.Parent = nil
	res.Acm = converted

	return &res, nil
}

// Lucas says: sorry, I just didn't feel like reimplementing this here since audit isn't used yet.
// // ODObjectACMToAuditorACM ...
// func ODObjectACMToAuditorACM(from acm.ACM) *acm_thrift.Acm {
// 	var to acm_thrift.Acm
// 	// NOTE We are setting pointers here. If from.Foo is empty string,
// 	// a pointer to empty string will be set. A nil will not be set
// 	// unless stringPtrOrNil is used.
// 	to.Version = stringPtr(from.Version)
// 	to.Classif = from.Classif
// 	to.OwnerProd = from.OwnerProducer
// 	to.AtomEnergy = from.AtomEnergy
// 	to.NonUsCtrls = make([]string, 0) // no equivalent field?
// 	to.NonIc = from.NonICMarkings
// 	to.SciCtrls = from.SCIControls
// 	to.SarId = from.SpecialAccessRequiredID
// 	to.DisponlyTo = from.DisplayOnlyTo
// 	to.DispOnly = stringPtr(from.DisplayOnly)
// 	to.DissemCtrls = from.DisseminationControls
// 	to.RelTo = from.ReleasableTo
// 	to.ClassifRsn = stringPtr("")   // no equivalent field?
// 	to.ClassifBy = stringPtr("")    // no equivalent field?
// 	to.DerivFrom = stringPtr("")    // no equivalent field?
// 	to.CompliesWith = stringPtr("") // no equivalent field?
// 	to.ClassifDt = stringPtr("")    // no equivalent field?
// 	to.DeclassDt = stringPtr("")    // no equivalent field?
// 	to.DeclassEvent = stringPtr("") // no equivalent field?
// 	to.DeclassEx = stringPtr("")    // no equivalent field?
// 	to.DerivClassBy = stringPtr("") // no equivalent field?
// 	to.DesVersion = stringPtr("")   // no equivalent field?
// 	to.NoticeRsn = stringPtr("")    // no equivalent field?
// 	to.Poc = stringPtr("")          // no equivalent field?
// 	to.RsrcElem = stringPtr("")     // no equivalent field?
// 	to.CompilRsn = stringPtr("")    // no equivalent field?
// 	to.ExFromRollup = stringPtr("") // no equivalent field?
// 	to.FgiOpen = from.FGIOpen
// 	to.FgiProtect = from.FGIProtect
// 	to.Portion = stringPtr(from.PortionMark)
// 	to.Banner = stringPtr(from.OverallBanner)
// 	to.DissemCountries = from.DisseminationCountries
// 	to.OcAttribs = mapOCAttributesSlice(from.OCAttributes)
// 	to.Accms = mapACCMsSlice(from.ACCMs)
// 	to.Macs = mapACCMsSlice(from.MACs)
// 	to.AssignedControls = nil // no equivalent field?
// 	to.Share = nil            // no equivalent field?
// 	to.FClearance = from.FlatClearance
// 	to.FClassifRank = make([]string, 0) // no equivalent field?
// 	to.FSciCtrls = from.FlatSCIControls
// 	to.FAccms = from.FlatACCMs
// 	to.FOcOrg = from.FlatOCOrgs
// 	to.FRegions = from.FlatRegions
// 	to.FMissions = from.FlatMissions
// 	to.FShare = from.FlatShare
// 	to.FSarId = make([]string, 0) // no equivalent field?
// 	to.FAtomEnergy = from.FlatAtomEnergy
// 	to.FMacs = from.FlatMACs
// 	return &to
// }

func accmFromJSON(from string) *acm_thrift.Accm {

	var to acm_thrift.Accm

	err := json.NewDecoder(bytes.NewBufferString(from)).Decode(&to)
	if err != nil {
		log.Printf("Could not decode into acm_thrift.Acm: %s\n", from)
		return nil
	}
	return &to
}

func mapACCMsSlice(from []string) []*acm_thrift.Accm {

	var result []*acm_thrift.Accm
	for _, accm := range from {
		decoded := accmFromJSON(accm)
		result = append(result, decoded)
	}
	return result
}

// Lucas says: sorry, I just didn't feel like reimplementing this here since audit isn't used yet.
// func mapOCAttributesSlice(from []acm.OCAttributeInfo) []*acm_thrift.OCAttribs {

// 	var result []*acm_thrift.OCAttribs
// 	for _, oc := range from {
// 		converted := OCAttributeInfoToOcAttribs(oc)
// 		result = append(result, converted)
// 	}
// 	return result
// }

// Lucas says: sorry, I just didn't feel like reimplementing this here since audit isn't used yet.
// func OCAttributeInfoToOcAttribs(from acm.OCAttributeInfo) *acm_thrift.OCAttribs {

// 	var to acm_thrift.OCAttribs
// 	// handle nil case?
// 	to.Missions = from.Missions
// 	to.Orgs = from.Organizations
// 	to.Regions = from.Regions
// 	return &to
// }

func RawAcmToThriftAcm(rawAcm string) (*acm_thrift.Acm, error) {
	// Lucas says: sorry, I just didn't feel like reimplementing this here since audit isn't used yet.
	return nil, fmt.Errorf("Not implemented")
}

func determineObjectType(e *events_thrift.AuditEvent) *string {
	if *e.Type == "EventModified" {
		return stringPtr("ResourceModified")
	} else {
		return stringPtr("Resource")
	}
}

func determineType(filename string) *string {
	// Resource.Type is enumerated, but most do not apply to odrive. See:
	// https://bedrock.363-283.io/services/audit/ics/1.0/2015-feb/audit_xml_documentation/2015-feb-schema-guide/0f4889ff-4421-460e-85f8-df37444d39a8.html#urn_us_gov_ic_audit_resourceType
	splitted := strings.Split(filename, ".")
	switch splitted[len(splitted)-1] {
	case "doc", "docx", "xls", "xlsx", "ppt", "pptx":
		return stringPtr("DOCUMENT")
	default:
		return stringPtr("FILE")
	}
}

// pointer utils
func boolPtr(b bool) *bool { return &b }
func defaultStringPtr(s string, defaultString string) *string {
	if s == "" {
		return stringPtr(defaultString)
	}
	return stringPtr(s)
}
func stringPtr(s string) *string { return &s }
func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func nullInt64ToInt32Ptr(i models.NullInt64) *int32 {
	if !i.Valid {
		return nil
	}
	cast := int32(i.Int64)
	return &cast

}
