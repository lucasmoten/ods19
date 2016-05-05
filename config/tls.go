package config

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/spacemonkeygo/openssl"
)

// TODO export these as globals so we can set them with command line flags also?
var (
	uploaderCertPath     string
	thriftClientCertPath string
)

// OpenSSLDialOptions wraps the bitmask flags that are passed as the last arg
// to openssl.Dial. Create an instance of this struct and pass the value of the
// Flags field to Dial.
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

// func init() {
// 	if certPathFromEnv := os.Getenv("ODRIVE_UPLOADER_CERT"); certPathFromEnv != "" {
// 		log.Printf("UPLOADER X509 certificate path read from environment: %s", certPathFromEnv)
// 		uploaderCertPath = certPathFromEnv
// 	} else {
// 		uploaderCertPath = "cert.pem" // TODO point to default
// 	}

// 	if certPathFromEnv := os.Getenv("ODRIVE_THRIFT_CLIENT_CERT"); certPathFromEnv != "" {
// 		log.Printf("THRIFT CLIENT X509 certificate path read from environment: %s", certPathFromEnv)
// 		thriftClientCertPath = certPathFromEnv
// 	} else {
// 		thriftClientCertPath = "./certs/ling/twlserver.crt"
// 	}
// }

// NewTLSConfigFromPEM ...
func NewTLSConfigFromPEM(trustPath, certPath string) (*tls.Config, error) {

	clientCertPool := x509.NewCertPool()

	certBytes, err := ioutil.ReadFile(certPath)
	if err != nil {
		log.Fatalln("Unable to open cert file at: ", certPath, err)
	}
	ok := clientCertPool.AppendCertsFromPEM(certBytes)
	if !ok {
		return nil, fmt.Errorf("Could not append cert from PEM file: %s\n", certPath)
	}

	trustBytes, err := ioutil.ReadFile(trustPath)
	if err != nil {
		log.Fatalln("Unable to open trust file at: ", trustPath, err)
	}
	if ok := clientCertPool.AppendCertsFromPEM(trustBytes); !ok {
		return nil, fmt.Errorf("Could not append trusts from PEM file: %s\n", trustPath)
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

// NewTLSConfig returns a tls.Config object for creating standard Golang https
// clients. This method is a helper and users can implement their own.
func NewTLSConfig(trustPath, certPath, keyPath string) (*tls.Config, error) {

	// Create the trusted certificate pool.
	trustBytes, err := ioutil.ReadFile(trustPath)
	if err != nil {
		log.Printf("Unable to read %s: %v", trustPath, err)
		return nil, err
	}
	trustCertPool := x509.NewCertPool()
	if !trustCertPool.AppendCertsFromPEM(trustBytes) {
		log.Printf("Error parsing cert: %v", err)
		return nil, err
	}

	// Create certificate.
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Printf("Error parsing cert: %v", err)
		return nil, err
	}

	cfg := tls.Config{
		Certificates:             []tls.Certificate{cert},
		ClientCAs:                trustCertPool,
		InsecureSkipVerify:       true,
		ServerName:               "twl-server-generic2",
		PreferServerCipherSuites: true,
	}
	cfg.BuildNameToCertificate()

	return &cfg, nil
}

// NewOpenSSLConn returns a TCP connection establish with OpenSSL.
func NewOpenSSLConn(trustPath, certPath, keyPath, host, port string, dialOpts *OpenSSLDialOptions) (*openssl.Conn, error) {

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

// func NewOpenSSLHTTPClient(trustPath, certPath, keyPath, "twl-server-generic2", "9093", dialOpts)
// NewOpenSSLHTTPClient
func NewOpenSSLHTTPClient(trustPath, certPath, keyPath, host, port string, dialOpts *OpenSSLDialOptions) (*http.Client, error) {

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

	c := &http.Client{Transport: &http.Transport{
		DialTLS: func(network, address string) (net.Conn, error) {
			return openssl.Dial("tcp", address, ctx, dialOpts.Flags)
		},
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}}

	return c, nil
}
