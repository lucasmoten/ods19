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

/////XXX we should probably eliminate globals from config
// TODO export these as globals so we can set them with command line flags also?
var (
	uploaderCertPath     string
	thriftClientCertPath string
)

// OpenSSLDialOptions wraps the bitmask flags that are passed as the last arg
// to openssl.Dial
type OpenSSLDialOptions struct {
	Flags openssl.DialFlags
}

// SetInsecureSkipHostVerification sets flag openssl.InsecureSkipHostVerification
func (opts *OpenSSLDialOptions) SetInsecureSkipHostVerification() {
	opts.Flags = opts.Flags | 1
}

// SetDisableSNI sets the flag openssl.DisableSNI
func (opts *OpenSSLDialOptions) SetDisableSNI() {
	opts.Flags = opts.Flags | 2
}

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
func NewUploaderTLSConfig() (*tls.Config, int) {
	//XXX This doesn't look right (cert.pem),
	// as files going into clientCertPool are trust certs.
	return NewUploaderTLSConfigWithParms(uploaderCertPath, "")
}

// NewUploaderTLSConfigWithEnvironment picks through the environment
// to give us a custom TLS configuration
func NewUploaderTLSConfigWithEnvironment(env *Environment) (*tls.Config, int) {
	return NewUploaderTLSConfigWithParms("", env.ServerTrustFile)
}

// NewUploaderTLSConfigWithParms reads the environment for paths to X509 certificates
// or uses a default. A pointer to TLSConfig is returned
//
// TODO: fatals should not be in libraries.  Return error codes
func NewUploaderTLSConfigWithParms(certPath string, trustPath string) (*tls.Config, int) {
	clientCertPool := x509.NewCertPool()
	errCode := 0

	//XXX this does not seem right - clientCertPool should be trusts
	// but parsing it on startup might be interesting
	if certPath != "" {
		certBytes, err := ioutil.ReadFile(certPath)
		if err != nil {
			log.Printf("Unable to open certificate file at path '%s': %s", certPath, err.Error())
			errCode = 1
		} else {
			actualCert, err := x509.ParseCertificate(certBytes)
			if err != nil {
				log.Printf("Error parsing certificate: %s", err.Error())
				errCode = 2
			} else {
				clientCertPool.AddCert(actualCert)
			}
		}
	}

	//TODO: this does not explicitly sanity check the certificates here
	if trustPath != "" {
		trustBytes, err := ioutil.ReadFile(trustPath)
		if err != nil {
			log.Printf("Unable to open trust file at path '%s': %s", trustPath, err.Error())
			errCode = 3
		} else {
			if ok := clientCertPool.AppendCertsFromPEM(trustBytes); !ok {
				log.Printf("Error appending the cert to the pool: %s", err.Error())
				errCode = 4
			}
		}
	}

	if errCode == 0 {
		tlsConfig := &tls.Config{
			ClientAuth:               tls.RequireAndVerifyClientCert,
			ClientCAs:                clientCertPool,
			PreferServerCipherSuites: true,
			MinVersion:               tls.VersionTLS10,
		}
		tlsConfig.BuildNameToCertificate()
		return tlsConfig, 0
	} else {
		return nil, errCode
	}

}

// NewTLSConfigFromPEM ...
func NewTLSConfigFromPEM(trustPath, certPath string) (*tls.Config, error) {

	clientCertPool := x509.NewCertPool()

	if certPath != "" {
		certBytes, err := ioutil.ReadFile(certPath)
		if err != nil {
			log.Fatalln("Unable to open cert file at: ", certPath, err)
		}
		ok := clientCertPool.AppendCertsFromPEM(certBytes)
		if !ok {
			return nil, fmt.Errorf("Could not append cert from PEM file: %s\n", certPath)
		}
	}

	if trustPath != "" {
		trustBytes, err := ioutil.ReadFile(trustPath)
		if err != nil {
			log.Fatalln("Unable to open trust file at: ", trustPath, err)
		}
		if ok := clientCertPool.AppendCertsFromPEM(trustBytes); !ok {
			return nil, fmt.Errorf("Could not append trusts from PEM file: %s\n", trustPath)
		}
	}

	tlsConfig := &tls.Config{
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                clientCertPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS10,
		InsecureSkipVerify:       true,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}

// NewAACTLSConfig ...
func NewAACTLSConfig() *tls.Config {

	certBytes, err := ioutil.ReadFile(thriftClientCertPath)
	if err != nil {
		log.Fatalln("Unable to read cert.pem", err)
	}

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
		// Ensure that we only use our "CA" to validate certificates
		ClientCAs: clientCertPool,
		// PFS because we can but this will reject client with RSA certificates
		// CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		// Force it server side
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS10,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig
}

// NewOpenSSLTransport returns a TCP connection establish with OpenSSL.
func NewOpenSSLTransport(trustPath, certPath, keyPath, host, port string, dialOpts *OpenSSLDialOptions) (*openssl.Conn, error) {

	// Default to flag 0
	if dialOpts == nil {
		dialOpts = &OpenSSLDialOptions{}
	}

	ctx, err := openssl.NewCtx()
	if err != nil {
		return nil, err
	}
	ctx.SetOptions(openssl.CipherServerPreference)
	ctx.SetOptions(openssl.NoSSLv3)

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
	conn, err := openssl.Dial("tcp", addr, ctx, dialOpts.Flags)
	if err != nil {
		log.Printf("Error making openssl connection: %s", err.Error())
		return nil, err
	}
	return conn, nil
}

// GetDNFromCert will extract the DN in the format that everything expects.
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
