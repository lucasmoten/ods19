// Copyright 2017 Decipher Technology Studios LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

// Opt sets an option on a *tls.Config
type Opt func(*tls.Config)

// Insecure sets InsecureSkipVerify.
func Insecure(b bool) Opt {
	return func(cfg *tls.Config) {
		cfg.InsecureSkipVerify = true
	}
}

// WithCN sets ServerName.
func WithCN(name string) Opt {
	return func(cfg *tls.Config) {
		cfg.ServerName = name
	}
}

// WithClientAuth sets the ClientAuth type. These are constants in package tls.
// For instance, you can pass tls.RequestClientCert
func WithClientAuth(auth tls.ClientAuthType) Opt {
	return func(cfg *tls.Config) {
		cfg.ClientAuth = auth
	}
}

// NewTLSConfig gets a config to make TLS connections.
func NewTLSConfig(trustPath, certPath, keyPath string, opts ...Opt) (*tls.Config, error) {
	trustBytes, err := ioutil.ReadFile(trustPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing CA trust %s: %v", trustPath, err)
	}
	trustCertPool := x509.NewCertPool()
	if !trustCertPool.AppendCertsFromPEM(trustBytes) {
		return nil, fmt.Errorf("error adding CA trust to pool: %v", err)
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing cert: %v", err)
	}
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    trustCertPool,
		RootCAs:      trustCertPool,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	cfg.BuildNameToCertificate()

	return cfg, nil
}

// NewTLSClientConfig gets a config to make TLS connections.
func NewTLSClientConfig(trustPath, certPath, keyPath, serverCN string) (*tls.Config, error) {
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
		Certificates:       []tls.Certificate{cert},
		ClientCAs:          trustCertPool,
		RootCAs:            trustCertPool,
		InsecureSkipVerify: false,
		ServerName:         serverCN,
	}
	cfg.BuildNameToCertificate()

	return &cfg, nil
}

// NewTLSClientConn gets a TLS connection for a client.
func NewTLSClientConn(trustPath, certPath, keyPath, serverCN, host, port string) (io.ReadWriteCloser, error) {
	conf, err := NewTLSClientConfig(trustPath, certPath, keyPath, serverCN)
	if err != nil {
		return nil, err
	}
	return tls.Dial("tcp", fmt.Sprintf("%s:%s", host, port), conf)
}

// LearnServerCN is the ONLY plausible place to see InsecureSkipVerify, because
// it just connects to get the serverCN value, and then disconnects.
func LearnServerCN(trustPath, certPath, keyPath, host, port string) (string, error) {
	conf, err := NewTLSClientConfig(trustPath, certPath, keyPath, "")
	if err != nil {
		return "", err
	}
	if conf != nil {
		conf.InsecureSkipVerify = true
		conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%s", host, port), conf)
		if conn != nil {
			conn.Close()
			cs := conn.ConnectionState()
			pc := cs.PeerCertificates
			if len(pc) > 0 {
				serverCN := pc[0].Subject.CommonName
				return serverCN, nil
			}
		}
		if err != nil && err != io.EOF {
			return "", err
		}
	}
	return "", err
}

// NewTLSClientConnFactory lets us continue to spawn TLS connections for a given host and port.
func NewTLSClientConnFactory(trustPath, certPath, keyPath, serverCN, host, port string) (*http.Client, error) {
	conf, err := NewTLSClientConfig(trustPath, certPath, keyPath, serverCN)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport: &http.Transport{
			DialTLS: func(network, address string) (net.Conn, error) {
				return tls.Dial("tcp", fmt.Sprintf("%s:%s", host, port), conf)
			},
		},
	}, nil
}

// GetDistinguishedName is the dn off of the given certificate.
func GetDistinguishedName(theCert *x509.Certificate) string {
	result := ""
	if len(theCert.Subject.CommonName) > 0 {
		result += ",CN=" + theCert.Subject.CommonName
	}
	for l := len(theCert.Subject.Locality); l > 0; l-- {
		result += ",L=" + theCert.Subject.Locality[l-1]
	}
	for p := len(theCert.Subject.Province); p > 0; p-- {
		result += ",ST=" + theCert.Subject.Province[p-1]
	}
	for ou := len(theCert.Subject.OrganizationalUnit); ou > 0; ou-- {
		result += ",OU=" + theCert.Subject.OrganizationalUnit[ou-1]
	}
	for o := len(theCert.Subject.Organization); o > 0; o-- {
		result += ",O=" + theCert.Subject.Organization[o-1]
	}
	for c := len(theCert.Subject.Country); c > 0; c-- {
		result += ",C=" + theCert.Subject.Country[c-1]
	}
	for street := len(theCert.Subject.StreetAddress); street > 0; street-- {
		result += ",STREET=" + theCert.Subject.StreetAddress[street-1]
	}
	if len(result) > 0 {
		result = result[1:len(result)]
	}

	return result
}

// GetCommonName returns the CN value part of a passed in distinguished name.
func GetCommonName(DistinguishedName string) string {
	if DistinguishedName == "" {
		return ""
	}
	dnParts := strings.Split(DistinguishedName, ",")
	for _, s := range dnParts {
		if strings.Index(strings.ToUpper(s), "CN=") == 0 {
			return s[3:len(s)]
		}
	}
	return DistinguishedName
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

// GetNormalizedDistinguishedName returns a normalized distinguished name that
// reverses the apache format and comma delimits.
func GetNormalizedDistinguishedName(distinguishedName string) string {
	if strings.Index(distinguishedName, "/") == -1 {
		// assume already in appropriate format
		return distinguishedName
	}

	dnParts := strings.Split(distinguishedName, "/")
	result := ""
	for p := len(dnParts); p > 0; p-- {
		if len(dnParts[p-1]) > 0 {
			result += "," + dnParts[p-1]
		}
	}
	if len(result) > 0 {
		result = result[1:len(result)]
	}
	return result
}
