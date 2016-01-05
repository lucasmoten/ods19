package integration

import (
	"crypto/tls"
	"fmt"
	"log"
	"testing"
	"time"

	apacheAac "decipher.com/oduploader/cmd/cryptotest/gen-go/aac"
	aac "decipher.com/oduploader/cmd/cryptotest/gen-go2/aac"
	"decipher.com/oduploader/config"
	apacheThrift "git.apache.org/thrift.git/lib/go/thrift"
	t2 "github.com/samuel/go-thrift/thrift"
)

func ApacheThriftImpl(t *testing.T) {

	dur, _ := time.ParseDuration("20s")
	// transport, err := apacheThrift.NewTSSLSocket("twl-server-generic2:9093", cfg)
	fac := apacheThrift.NewTBinaryProtocolFactoryDefault()
	conn, err := config.NewOpenSSLTransport()
	if err != nil {
		log.Fatal(err)
	}
	cfg := tls.Config{InsecureSkipVerify: true}
	transport := apacheThrift.NewTSSLSocketFromConnTimeout(conn, &cfg, dur)
	//protocol := fac.GetProtocol(transport)
	client := apacheAac.NewAacServiceClientFactory(transport, fac)

	propertiesMap := make(map[string]string)
	propertiesMap["bedrock.message.traffic.dia.orgs"] = "DIA DOD_DIA"
	propertiesMap["bedrock.message.traffic.orcon.project"] = "DCTC"
	propertiesMap["bedrock.message.traffic.orcon.group"] = "All"
	byteList, _ := stringToInt8Slice("<ddms:title ism:classification='S'>Foo</ddms:title>")
	res, err := client.BuildAcm(byteList, "XML", propertiesMap)
	if err != nil {
		log.Println("error encountered")
		log.Fatal(err)
	}
	log.Println("Success?  ----  ", res.GetAcmValid())
}

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

func stringToInt8Slice(input string) ([]int8, error) {
	byteSliced := []byte(input)
	result := make([]int8, len(byteSliced))
	for i := 0; i < len(byteSliced); i++ {
		// TODO this can panic. Is this a case for panic/recover?
		fmt.Printf("slicin! %v of %v \n", i, len(byteSliced))
		result[i] = int8(byteSliced[i])
	}
	fmt.Println("Returning...")
	return result, nil
}
