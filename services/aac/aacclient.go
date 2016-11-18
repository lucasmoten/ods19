package aac

import (
	"decipher.com/object-drive-server/legacyssl"
	"github.com/samuel/go-thrift/thrift"
)

// GetAACClient creates a new AacServiceClient.
func GetAACClient(aacHost string, aacPort int, trustPath, certPath, keyPath string) (*AacServiceClient, error) {

	conn, err := legacyssl.NewOpenSSLTransport(
		trustPath, certPath, keyPath, aacHost, aacPort, nil)

	if err != nil {
		return nil, err
	}
	trns := thrift.NewTransport(thrift.NewFramedReadWriteCloser(conn, 0), thrift.BinaryProtocol)
	client := thrift.NewClient(trns, true)

	return &AacServiceClient{Client: client}, nil
}
