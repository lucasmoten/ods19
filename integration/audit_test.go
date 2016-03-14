package integration

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"testing"

	"decipher.com/oduploader/config"
	"decipher.com/oduploader/services/audit"
)

var auditClient *audit.Client

func TestAuditServiceProxyThroughGatekeeper(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	trustPath := filepath.Join(config.CertsDir, "clients", "client.trust.pem")
	certPath := filepath.Join(config.CertsDir, "clients", "test_0.cert.pem")
	keyPath := filepath.Join(config.CertsDir, "clients", "test_0.key.pem")
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	tlsConfig, err := config.NewTLSConfigFromPEM(trustPath, certPath)
	if err != nil {
		t.Logf("Error from NewTLSConfigFromPEM: %v", err)
		t.Fail()
	}
	tlsConfig.Certificates = []tls.Certificate{cert}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}
	url := fmt.Sprintf("https://%s:8080/service/auditservice/1.0/ping", config.DockerVM)
	resp, err := client.Get(url)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	defer resp.Body.Close()

	// We don't care about the response, only that it succeeds.
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Log("Error calling out to audit service. Are you on the VPN?")
		t.Fail()
	}
}

func TestAuditServiceThriftCommunication(t *testing.T) {

	if true {
		t.Skip("Audit test skipped due to remote server hostname validation requirements.")
	}

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
