package integration

import (
	"crypto/tls"
	"crypto/x509"
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

func notinit() {
	// NOTE: USING RAW TLS CONFIG
	// tlsConfig, err := config.NewTLSConfigFromPEM(certPath, trustPath)
	tlsConfig, err := setUpTLSConfig()
	if err != nil {
		log.Fatal("Could not get TLS Config")
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	client := &http.Client{Transport: tr}
	// req, err := http.NewRequest("GET", "https://10.2.11.46:10443/ping", nil)
	//req, err := http.NewRequest("GET", "https://10.2.11.47:10443/ping", nil)
	req, err := http.NewRequest("GET", "https://10.2.11.46:10443/ping", nil)
	if err != nil {
		log.Fatal("Could not construct request")
	}
	//USER_DN: CN=Zuniga Brenda cnzunib,OU=people,OU=dia,OU=dod,O=U.S. Government,C=us
	req.Header.Add("USER_DN", "CN=Zuniga Brenda cnzunib,OU=people,OU=dia,OU=dod,O=U.S. Government,C=us")
	req.Header.Add("Accept-Encoding", "identity")
	resp, err := client.Do(req)

	if resp == nil {
		log.Println("Response is nil")
	}
	if err != nil {
		fmt.Println(err)
		log.Fatal("It did not work: ", err)
	}
	log.Println("It worked!: ", resp.Body)

}

func setUpTLSConfig() (*tls.Config, error) {

	basePath := filepath.Join("/Users", "cmcfarland", "Code", "certtool", "certs", "output")
	log.Println(basePath)
	rootCAPool := x509.NewCertPool()
	rootBytes, err := ioutil.ReadFile(filepath.Join(basePath, "root_cert3_DIASRootCA.asn1"))
	if err != nil {
		log.Println("Could not read rootCA: ", err)
		return nil, err
	}
	parsedRoot, err := x509.ParseCertificate(rootBytes)
	if err != nil {
		log.Println("Could not parse rootCA.")
		return nil, err
	}

	rootCAPool.AddCert(parsedRoot)

	tlsConfig := tls.Config{
		RootCAs: rootCAPool,
	}

	// cert1_subject_twl-server-generic2_issued_by_DIASSUBCA2.asn1
	// cert2_subject_DIASSUBCA2_issued_by_DIASRootCA.asn1
	// use the bytes from above for Root CA

	// Private key?
	keyBytes, _ := ioutil.ReadFile(filepath.Join(basePath, "twl-server-key.key"))
	priv2, err := x509.ParsePKCS1PrivateKey(keyBytes)
	if err != nil {
		log.Println("Could not parse private key")
	}
	c1Bytes, _ := ioutil.ReadFile(filepath.Join(basePath, "cert1_subject_twl-server-generic2_issued_by_DIASSUBCA2.asn1"))
	// c2Bytes, _ := ioutil.ReadFile(filepath.Join(basePath, "cert2_subject_DIASSUBCA2_issued_by_DIASRootCA.asn1"))

	cert := tls.Certificate{
		// Certificate: [][]byte{c1Bytes, c2Bytes, rootBytes},
		Certificate: [][]byte{c1Bytes},
		PrivateKey:  priv2,
	}

	tlsConfig.Certificates = []tls.Certificate{cert}
	tlsConfig.InsecureSkipVerify = true
	tlsConfig.ServerName = "twl-server-generic2"

	// parsedC1, err := x509.ParseCertificate(c1Bytes)
	// if err != nil {
	// 	log.Println("Could not parse c1")
	// 	return nil, err
	// }
	// parsedC2, err := x509.ParseCertificate(c1Bytes)
	// if err != nil {
	// 	log.Println("Could not parse c2")
	// 	return nil, err
	// }
	return &tlsConfig, nil
}

func TestThriftComm(t *testing.T) {
	trustPath := filepath.Join(config.CertsDir, "server", "server.trust.pem")
	certPath := filepath.Join(config.CertsDir, "server", "server.cert.pem")
	keyPath := filepath.Join(config.CertsDir, "server", "server.key.pem")

	conn, err := config.NewOpenSSLTransport(
		trustPath, certPath, keyPath, "10.2.11.46", "10443")
	if err != nil {
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

func ThriftCommFail03(t *testing.T) {
	trustPath := filepath.Join(config.CertsDir, "server", "trust.pem")
	certPath := filepath.Join(config.CertsDir, "server", "server.cert.pem")
	keyPath := filepath.Join(config.CertsDir, "server", "server.key.pem")
	// audit path: /bedrock/service/audit/2.3
	// Audit service address 10.2.11.46:10443
	// or 10.2.11.47:10443
	// bedrock.363-283.io
	conn, err := config.NewOpenSSLTransport(
		trustPath, certPath, keyPath, "10.2.11.46", "10443")
	if err != nil {
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

func ThriftCommFail02(t *testing.T) {
	trustPath := filepath.Join(config.CertsDir, "server", "trust.pem")
	certPath := filepath.Join(config.CertsDir, "server", "server.cert.pem")
	keyPath := filepath.Join(config.CertsDir, "server", "server.key.pem")
	// audit path: /bedrock/service/audit/2.3
	// Audit service address 10.2.11.46:10443
	// or 10.2.11.47:10443
	// bedrock.363-283.io
	conn, err := config.NewOpenSSLTransport(
		trustPath, certPath, keyPath, "bedrock.363-283.io", "10443")
	if err != nil {
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

func ThriftCommFail01(t *testing.T) {
	trustPath := filepath.Join(config.CertsDir, "server", "trust.pem")
	certPath := filepath.Join(config.CertsDir, "server", "server.cert.pem")
	keyPath := filepath.Join(config.CertsDir, "server", "server.key.pem")
	// audit path: /bedrock/service/audit/2.3
	// Audit service address 10.2.11.46:10443
	// or 10.2.11.47:10443
	// bedrock.363-283.io
	conn, err := config.NewOpenSSLTransport(
		trustPath, certPath, keyPath, "10.2.11.47", "10443")
	if err != nil {
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

// removed Test
func AuditServicePing(t *testing.T) {

	if testing.Short() {
		t.Skip("Skipping as integration test.")
	}

	// var event events_thrift.AuditEvent

	// res, err := auditClient.SubmitAuditEvent(&event)
	// if err != nil {
	// 	t.Error("Error from auditClient.Ping(): ", err)
	// }
	//
	// fmt.Println("result: ", res)

	// res, err := auditClient.Ping()
	// if err != nil {
	// 	t.Error("Error from auditClient.Ping(): ", err)
	// }
	//
	// fmt.Println("result: ", res)

}

// func makeCertList(items... []byte) []tls.Certificate {
// 	certs := make([]tls.Certificate, 0)
// 	for _, item := range items {
// 	certs = append(certs, item)
// 	}
// 	return certs
// }
