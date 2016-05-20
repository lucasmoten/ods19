package aac

import (
	"log"
	"path/filepath"

	oduconfig "decipher.com/object-drive-server/config"
	"github.com/samuel/go-thrift/thrift"
)

// TODO: remove the hardcoded filepath dependency in this function
// Is there some env vars we can rely on here?
func GetAACClient() (*AacServiceClient, error) {

	trustPath := filepath.Join(oduconfig.CertsDir, "client-aac", "trust", "client.trust.pem")
	certPath := filepath.Join(oduconfig.CertsDir, "client-aac", "id", "client.cert.pem")
	keyPath := filepath.Join(oduconfig.CertsDir, "client-aac", "id", "client.key.pem")
	aacHost := oduconfig.GetEnvOrDefault("OD_AAC_HOST", "twl-server-generic2")
	aacPort := oduconfig.GetEnvOrDefault("OD_AAC_PORT", "9093")
	conn, err := oduconfig.NewOpenSSLTransport(
		trustPath, certPath, keyPath, aacHost, aacPort, nil)

	if err != nil {
		log.Printf("cannot create aac client: %v", err)
		return nil, err
	}
	trns := thrift.NewTransport(thrift.NewFramedReadWriteCloser(conn, 0), thrift.BinaryProtocol)
	client := thrift.NewClient(trns, true)

	return &AacServiceClient{Client: client}, nil

}
