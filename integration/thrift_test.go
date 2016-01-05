package integration

import (
	"fmt"
	"log"
	"testing"

	aac "decipher.com/oduploader/cmd/cryptotest/gen-go2/aac"
	"decipher.com/oduploader/config"
	t2 "github.com/samuel/go-thrift/thrift"
)

func TestThriftCommunication(t *testing.T) {

	// dur, _ := time.ParseDuration("20s")
	// cfg := config.NewAACTLSConfig()
	// cfg.InsecureSkipVerify = true
	// transport, err := thrift.NewTSSLSocket("twl-server-generic2:9093", cfg)
	conn, err := config.NewOpenSSLTransport()
	if err != nil {
		log.Fatal(err)
	}

	// try other Thrift lib
	trns := t2.NewTransport(conn, t2.BinaryProtocol)
	client := t2.NewClient(trns, false)

	aacClient := aac.AacServiceClient{Client: client}
	// log.Println(aacClient)

	// byteList, _ := stringToInt8Slice("<ddms:title ism:classification='S'>Foo</ddms:title>")
	byteList := []byte("<ddms:title ism:classification='S'>Foo</ddms:title>")
	log.Println(byteList)
	propertiesMap := make(map[string]string)

	propertiesMap["bedrock.message.traffic.dia.orgs"] = "DIA DOD_DIA"
	propertiesMap["bedrock.message.traffic.orcon.project"] = "DCTC"
	propertiesMap["bedrock.message.traffic.orcon.group"] = "All"

	resp, err := aacClient.BuildAcm(byteList, "XML", propertiesMap)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Reponse: ", resp)
	// client.BuildAcm(byteList, "XML", propertiesMap)

	return
}

func stringToInt8Slice(input string) ([]int8, error) {
	byteSliced := []byte(input)
	result := make([]int8, len(byteSliced))
	for i := 0; i < len(byteSliced); i++ {
		// TODO this can panic. Is this a case for panic/recover?
		fmt.Printf("slicin! %v of %v \n", i, len(byteSliced))
		result[i] = int8(byteSliced[i])
	}
	return result, nil
}
