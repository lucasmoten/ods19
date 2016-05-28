package aac

import (
	"path/filepath"

	globalconfig "decipher.com/object-drive-server/config"
	"github.com/samuel/go-thrift/thrift"
	"github.com/uber-go/zap"
)

var (
	logger = globalconfig.RootLogger
)

//GetAACClient is a connection to AAC
func GetAACClient(aacHost string, aacPort int) (*AacServiceClient, error) {
	trustPath := filepath.Join(globalconfig.CertsDir, "client-aac", "trust", "client.trust.pem")
	certPath := filepath.Join(globalconfig.CertsDir, "client-aac", "id", "client.cert.pem")
	keyPath := filepath.Join(globalconfig.CertsDir, "client-aac", "id", "client.key.pem")

	aacLogger := logger.With(zap.String("host", aacHost), zap.Int("port", aacPort))
	conn, err := globalconfig.NewOpenSSLTransport(
		trustPath, certPath, keyPath, aacHost, aacPort, nil)

	if err != nil {
		aacLogger.Error("cannot create aac client", zap.String("err", err.Error()))
		return nil, err
	}
	trns := thrift.NewTransport(thrift.NewFramedReadWriteCloser(conn, 0), thrift.BinaryProtocol)
	client := thrift.NewClient(trns, true)

	return &AacServiceClient{Client: client}, nil
}
