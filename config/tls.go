package config

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
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
