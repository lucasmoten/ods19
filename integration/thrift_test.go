package integration

import (
	"fmt"
	"log"
	"testing"

	aac "decipher.com/oduploader/cmd/cryptotest/gen-go/aac"
	"decipher.com/oduploader/config"
	"git.apache.org/thrift.git/lib/go/thrift"
)

func TestThriftCommunication(t *testing.T) {

	cfg := config.NewAACTLSConfig()
	cfg.InsecureSkipVerify = true
	transport, err := thrift.NewTSSLSocket("dockervm:9093", cfg)
	// transportFactory := thrift.NewTTransportFactory()
	// transport = transportFactory.GetTransport(transport)
	defer transport.Close()
	if err = transport.Open(); err != nil {
		log.Println("Failed to open transport.")
		t.Fail()
	}
	// get protocol fac
	fac := thrift.NewTBinaryProtocolFactoryDefault()
	client := aac.NewAacServiceClientFactory(transport, fac) // (t thrift.TTransport, f thrift.TProtocolFactory) *AacServiceClient

	byteList, _ := stringToInt8Slice("<ddms:title ism:classification='S'>Foo</ddms:title>")
	propertiesMap := make(map[string]string)

	propertiesMap["bedrock.message.traffic.dia.orgs"] = "DIA DOD_DIA"
	propertiesMap["bedrock.message.traffic.orcon.project"] = "DCTC"
	propertiesMap["bedrock.message.traffic.orcon.group"] = "All"

	client.BuildAcm(byteList, "XML", propertiesMap)

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
