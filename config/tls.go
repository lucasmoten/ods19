package config

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/spacemonkeygo/openssl"
)

// Information about DoDIIS two-way SSL is here:
// https://confluence.363-283.io/pages/viewpage.action?pageId=557803

// PKCS12 implementation is here: https://godoc.org/golang.org/x/crypto/pkcs12

/////XXX we should probably eliminate globals from config
// TODO export these as globals so we can set them with command line flags also?
var (
	uploaderCertPath     string
	thriftClientCertPath string
)

//TODO: globals should deal with this in flags and envs.
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
	//XXX This doesn't look right (cert.pem),
	// as files going into clientCertPool are trust certs.
	return NewUploaderTLSConfigWithParms(uploaderCertPath, "")
}

// NewUploaderTLSConfigWithEnvironment picks through the environment
// to give us a custom TLS configuration
func NewUploaderTLSConfigWithEnvironment(env *Environment) *tls.Config {
	return NewUploaderTLSConfigWithParms("", env.ServerTrustFile)
}

// NewUploaderTLSConfigWithParms reads the environment for paths to X509 certificates
// or uses a default. A pointer to TLSConfig is returned
//
// TODO: fatals should not be in libraries.  Return error codes
func NewUploaderTLSConfigWithParms(certPath string, trustPath string) *tls.Config {
	clientCertPool := x509.NewCertPool()

	//XXX this does not seem right - clientCertPool should be trusts
	// but parsing it on startup might be interesting
	if certPath != "" {
		certBytes, err := ioutil.ReadFile(certPath)
		if err != nil {
			log.Fatalln("Unable to open cert file at: ", certPath, err)
		}
		actualCert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			log.Fatal("Error parsing cert: ", err)
		}
		clientCertPool.AddCert(actualCert)
	}

	//TODO: this does not explicitly sanity check the certificates here
	if trustPath != "" {
		trustBytes, err := ioutil.ReadFile(trustPath)
		if err != nil {
			log.Fatalln("Unable to open trust file at: ", trustPath, err)
		}
		if ok := clientCertPool.AppendCertsFromPEM(trustBytes); !ok {
			log.Fatal("Error appending cert: ", err)
		}
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

	certLocation := CertsDir + "/server/server.cert.pem"
	keyLocation := CertsDir + "/server/server.key.pem"
	cert, err := tls.LoadX509KeyPair(certLocation, keyLocation)
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
func NewOpenSSLTransport(trustPath, certPath, keyPath, host, port string) (*openssl.Conn, error) {
	ctx, err := openssl.NewCtx()
	if err != nil {
		return nil, err
	}
	ctx.SetOptions(openssl.CipherServerPreference)
	// ctx.SetOptions(openssl.NoSSLv3)

	err = ctx.LoadVerifyLocations(trustPath, "")
	if err != nil {
		return nil, err
	}

	certBytes, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	cert, err := openssl.LoadCertificateFromPEM(certBytes)
	if err != nil {
		return nil, err
	}
	ctx.UseCertificate(cert)

	keyBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	privKey, err := openssl.LoadPrivateKeyFromPEM(keyBytes)
	if err != nil {
		return nil, err
	}
	ctx.UsePrivateKey(privKey)

	addr := host + ":" + port
	conn, err := openssl.Dial("tcp", addr, ctx, 1)
	if err != nil {
		log.Println("Error making openssl conn!")
		return nil, err
	}
	return conn, nil
}

// GetDNFromCert will extract the dn in the format that everything expects
func GetDNFromCert(name pkix.Name) string {
	dnSeq := name.ToRDNSequence()
	dnArray := ""
	iLen := len(dnSeq)
	//Traverse the pkix name backwards
	for i := 0; i < iLen; i++ {
		dnPart := dnSeq[iLen-1-i]
		jLen := len(dnPart)
		var pPart string
		for j := 0; j < jLen; j++ {
			if i > 0 || j > 0 {
				dnArray = dnArray + ","
			}
			switch {
			case dnPart[jLen-1-j].Type.String() == "2.5.4.6":
				pPart = "C"
			case dnPart[jLen-1-j].Type.String() == "2.5.4.10":
				pPart = "O"
			case dnPart[jLen-1-j].Type.String() == "2.5.4.11":
				pPart = "OU"
			case dnPart[jLen-1-j].Type.String() == "2.5.4.3":
				pPart = "CN"
			}
			dnArray = dnArray + fmt.Sprintf("%s=%v", pPart, dnPart[jLen-1-j].Value)
		}
	}
	return dnArray
}
