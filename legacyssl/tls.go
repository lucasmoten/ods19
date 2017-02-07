package legacyssl

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"decipher.com/object-drive-server/config"

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

func NewSSLConn(trustPath, certPath, keyPath, host, port string, insecure bool) (io.ReadWriteCloser, error) {
	//first try it without open ssl, and complain when falling back on openssl
	serverName := ""
	conn, err := config.NewTLSClientConn(trustPath, certPath, keyPath, serverName, host, port, insecure)
	if err != nil {
		conn2, err2 := NewOpenSSLConn(trustPath, certPath, keyPath, host, port, insecure)
		if err2 != nil {
			return nil, err2
		}
		// If we never get this error, then we can just call config.NewTLSClientConn directly, and eliminate cgo
		log.Printf("ERROR: we could not connect with standard Go TLS, but could connect with OpenSSL.  Re-issuing this certificate may make this problem go away.")
		return conn2, nil

	}
	log.Printf("INFO: we are using native Go TLS to connect to %s:%s", host, port)
	return conn, nil
}

// NewOpenSSLConn returns a TCP connection establish with OpenSSL.
func NewOpenSSLConn(trustPath, certPath, keyPath, host, port string, insecure bool) (io.ReadWriteCloser, error) {

	dialOpts := &OpenSSLDialOptions{}
	if insecure {
		dialOpts.SetInsecureSkipHostVerification()
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
