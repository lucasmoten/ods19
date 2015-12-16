package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

// Information about DoDIIS two-way SSL is here:
// https://confluence.363-283.io/pages/viewpage.action?pageId=557803

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

// Stuffs does so much stuff, it's really a great function, you guys.

func DoStuff() {
	fmt.Println("Stuff done!")
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
