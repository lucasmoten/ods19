package integration

import (
	"log"
	"testing"

	aac "decipher.com/oduploader/cmd/cryptotest/gen-go2/aac"
	"decipher.com/oduploader/config"
	t2 "github.com/samuel/go-thrift/thrift"
)

func TestThriftCommunication(t *testing.T) {

	conn, err := config.NewOpenSSLTransport()
	if err != nil {
		log.Fatal(err)
	}

	// trns := t2.NewTransport(conn, t2.BinaryProtocol)
	trns := t2.NewTransport(t2.NewFramedReadWriteCloser(conn, 0), t2.BinaryProtocol)
	// trns := t2.NewTransport(conn, t2.CompactProtocol)
	client := t2.NewClient(trns, true)
	aacClient := aac.AacServiceClient{Client: client}

	// buildAcm params
	// byteList := []byte("<ddms:title ism:classification='S'>Foo</ddms:title>")
	// propertiesMap := make(map[string]string)
	// propertiesMap["bedrock.message.traffic.dia.orgs"] = "DIA DOD_DIA"
	// propertiesMap["bedrock.message.traffic.orcon.project"] = "DCTC"
	// propertiesMap["bedrock.message.traffic.orcon.group"] = "All"
	//resp, err := aacClient.BuildAcm(byteList, "XML", propertiesMap)

	// checkAccess params
	dn := "CN=Holmes Jonathan,OU=People,OU=Bedrock,OU=Six 3 Systems,O=U.S. Government,C=US"
	tokenType := "pki_dias"
	acmComplete := "{ \"version\":\"2.1.0\", \"classif\":\"TS\", \"owner_prod\":[], \"atom_energy\":[], \"sar_id\":[], \"sci_ctrls\":[ \"HCS\", \"SI-G\", \"TK\" ], \"disponly_to\":[ \"\" ], \"dissem_ctrls\":[ \"OC\" ], \"non_ic\":[], \"rel_to\":[], \"fgi_open\":[], \"fgi_protect\":[], \"portion\":\"TS//HCS/SI-G/TK//OC\", \"banner\":\"TOP SECRET//HCS/SI-G/TK//ORCON\", \"dissem_countries\":[ \"USA\" ], \"accms\":[], \"macs\":[], \"oc_attribs\":[ { \"orgs\":[ \"dia\" ], \"missions\":[], \"regions\":[] } ], \"f_clearance\":[ \"ts\" ], \"f_sci_ctrls\":[ \"hcs\", \"si_g\", \"tk\" ], \"f_accms\":[], \"f_oc_org\":[ \"dia\", \"dni\" ], \"f_regions\":[], \"f_missions\":[], \"f_share\":[], \"f_atom_energy\":[], \"f_macs\":[], \"disp_only\":\"\" }"
	resp, err := aacClient.CheckAccess(dn, tokenType, acmComplete)

	if err != nil {
		log.Fatal(err)
	}
	log.Println("Reponse: ", resp)
}
