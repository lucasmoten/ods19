package integration

import (
	"log"
	"testing"

	aac "decipher.com/oduploader/cmd/cryptotest/gen-go2/aac"
	"decipher.com/oduploader/config"
	t2 "github.com/samuel/go-thrift/thrift"
)

var acmPartial1 = "{ \"path\": \"\", \"classif\":\"TS\", \"sci_ctrls\":[ \"HCS\", \"SI-G\", \"TK\" ], \"dissem_ctrls\":[ \"OC\" ], \"dissem_countries\":[ \"USA\" ], \"oc_attribs\":[ { \"orgs\":[ \"dia\" ] } ] }"
var acmComplete1 = "{ \"version\":\"2.1.0\", \"classif\":\"TS\", \"owner_prod\":[], \"atom_energy\":[], \"sar_id\":[], \"sci_ctrls\":[ \"HCS\", \"SI-G\", \"TK\" ], \"disponly_to\":[ \"\" ], \"dissem_ctrls\":[ \"OC\" ], \"non_ic\":[], \"rel_to\":[], \"fgi_open\":[], \"fgi_protect\":[], \"portion\":\"TS//HCS/SI-G/TK//OC\", \"banner\":\"TOP SECRET//HCS/SI-G/TK//ORCON\", \"dissem_countries\":[ \"USA\" ], \"accms\":[], \"macs\":[], \"oc_attribs\":[ { \"orgs\":[ \"dia\" ], \"missions\":[], \"regions\":[] } ], \"f_clearance\":[ \"ts\" ], \"f_sci_ctrls\":[ \"hcs\", \"si_g\", \"tk\" ], \"f_accms\":[], \"f_oc_org\":[ \"dia\", \"dni\" ], \"f_regions\":[], \"f_missions\":[], \"f_share\":[], \"f_atom_energy\":[], \"f_macs\":[], \"disp_only\":\"\" }"

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
	dn := "CN=Holmes Jonathan,OU=People,OU=Bedrock,OU=Six 3 Systems,O=U.S. Government,C=US"
	tokenType := "pki_dias"
	acmComplete := "{ \"version\":\"2.1.0\", \"classif\":\"TS\", \"owner_prod\":[], \"atom_energy\":[], \"sar_id\":[], \"sci_ctrls\":[ \"HCS\", \"SI-G\", \"TK\" ], \"disponly_to\":[ \"\" ], \"dissem_ctrls\":[ \"OC\" ], \"non_ic\":[], \"rel_to\":[], \"fgi_open\":[], \"fgi_protect\":[], \"portion\":\"TS//HCS/SI-G/TK//OC\", \"banner\":\"TOP SECRET//HCS/SI-G/TK//ORCON\", \"dissem_countries\":[ \"USA\" ], \"accms\":[], \"macs\":[], \"oc_attribs\":[ { \"orgs\":[ \"dia\" ], \"missions\":[], \"regions\":[] } ], \"f_clearance\":[ \"ts\" ], \"f_sci_ctrls\":[ \"hcs\", \"si_g\", \"tk\" ], \"f_accms\":[], \"f_oc_org\":[ \"dia\", \"dni\" ], \"f_regions\":[], \"f_missions\":[], \"f_share\":[], \"f_atom_energy\":[], \"f_macs\":[], \"disp_only\":\"\" }"
	resp, err := aacClient.CheckAccess(dn, tokenType, acmComplete)

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
		log.Printf("Error from ValidateAcm() method: ", err)
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

	return
}

func TestRollupAcms(t *testing.T)             { return }
func TestCheckAccessAndPopulate(t *testing.T) { return }
func TestGetUserAttributes(t *testing.T)      { return }
func TestGetSnippets(t *testing.T)            { return }
func TestGetShare(t *testing.T)               { return }
