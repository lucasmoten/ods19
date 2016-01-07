package integration

import (
	"log"
	"testing"

	aac "decipher.com/oduploader/cmd/cryptotest/gen-go2/aac"
	"decipher.com/oduploader/config"
	t2 "github.com/samuel/go-thrift/thrift"
)

var userDN1 = "CN=Holmes Jonathan,OU=People,OU=Bedrock,OU=Six 3 Systems,O=U.S. Government,C=US"
var shareType1 = "public"
var shareType2 = "private"
var shareType3 = "other"
var acmPartial1 = "{ \"path\": \"\", \"classif\":\"TS\", \"sci_ctrls\":[ \"HCS\", \"SI-G\", \"TK\" ], \"dissem_ctrls\":[ \"OC\" ], \"dissem_countries\":[ \"USA\" ], \"oc_attribs\":[ { \"orgs\":[ \"dia\" ] } ] }"
var acmComplete1 = "{ \"version\":\"2.1.0\", \"classif\":\"TS\", \"owner_prod\":[], \"atom_energy\":[], \"sar_id\":[], \"sci_ctrls\":[ \"HCS\", \"SI-G\", \"TK\" ], \"disponly_to\":[ \"\" ], \"dissem_ctrls\":[ \"OC\" ], \"non_ic\":[], \"rel_to\":[], \"fgi_open\":[], \"fgi_protect\":[], \"portion\":\"TS//HCS/SI-G/TK//OC\", \"banner\":\"TOP SECRET//HCS/SI-G/TK//ORCON\", \"dissem_countries\":[ \"USA\" ], \"accms\":[], \"macs\":[], \"oc_attribs\":[ { \"orgs\":[ \"dia\" ], \"missions\":[], \"regions\":[] } ], \"f_clearance\":[ \"ts\" ], \"f_sci_ctrls\":[ \"hcs\", \"si_g\", \"tk\" ], \"f_accms\":[], \"f_oc_org\":[ \"dia\", \"dni\" ], \"f_regions\":[], \"f_missions\":[], \"f_share\":[], \"f_atom_energy\":[], \"f_macs\":[], \"disp_only\":\"\" }"
var acmComplete2 = "{\"version\":\"2.1.0\",\"classif\":\"S\",\"owner_prod\":[],\"atom_energy\":[],\"sar_id\":[],\"sci_ctrls\":[],\"disponly_to\":[\"\"],\"dissem_ctrls\":[\"NF\"],\"non_ic\":[],\"rel_to\":[],\"fgi_open\":[],\"fgi_protect\":[],\"portion\":\"S//NF\",\"banner\":\"SECRET//NOFORN\",\"dissem_countries\":[\"USA\"],\"accms\":[],\"macs\":[],\"oc_attribs\":[{\"orgs\":[],\"missions\":[],\"regions\":[]}],\"f_clearance\":[\"s\"],\"f_sci_ctrls\":[],\"f_accms\":[],\"f_oc_org\":[],\"f_regions\":[],\"f_missions\":[],\"f_share\":[],\"f_atom_energy\":[],\"f_macs\":[],\"disp_only\":\"\"}"
var acmComplete3 = "{\"version\":\"2.1.0\",\"classif\":\"C\",\"owner_prod\":[],\"atom_energy\":[],\"sar_id\":[],\"sci_ctrls\":[],\"disponly_to\":[\"\"],\"dissem_ctrls\":[],\"non_ic\":[],\"rel_to\":[],\"fgi_open\":[],\"fgi_protect\":[],\"portion\":\"C\",\"banner\":\"CONFIDENTIAL\",\"dissem_countries\":[\"USA\"],\"f_clearance\":[\"c\"],\"f_sci_ctrls\":[],\"f_accms\":[],\"f_oc_org\":[],\"f_regions\":[],\"f_missions\":[],\"f_share\":[],\"f_atom_energy\":[],\"f_macs\":[],\"disp_only\":\"\"}"
var acmComplete4 = "{\"version\":\"2.1.0\",\"classif\":\"U\",\"owner_prod\":[],\"atom_energy\":[],\"sar_id\":[],\"sci_ctrls\":[],\"disponly_to\":[\"\"],\"dissem_ctrls\":[\"FOUO\"],\"non_ic\":[],\"rel_to\":[],\"fgi_open\":[],\"fgi_protect\":[],\"portion\":\"U//FOUO\",\"banner\":\"UNCLASSIFIED//FOUO\",\"dissem_countries\":[\"USA\"],\"accms\":[],\"macs\":[],\"oc_attribs\":[{\"orgs\":[],\"missions\":[],\"regions\":[]}],\"f_clearance\":[\"u\"],\"f_sci_ctrls\":[],\"f_accms\":[],\"f_oc_org\":[],\"f_regions\":[],\"f_missions\":[],\"f_share\":[],\"f_atom_energy\":[],\"f_macs\":[],\"disp_only\":\"\"}"

// This package level var will be populated by init()
var aacClient = aac.AacServiceClient{}

func init() {

	conn, err := config.NewOpenSSLTransport()
	if err != nil {
		log.Fatal(err)
	}
	trns := t2.NewTransport(t2.NewFramedReadWriteCloser(conn, 0), t2.BinaryProtocol)
	client := t2.NewClient(trns, true)
	aacClient = aac.AacServiceClient{Client: client}
}

func TestCheckAccess(t *testing.T) {

	// checkAccess params
	tokenType := "pki_dias"
	acmComplete := "{ \"version\":\"2.1.0\", \"classif\":\"TS\", \"owner_prod\":[], \"atom_energy\":[], \"sar_id\":[], \"sci_ctrls\":[ \"HCS\", \"SI-G\", \"TK\" ], \"disponly_to\":[ \"\" ], \"dissem_ctrls\":[ \"OC\" ], \"non_ic\":[], \"rel_to\":[], \"fgi_open\":[], \"fgi_protect\":[], \"portion\":\"TS//HCS/SI-G/TK//OC\", \"banner\":\"TOP SECRET//HCS/SI-G/TK//ORCON\", \"dissem_countries\":[ \"USA\" ], \"accms\":[], \"macs\":[], \"oc_attribs\":[ { \"orgs\":[ \"dia\" ], \"missions\":[], \"regions\":[] } ], \"f_clearance\":[ \"ts\" ], \"f_sci_ctrls\":[ \"hcs\", \"si_g\", \"tk\" ], \"f_accms\":[], \"f_oc_org\":[ \"dia\", \"dni\" ], \"f_regions\":[], \"f_missions\":[], \"f_share\":[], \"f_atom_energy\":[], \"f_macs\":[], \"disp_only\":\"\" }"
	resp, err := aacClient.CheckAccess(userDN1, tokenType, acmComplete)

	if err != nil {
		log.Printf("Error calling CheckAccess(): %v", err)
	}
	log.Println("Reponse: ", resp)

	if resp.Success != true {
		log.Println("Expected true, got ", resp.Success)
		t.Fail()
	}
}

func TestBuildAcm(t *testing.T) {

	// TODO: this is currently broken. We need to pass []int8 instead of []byte
	// buildAcm params
	// byteList := []byte("<ddms:title ism:classification='S'>Foo</ddms:title>")
	// propertiesMap := make(map[string]string)
	// propertiesMap["bedrock.message.traffic.dia.orgs"] = "DIA DOD_DIA"
	// propertiesMap["bedrock.message.traffic.orcon.project"] = "DCTC"
	// propertiesMap["bedrock.message.traffic.orcon.group"] = "All"
	//resp, err := aacClient.BuildAcm(byteList, "XML", propertiesMap)
	return
}

func TestValidateAcm(t *testing.T) {
	resp, err := aacClient.ValidateAcm(acmPartial1)
	if err != nil {
		log.Printf("Error from ValidateAcm() method: %v", err)
		t.Fail()
	}
	success := resp.Success
	if !success {
		log.Printf("Unexpected ValidateAcm response: Success: %v \n", resp.Success)
		t.Fail()
	}
	acmValid := resp.AcmValid
	if acmValid != false {
		// TODO should this be false?
		log.Printf("Unexpected ValidateAcm response: AcmValid: %v \n", resp.AcmValid)
		t.Fail()
	}
}

func TestPopulateAndValidateAcm(t *testing.T) {

	resp, err := aacClient.PopulateAndValidateAcm(acmComplete1)
	if err != nil {
		log.Println("Error from remote call PopulateAndValidateAcm() method: ", err)
		t.Fail()
	}
	result := resp.AcmValid
	if !result {
		log.Printf("Unexpected PopulateAndValidateAcm response: AcmValid: %v \n", resp.AcmValid)
		t.Fail()
	}
	return
}

func TestCreateAcmFromBannerMarking(t *testing.T) {
	log.Printf("CreateAcmFromBannerMarking() not implemented within service.\n")
}

func TestRollupAcms(t *testing.T) {
	acmList := []string{
		acmComplete1,
		acmComplete2,
		acmComplete3,
		acmComplete4,
	}

	resp, err := aacClient.RollupAcms(userDN1, acmList, shareType1, "")
	if err != nil {
		log.Printf("Error from remote call RollupAcms() method: %v\n", err)
		t.Fail()
	}

	if !resp.Success {
		log.Printf("Error in RollupAcms() response. Expected Success: true. Got: %v\n", resp.Success)
		t.Fail()
	}

	if !resp.AcmValid {
		log.Printf("Error in RollupAcms() response. Expected AcmValid: true. Got %v\n", resp.AcmValid)
		t.Fail()
	}
	return
}

func TestCheckAccessAndPopulate(t *testing.T) {

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
		log.Printf("Error from remote call CheckAccessAndPopulate() method: %v\n", err)
		t.Fail()
	}

	if !resp.Success {
		log.Printf("Error in CheckAccessAndPopulate response. Expected Success: true. Got: %v\n", resp.Success)
		t.Fail()
	}

	if numResponses := len(resp.AcmResponseList); numResponses != 4 {
		log.Printf("Expected 4 AcmResponse objects in AcmResponseList. Got: %v", numResponses)
		t.Fail()
	}

	// Check the validity and access for each item in AcmResponseList
	for _, item := range resp.AcmResponseList {
		expectedValid := true
		expectedHasAccess := true
		if item.AcmValid != expectedValid {
			log.Printf("Expected AcmValid to be %v. Got: %v", expectedValid, item.AcmValid)
			t.Fail()
		}
		if item.HasAccess != expectedHasAccess {
			log.Printf("Expected HasAccess to be %v. Got: %v", expectedHasAccess, item.HasAccess)
			t.Fail()
		}
	}
	return
}

func TestGetUserAttributes(t *testing.T) { return }
func TestGetSnippets(t *testing.T)       { return }
func TestGetShare(t *testing.T)          { return }
