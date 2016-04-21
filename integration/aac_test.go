package integration

import (
	"log"
	"path/filepath"
	"testing"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models/acm"
	aac "decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/util"
	t2 "github.com/samuel/go-thrift/thrift"
)

var userDN1 = "CN=Holmes Jonathan,OU=People,OU=Bedrock,OU=Six 3 Systems,O=U.S. Government,C=US"
var shareType1 = "public"
var shareType2 = "private"
var shareType3 = "other"
var share1 = ""
var share2 = "{\"users\":[\"b86b59b6d96b467db3dd2dafee2cb3d7\"],\"projects\":null}"
var share3 = "{\"users\":[\"b86b59b6d96b467db3dd2dafee2cb3d7\", \"4838db45d94343d888322f012b918de0\"],\"projects\":null}"

var acmPartial1 = "{ \"path\": \"\", \"classif\":\"TS\", \"sci_ctrls\":[ \"HCS\", \"SI-G\", \"TK\" ], \"dissem_ctrls\":[ \"OC\" ], \"dissem_countries\":[ \"USA\" ], \"oc_attribs\":[ { \"orgs\":[ \"dia\" ] } ] }"
var acmComplete1 = "{ \"version\":\"2.1.0\", \"classif\":\"TS\", \"owner_prod\":[], \"atom_energy\":[], \"sar_id\":[], \"sci_ctrls\":[ \"HCS\", \"SI-G\", \"TK\" ], \"disponly_to\":[ \"\" ], \"dissem_ctrls\":[ \"OC\" ], \"non_ic\":[], \"rel_to\":[], \"fgi_open\":[], \"fgi_protect\":[], \"portion\":\"TS//HCS/SI-G/TK//OC\", \"banner\":\"TOP SECRET//HCS/SI-G/TK//ORCON\", \"dissem_countries\":[ \"USA\" ], \"accms\":[], \"macs\":[], \"oc_attribs\":[ { \"orgs\":[ \"dia\" ], \"missions\":[], \"regions\":[] } ], \"f_clearance\":[ \"ts\" ], \"f_sci_ctrls\":[ \"hcs\", \"si_g\", \"tk\" ], \"f_accms\":[], \"f_oc_org\":[ \"dia\", \"dni\" ], \"f_regions\":[], \"f_missions\":[], \"f_share\":[], \"f_atom_energy\":[], \"f_macs\":[], \"disp_only\":\"\" }"
var acmComplete2 = "{\"version\":\"2.1.0\",\"classif\":\"S\",\"owner_prod\":[],\"atom_energy\":[],\"sar_id\":[],\"sci_ctrls\":[],\"disponly_to\":[\"\"],\"dissem_ctrls\":[\"NF\"],\"non_ic\":[],\"rel_to\":[],\"fgi_open\":[],\"fgi_protect\":[],\"portion\":\"S//NF\",\"banner\":\"SECRET//NOFORN\",\"dissem_countries\":[\"USA\"],\"accms\":[],\"macs\":[],\"oc_attribs\":[{\"orgs\":[],\"missions\":[],\"regions\":[]}],\"f_clearance\":[\"s\"],\"f_sci_ctrls\":[],\"f_accms\":[],\"f_oc_org\":[],\"f_regions\":[],\"f_missions\":[],\"f_share\":[],\"f_atom_energy\":[],\"f_macs\":[],\"disp_only\":\"\"}"
var acmComplete3 = "{\"version\":\"2.1.0\",\"classif\":\"C\",\"owner_prod\":[],\"atom_energy\":[],\"sar_id\":[],\"sci_ctrls\":[],\"disponly_to\":[\"\"],\"dissem_ctrls\":[],\"non_ic\":[],\"rel_to\":[],\"fgi_open\":[],\"fgi_protect\":[],\"portion\":\"C\",\"banner\":\"CONFIDENTIAL\",\"dissem_countries\":[\"USA\"],\"f_clearance\":[\"c\"],\"f_sci_ctrls\":[],\"f_accms\":[],\"f_oc_org\":[],\"f_regions\":[],\"f_missions\":[],\"f_share\":[],\"f_atom_energy\":[],\"f_macs\":[],\"disp_only\":\"\"}"
var acmComplete4 = "{\"version\":\"2.1.0\",\"classif\":\"U\",\"owner_prod\":[],\"atom_energy\":[],\"sar_id\":[],\"sci_ctrls\":[],\"disponly_to\":[\"\"],\"dissem_ctrls\":[\"FOUO\"],\"non_ic\":[],\"rel_to\":[],\"fgi_open\":[],\"fgi_protect\":[],\"portion\":\"U//FOUO\",\"banner\":\"UNCLASSIFIED//FOUO\",\"dissem_countries\":[\"USA\"],\"accms\":[],\"macs\":[],\"oc_attribs\":[{\"orgs\":[],\"missions\":[],\"regions\":[]}],\"f_clearance\":[\"u\"],\"f_sci_ctrls\":[],\"f_accms\":[],\"f_oc_org\":[],\"f_regions\":[],\"f_missions\":[],\"f_share\":[],\"f_atom_energy\":[],\"f_macs\":[],\"disp_only\":\"\"}"

var snippetType = "ES"

// This package level var will be populated by init()
var aacClient = aac.AacServiceClient{}

func DontRun() bool {
	return testing.Short() || config.StandaloneMode
}

func TestMain(m *testing.M) {
	if DontRun() {
		return
	} else {
		trustPath := filepath.Join(config.CertsDir, "clients", "client.trust.pem")
		certPath := filepath.Join(config.CertsDir, "clients", "test_1.cert.pem")
		keyPath := filepath.Join(config.CertsDir, "clients", "test_1.key.pem")

		dialOpts := &config.OpenSSLDialOptions{}
		dialOpts.SetInsecureSkipHostVerification()
		conn, err := config.NewOpenSSLTransport(
			trustPath, certPath, keyPath, "twl-server-generic2", "9093", dialOpts)
		if err != nil {
			log.Fatal(err)
		}
		trns := t2.NewTransport(t2.NewFramedReadWriteCloser(conn, 0), t2.BinaryProtocol)
		client := t2.NewClient(trns, true)
		aacClient = aac.AacServiceClient{Client: client}
		m.Run()
	}
}

func TestCheckAccess(t *testing.T) {

	if DontRun() {
		t.Skip("Skipping as integration test.")
	}

	tokenType := "pki_dias"
	acmComplete := "{ \"version\":\"2.1.0\", \"classif\":\"TS\", \"owner_prod\":[], \"atom_energy\":[], \"sar_id\":[], \"sci_ctrls\":[ \"HCS\", \"SI-G\", \"TK\" ], \"disponly_to\":[ \"\" ], \"dissem_ctrls\":[ \"OC\" ], \"non_ic\":[], \"rel_to\":[], \"fgi_open\":[], \"fgi_protect\":[], \"portion\":\"TS//HCS/SI-G/TK//OC\", \"banner\":\"TOP SECRET//HCS/SI-G/TK//ORCON\", \"dissem_countries\":[ \"USA\" ], \"accms\":[], \"macs\":[], \"oc_attribs\":[ { \"orgs\":[ \"dia\" ], \"missions\":[], \"regions\":[] } ], \"f_clearance\":[ \"ts\" ], \"f_sci_ctrls\":[ \"hcs\", \"si_g\", \"tk\" ], \"f_accms\":[], \"f_oc_org\":[ \"dia\", \"dni\" ], \"f_regions\":[], \"f_missions\":[], \"f_share\":[], \"f_atom_energy\":[], \"f_macs\":[], \"disp_only\":\"\" }"

	resp, err := aacClient.CheckAccess(userDN1, tokenType, acmComplete)

	if err != nil {
		t.Logf("Error calling CheckAccess(): %v \n", err)
		t.FailNow()
	}

	if resp.Success != true {
		t.Logf("Expected true, got %v \n", resp.Success)
		t.Log("Messages: ", resp.Messages)
		t.Fail()
	}

	if !resp.HasAccess {
		t.Logf("Expected resp.HasAccess to be true\n")
		t.Fail()
	}
}

func TestBuildAcm(t *testing.T) {

	if DontRun() {
		t.Skip("Skipping as integration test.")
	}

	// NOTE: This relies on a hacky find/replace in the generated code, changing
	// every instance of []byte to []int8. This is due to a bug in the Thrift lib
	// https://github.com/samuel/go-thrift/issues/84
	byteList, err := util.StringToInt8Slice("<ddms:title ism:classification='S'>Foo</ddms:title>")
	if err != nil {
		t.Logf("Could not convert string to []int8\n")
		t.FailNow()
	}
	propertiesMap := make(map[string]string)
	propertiesMap["bedrock.message.traffic.dia.orgs"] = "DIA DOD_DIA" // propertiesMap["bedrock.message.traffic.orcon.project"] = "DCTC"
	propertiesMap["bedrock.message.traffic.orcon.group"] = "All"
	resp, err := aacClient.BuildAcm(byteList, "XML", propertiesMap)

	if err != nil {
		t.Logf("Error from BuildAcm() method: %v \n", err)
		t.Fail()
	}

	if !resp.Success {
		t.Logf("Unexpected BuildAcm response: Success: %v\n", resp.Success)
		t.Fail()
	}

	// TODO: Check the validity of specific fields in AcmInfo objects returned.
}

func TestValidateAcm(t *testing.T) {

	if DontRun() {
		t.Skip("Skipping as integration test.")
	}

	resp, err := aacClient.ValidateAcm(acmPartial1)
	if err != nil {
		t.Logf("Error from ValidateAcm() method: %v \n", err)
		t.FailNow()
	}
	success := resp.Success
	if !success {
		t.Logf("Unexpected ValidateAcm response: Success: %v \n", resp.Success)
		t.Fail()
	}
	acmValid := resp.AcmValid
	if acmValid != false {
		// TODO should this be false?
		t.Logf("Unexpected ValidateAcm response: AcmValid: %v \n", resp.AcmValid)
		t.Fail()
	}
}

func TestPopulateAndValidateAcm(t *testing.T) {

	if DontRun() {
		t.Skip("Skipping as integration test.")
	}

	resp, err := aacClient.PopulateAndValidateAcm(acmComplete1)
	if err != nil {
		t.Logf("Error from remote call PopulateAndValidateAcm() method: %v \n", err)
		t.FailNow()
	}
	result := resp.AcmValid
	if !result {
		t.Logf("Unexpected PopulateAndValidateAcm response: AcmValid: %v \n", resp.AcmValid)
		t.Fail()
	}
	return
}

func TestCreateAcmFromBannerMarking(t *testing.T) {

	if DontRun() {
		t.Skip("Skipping as integration test.")
	}

	t.Skipf("CreateAcmFromBannerMarking() not implemented within service.\n")
}

func TestRollupAcms(t *testing.T) {

	if DontRun() {
		t.Skip("Skipping as integration test.")
	}

	acmList := []string{
		acmComplete1,
		acmComplete2,
		acmComplete3,
		acmComplete4,
	}

	resp, err := aacClient.RollupAcms(userDN1, acmList, shareType1, "")
	if err != nil {
		t.Logf("Error from remote call RollupAcms() method: %v \n", err)
		t.FailNow()
	}

	if !resp.Success {
		t.Logf("Error in RollupAcms() response. Expected Success: true. Got: %v \n", resp.Success)
		t.Fail()
	}

	if !resp.AcmValid {
		t.Logf("Error in RollupAcms() response. Expected AcmValid: true. Got %v \n", resp.AcmValid)
		t.Fail()
	}
	return
}

func TestCheckAccessAndPopulate(t *testing.T) {

	if DontRun() {
		t.Skip("Skipping as integration test.")
	}

	// TODO: Refactor to table-driven test with failure scenarios. Right now,
	// everything succeeds. This is not realistic.

	acmInfo1 := aac.AcmInfo{Path: "acm:path:1", Acm: acmComplete1, IncludeInRollup: true}
	acmInfo2 := aac.AcmInfo{Path: "acm:path:2", Acm: acmComplete2, IncludeInRollup: true}
	acmInfo3 := aac.AcmInfo{Path: "acm:path:3", Acm: acmComplete3, IncludeInRollup: true}
	acmInfo4 := aac.AcmInfo{Path: "acm:path:4", Acm: acmComplete4, IncludeInRollup: true}

	// The method requires a slice of pointers to AcmInfo structs
	acmInfoList := []*aac.AcmInfo{&acmInfo1, &acmInfo2, &acmInfo3, &acmInfo4}

	resp, err := aacClient.CheckAccessAndPopulate(
		userDN1, "pki_dias", acmInfoList, true, shareType1, "")

	if err != nil {
		t.Logf("Error from remote call CheckAccessAndPopulate() method: %v \n", err)
		t.FailNow()
	}

	if !resp.Success {
		t.Logf("Error in CheckAccessAndPopulate response. Expected Success: true. Got: %v \n", resp.Success)
		t.Fail()
	}

	if numResponses := len(resp.AcmResponseList); numResponses != 4 {
		t.Logf("Expected 4 AcmResponse objects in AcmResponseList. Got: %v", numResponses)
		t.Fail()
	}

	// Check the validity and access for each item in AcmResponseList
	for _, item := range resp.AcmResponseList {
		expectedValid := true
		expectedHasAccess := true
		if item.AcmValid != expectedValid {
			t.Logf("Expected AcmValid to be %v. Got: %v", expectedValid, item.AcmValid)
			t.Fail()
		}
		if item.HasAccess != expectedHasAccess {
			t.Logf("Expected HasAccess to be %v. Got: %v", expectedHasAccess, item.HasAccess)
			t.Fail()
		}
	}
}

func TestGetUserAttributes(t *testing.T) {

	if DontRun() {
		t.Skip("Skipping as integration test.")
	}

	resp, err := aacClient.GetUserAttributes(userDN1, "pki_dias", "")

	if err != nil {
		t.Logf("Error from remote call GetUserAttributes() method: %v \n", err)
		t.FailNow()
	}

	if !resp.Success {
		t.Logf("Error in GetUserAttributes response. Expected Success: true. Got: %v\n", resp.Success)
		t.Fail()
	}
}

func TestGetSnippets(t *testing.T) {

	if DontRun() {
		t.Skip("Skipping as integration test.")
	}

	resp, err := aacClient.GetSnippets(userDN1, "pki_dias", snippetType)

	if err != nil {
		t.Logf("Error from remote call GetSnippets() method: %v \n", err)
		t.FailNow()
	}

	if !resp.Success {
		t.Logf("Error in GetSnippet() response. Expected Success: true. Got: %v \n", resp.Success)
		t.Fail()
	}

	if len(resp.Snippets) <= 0 {
		t.Logf("Error in GetSnippet() response object: Snippet length should not be zero.")
		t.Fail()
	}

}

func TestGetShare(t *testing.T) {

	if DontRun() {
		t.Skip("Skipping as integration test.")
	}

	// NOTE: when using private AAC, use the userToken passed. For Chimera this
	// should use the GUID generated not the DN.  This is only for this method.
	resp, err := aacClient.GetShare(userDN1, "pki_dias", shareType3, share3)

	if err != nil {
		t.Logf("Error from remote call GetShare() method: %v \n", err)
		t.FailNow()
	}

	if !resp.Success {
		t.Logf("Error in GetShare() response. Expected Success: true. Got: %v \n", resp.Success)
		t.Fail()
	}

}

func TestGetSnippetsOdriveRaw(t *testing.T) {
	if DontRun() {
		t.Skip("Skipping as integration test.")
	}
	snippetType := "odrive-raw"

	if DontRun() {
		t.Skip("Skipping as integration test.")
	}

	resp, err := aacClient.GetSnippets(userDN1, "pki_dias", snippetType)

	if err != nil {
		t.Logf("Error from remote call GetSnippets() method: %v \n", err)
		t.FailNow()
	}

	if !resp.Success {
		t.Logf("Error in GetSnippet() response. Expected Success: true. Got: %v \n", resp.Success)
		t.Fail()
	}

	if len(resp.Snippets) <= 0 {
		t.Logf("Error in GetSnippet() response object: Snippet length should not be zero.")
		t.Fail()
	}
}

func TestParseSnippetOdriveRaw(t *testing.T) {
	if DontRun() {
		t.Skip("Skipping as integration test.")
	}
	snippetType := "odrive-raw"

	if DontRun() {
		t.Skip("Skipping as integration test.")
	}

	resp, err := aacClient.GetSnippets(userDN1, "pki_dias", snippetType)

	if err != nil {
		t.Logf("Error from remote call GetSnippets() method: %v \n", err)
		t.FailNow()
	}

	if !resp.Success {
		t.Logf("Error in GetSnippet() response. Expected Success: true. Got: %v \n", resp.Success)
		t.Fail()
	}

	t.Logf("Snippets returned: %s", resp.Snippets)

	odriveRawSnippetFields, err := acm.NewODriveRawSnippetFieldsFromSnippetResponse(resp.Snippets)
	if err != nil {
		t.Log(err.Error())
		t.Logf("There were %d snippets", len(odriveRawSnippetFields.Snippets))
		t.Fail()
	}
	t.Logf("Building sql")
	var sql string
	for _, rawFields := range odriveRawSnippetFields.Snippets {
		fieldName := "acm." + rawFields.FieldName
		switch rawFields.Treatment {
		case "disallow":
			if len(rawFields.Values) > 0 {
				for _, value := range rawFields.Values {
					sql += " and " + fieldName + " not like '%," + value + ",%'"
				}
			}
		case "allowed":
			if len(rawFields.Values) == 0 {
				sql += " and (" + fieldName + " is null OR " + fieldName + " = '')"
			} else {
				sql += " and ("
				for v, value := range rawFields.Values {
					if v > 0 {
						sql += " or "
					}
					sql += fieldName + " = '" + value + "'"
				}
				sql += ")"
			}
		default:
			t.Logf("Unhandled treatment type: %s", rawFields.Treatment)
			t.Fail()
		}
	}
	t.Logf("Generated sql: %s", sql)

}
