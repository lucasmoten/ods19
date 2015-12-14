package config

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
)

var certPath string

func init() {
	certPathFromEnv := os.Getenv("ODRIVE_UPLOADER_CERT")
	if certPathFromEnv != "" {
		log.Printf("X509 certificate path read from environment: %s", certPathFromEnv)
		certPath = certPathFromEnv
	} else {
		certPath = "cert.pem"
	}
}

// NewUploaderTLSConfig reads the environment for paths to X509 certificates
// or uses a default. A pointer to TLSConfig is returned
func NewUploaderTLSConfig() *tls.Config {

	certBytes, err := ioutil.ReadFile("cert.pem")
	if err != nil {
		log.Fatalln("Unable to read cert.pem", err)
	}

	clientCertPool := x509.NewCertPool()
	if ok := clientCertPool.AppendCertsFromPEM(certBytes); !ok {
		log.Fatalln("Unable to add certificate to certificate pool")
	}

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
