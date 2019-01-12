package aac

import (
	"fmt"
	"log"
	"strings"

	"bitbucket.di2e.net/dime/object-drive-server/ssl"

	"bitbucket.di2e.net/dime/object-drive-server/config"
	"github.com/samuel/go-thrift/thrift"
)

// GetAACClient creates a new AacServiceClient.
func GetAACClient(aacHost string, aacPort int, trustPath, certPath, keyPath string, serverCN string) (*AacServiceClient, error) {
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
		if insecureSkipVerify == false && strings.Contains(err.Error(), "certificate is valid for") {
			log.Printf("Unable to get a valid AAC Client connection. AAC Server Certificate Common Name %s should match value configured in OD_AAC_CN, or OD_AAC_INSECURE_SKIP_VERIFY should be set to true: %v", serverCN, err)
		}
		return nil, err
	}
	trns := thrift.NewTransport(thrift.NewFramedReadWriteCloser(conn, 0), thrift.BinaryProtocol)
	client := thrift.NewClient(trns, true)

	return &AacServiceClient{Client: client}, nil
}
