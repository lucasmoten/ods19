package aac

import (
	"fmt"
	"log"

	"decipher.com/object-drive-server/ssl"

	"decipher.com/object-drive-server/config"
	"github.com/samuel/go-thrift/thrift"
)

// GetAACClient creates a new AacServiceClient.
func GetAACClient(aacHost string, aacPort int, trustPath, certPath, keyPath string) (*AacServiceClient, error) {
	serverCN := config.GetEnvOrDefault(config.OD_AAC_CN, "")
	insecureSkipVerify := config.GetEnvOrDefault(config.OD_AAC_INSECURE_SKIP_VERIFY, "false") == "true"
	conn, err := ssl.NewTLSClientConn(
		trustPath,
		certPath,
		keyPath,
		serverCN,
		aacHost,
		fmt.Sprintf("%d", aacPort),
		insecureSkipVerify,
	)
	if err != nil {
		if insecureSkipVerify == false {
			log.Printf("!!! Check that the AAC server cert CN matches us.  unable to get AAC. skipVerify:%t cn=%s: %v", insecureSkipVerify, serverCN, err)
		}
		return nil, err
	}
	trns := thrift.NewTransport(thrift.NewFramedReadWriteCloser(conn, 0), thrift.BinaryProtocol)
	client := thrift.NewClient(trns, true)

	return &AacServiceClient{Client: client}, nil
}
