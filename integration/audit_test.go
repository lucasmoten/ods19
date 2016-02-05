package integration

import (
	"fmt"
	"log"
	"path/filepath"
	"testing"

	"decipher.com/oduploader/config"
	"decipher.com/oduploader/services/audit"
)

var auditClient *audit.Client

func TestThriftComm(t *testing.T) {
	trustPath := filepath.Join(config.CertsDir, "server", "server.trust.pem")
	certPath := filepath.Join(config.CertsDir, "server", "server.cert.pem")
	keyPath := filepath.Join(config.CertsDir, "server", "server.key.pem")

	conn, err := config.NewOpenSSLTransport(
		trustPath, certPath, keyPath, "10.2.11.46", "10443", nil)
	if err != nil {
		t.Log("NewOpenSSLTransport failed.")
		log.Fatal(err)
	}
	auditClient = audit.NewAuditServiceClient(conn)
	res, err := auditClient.Ping()
	if err != nil {
		log.Println("Error calling Ping: ", err)
		t.FailNow()
	}
	fmt.Println("Reponse: ", res)
}
