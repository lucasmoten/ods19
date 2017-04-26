package ssl

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
)

// NewTLSClientConfig gets a config to make TLS connections
func NewTLSClientConfig(trustPath, certPath, keyPath, serverName string, insecure bool) (*tls.Config, error) {
	trustBytes, err := ioutil.ReadFile(trustPath)
	if err != nil {
		return nil, fmt.Errorf("Error parsing CA trust %s: %v", trustPath, err)
	}
	trustCertPool := x509.NewCertPool()
	if !trustCertPool.AppendCertsFromPEM(trustBytes) {
		return nil, fmt.Errorf("Error adding CA trust to pool: %v", err)
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("Error parsing cert: %v", err)
	}
	cfg := tls.Config{
		Certificates:             []tls.Certificate{cert},
		ClientCAs:                trustCertPool,
		RootCAs:                  trustCertPool,
		InsecureSkipVerify:       insecure,
		ServerName:               serverName,
		PreferServerCipherSuites: true,
	}
	cfg.BuildNameToCertificate()

	return &cfg, nil
}

// NewTLSClientConn gets a TLS connection for a client, not sharing the config
// Point NewSSLConn here when legacyssl package is gone.
func NewTLSClientConn(trustPath, certPath, keyPath, serverName, host, port string, insecure bool) (io.ReadWriteCloser, error) {
	conf, err := NewTLSClientConfig(trustPath, certPath, keyPath, serverName, insecure)
	if err != nil {
		return nil, err
	}
	return tls.Dial("tcp", fmt.Sprintf("%s:%s", host, port), conf)
}
