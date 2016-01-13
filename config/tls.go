package config

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/spacemonkeygo/openssl"
)

// Information about DoDIIS two-way SSL is here:
// https://confluence.363-283.io/pages/viewpage.action?pageId=557803

// PKCS12 implementation is here: https://godoc.org/golang.org/x/crypto/pkcs12

// TODO export these as globals so we can set them with command line flags also?
var (
	uploaderCertPath     string
	thriftClientCertPath string
)

func init() {
	if certPathFromEnv := os.Getenv("ODRIVE_UPLOADER_CERT"); certPathFromEnv != "" {
		log.Printf("UPLOADER X509 certificate path read from environment: %s", certPathFromEnv)
		uploaderCertPath = certPathFromEnv
	} else {
		uploaderCertPath = "cert.pem" // TODO point to default
	}

	if certPathFromEnv := os.Getenv("ODRIVE_THRIFT_CLIENT_CERT"); certPathFromEnv != "" {
		log.Printf("THRIFT CLIENT X509 certificate path read from environment: %s", certPathFromEnv)
		thriftClientCertPath = certPathFromEnv
	} else {
		thriftClientCertPath = "./certs/ling/twlserver.crt"
	}
}

// NewUploaderTLSConfig reads the environment for paths to X509 certificates
// or uses a default. A pointer to TLSConfig is returned
func NewUploaderTLSConfig() *tls.Config {

	certBytes, err := ioutil.ReadFile(uploaderCertPath)
	if err != nil {
		log.Fatalln("Unable to open file at: ", uploaderCertPath, err)
	}

	clientCertPool := x509.NewCertPool()
	actualCert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		log.Fatal("Error parsing cert: ", err)
	}
	clientCertPool.AddCert(actualCert)
	// if ok := clientCertPool.AppendCertsFromPEM(certBytes); !ok {
	//	log.Fatalln("Unable to add certificate to certificate pool")
	//	}

	tlsConfig := &tls.Config{
		// Reject any TLS certificate that cannot be validated
		ClientAuth: tls.RequireAndVerifyClientCert,
		// Ensure that we only use our "CA" to validate certificates
		ClientCAs: clientCertPool,
		// PFS because we can but this will reject client with RSA certificates
		// CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		// Force it server side
		PreferServerCipherSuites: true,
		// TLS 1.2 because we can
		MinVersion: tls.VersionTLS10,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig
}

// NewAACTLSConfig ...
func NewAACTLSConfig() *tls.Config {

	certBytes, err := ioutil.ReadFile(thriftClientCertPath)
	if err != nil {
		log.Fatalln("Unable to read cert.pem", err)
	}

	//cert := pkcs12.Decode(certBytes, "password")
	//pemBlocks, err := pkcs12.ToPEM(certBytes, "password")
	clientCertPool := x509.NewCertPool()
	actualCert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		log.Fatal("Error parsing cert: ", err)
	}
	clientCertPool.AddCert(actualCert)

	cert, err := tls.LoadX509KeyPair(thriftClientCertPath, "./certs/ling/twlserver.key")
	if err != nil {
		log.Fatal("Error parsing cert: ", err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		// Reject any TLS certificate that cannot be validated
		// ClientAuth: tls.RequireAndVerifyClientCert,
		// Ensure that we only use our "CA" to validate certificates
		ClientCAs: clientCertPool,
		// PFS because we can but this will reject client with RSA certificates
		// CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		// Force it server side
		PreferServerCipherSuites: true,
		// TLS 1.2 because we can
		MinVersion: tls.VersionTLS10,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig
}

// NewOpenSSLTransport ...
func NewOpenSSLTransport() (*openssl.Conn, error) {
	ctx, err := openssl.NewCtx()
	if err != nil {
		log.Fatal(err)
	}
	ctx.SetOptions(openssl.CipherServerPreference)
	ctx.SetOptions(openssl.NoSSLv3)

	trustLoc := filepath.Join(CertsDir, "clients", "client.trust.pem")
	err = ctx.LoadVerifyLocations(trustLoc, "")
	if err != nil {
		log.Fatal(err)
	}

	certPath := filepath.Join(CertsDir, "clients", "test_1.cert.pem")
	certBytes, err := ioutil.ReadFile(certPath)
	if err != nil {
		log.Fatalf("Unable to trust file: %v\n", err)
	}

	cert, err := openssl.LoadCertificateFromPEM(certBytes)
	if err != nil {
		log.Printf("Unable to parse cert:%v", err)
	}
	ctx.UseCertificate(cert)

	keyPath := filepath.Join(CertsDir, "clients", "test_1.key.pem")
	keyBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		log.Fatalf("Unable to key file: %v\n", err)
	}
	privKey, err := openssl.LoadPrivateKeyFromPEM(keyBytes)
	if err != nil {
		log.Fatalf("Unable to parse private key:%v", nil)
	}
	ctx.UsePrivateKey(privKey)

	conn, err := openssl.Dial("tcp", "twl-server-generic2:9093", ctx, 1)
	if err != nil {
		log.Println("Error making openssl conn!")
		log.Fatal(err)
	}
	return conn, nil
}
