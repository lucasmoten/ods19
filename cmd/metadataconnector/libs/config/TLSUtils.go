package config

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func buildClientTLSConfig(CAPath string, ClientCertPath string, ClientKeyPath string, ServerName string, InsecureSkipVerify bool) tls.Config {
	// Root Certificate pool
	// The set of root certificate authorities that this client will use when
	// verifying the server certificate indicated as the identity of the
	// server this config will be used to connect to.
	rootCAsCertPool := buildCertPoolFromPath(CAPath)

	// Client public and private certificate
	clientCert := buildx509Identity(ClientCertPath, ClientKeyPath)

	return tls.Config{
		RootCAs:            rootCAsCertPool,
		Certificates:       clientCert,
		ServerName:         ServerName,
		InsecureSkipVerify: InsecureSkipVerify,
	}
}

func buildServerTLSConfig(CAPath string, ServerCertPath string, ServerKeyPath string, RequireClientCert bool, CipherSuites []string, MinimumVersion string) tls.Config {
	// Client Certificate pool
	// The set of root certificate authorities that the sever will use to verify
	// client certificates
	clientCAsCertPool := buildCertPoolFromPath(CAPath)

	// Server public and private certificate
	serverCert := buildx509Identity(ServerCertPath, ServerKeyPath)

	clientAuthType := tls.NoClientCert
	if RequireClientCert {
		clientAuthType = tls.RequireAndVerifyClientCert
	}

	preferServerCipherSuites := false
	cipherSuites := buildCipherSuites(CipherSuites)
	if len(cipherSuites) > 0 {
		preferServerCipherSuites = true
	}

	var minimumVersion uint16
	minimumVersion = tls.VersionTLS10
	if MinimumVersion == "1.1" {
		minimumVersion = tls.VersionTLS11
	}
	if MinimumVersion == "1.2" {
		minimumVersion = tls.VersionTLS12
	}

	return tls.Config{
		Certificates:             serverCert,
		ClientAuth:               clientAuthType,
		ClientCAs:                clientCAsCertPool,
		CipherSuites:             cipherSuites,
		PreferServerCipherSuites: preferServerCipherSuites,
		MinVersion:               minimumVersion,
	}
}

/*
var cipherNameConstLookup = map[uint16]string{
  tls.TLS_RSA_WITH_RC4_128_SHA                : `TLS_RSA_WITH_RC4_128_SHA`,
  tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA           : `TLS_RSA_WITH_3DES_EDE_CBC_SHA`,
  tls.TLS_RSA_WITH_AES_128_CBC_SHA            : `TLS_RSA_WITH_AES_128_CBC_SHA`,
  tls.TLS_RSA_WITH_AES_256_CBC_SHA            : `TLS_RSA_WITH_AES_256_CBC_SHA`,
  tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA        : `TLS_ECDHE_ECDSA_WITH_RC4_128_SHA`,
  tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA    : `TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA`,
  tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA    : `TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA`,
  tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA          : `TLS_ECDHE_RSA_WITH_RC4_128_SHA`,
  tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA     : `TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA`,
  tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA      : `TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA`,
  tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA      : `TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA`,
  tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256   : `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`,
  tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256 : `TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256`,
  tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384   : `TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384`,
  tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384 : `TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384`,
}
*/

func buildCipherSuites(CipherSuiteNames []string) []uint16 {
	var cipherSuites []uint16
	var cipherValueConstLookup = map[string]uint16{
		`TLS_RSA_WITH_RC4_128_SHA`:                tls.TLS_RSA_WITH_RC4_128_SHA,
		`TLS_RSA_WITH_3DES_EDE_CBC_SHA`:           tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		`TLS_RSA_WITH_AES_128_CBC_SHA`:            tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		`TLS_RSA_WITH_AES_256_CBC_SHA`:            tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		`TLS_ECDHE_ECDSA_WITH_RC4_128_SHA`:        tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
		`TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA`:    tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		`TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA`:    tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		`TLS_ECDHE_RSA_WITH_RC4_128_SHA`:          tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
		`TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA`:     tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		`TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA`:      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		`TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA`:      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		`TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`:   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		`TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256`: tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		`TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384`:   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		`TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384`: tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	}
	for i := 0; i < len(CipherSuiteNames); i++ {
		cipherSuites = append(cipherSuites, cipherValueConstLookup[CipherSuiteNames[i]])
	}
	return cipherSuites
}

func buildx509Identity(certFile string, keyFile string) []tls.Certificate {
	theCert := make([]tls.Certificate, 0, 1)
	certs, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}
	theCert = append(theCert, certs)
	return theCert
}

func buildCertPoolFromPath(filePath string) *x509.CertPool {

	theCertPool := x509.NewCertPool()

	// Open path indicated in configuration
	pathSpec, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer pathSpec.Close()

	// Check information about the path specification
	pathSpecInfo, err := pathSpec.Stat()
	if err != nil {
		log.Fatal(err.Error())
	}

	// Handle cases based on the type of path
	switch mode := pathSpecInfo.Mode(); {
	case mode.IsDir():
		// The path is a directory, read all the files
		files, err := ioutil.ReadDir(filePath)
		if err != nil {
			log.Fatal(err.Error())
		}
		if !strings.HasSuffix(filePath, "/") {
			filePath += "/"
		}
		// With each file
		for f := 0; f < len(files); f++ {
			addPEMFileToPool(filePath+files[f].Name(), theCertPool)
		}
	case mode.IsRegular():
		addPEMFileToPool(filePath, theCertPool)
	}

	return theCertPool
}

func addPEMFileToPool(PEMfile string, certPool *x509.CertPool) {
	log.Println("Adding PEM file " + PEMfile + " to certificate pool")
	pem, err := ioutil.ReadFile(PEMfile)
	if err != nil {
		log.Fatal(err.Error())
	}
	if ok := certPool.AppendCertsFromPEM(pem); !ok {
		log.Fatal("Failed to append PEM.")
	}
}

/*
GetDistinguishedName returns the common formatted distinguished name built up
from the sets of attributes on the certificate subject.
*/
func GetDistinguishedName(theCert *x509.Certificate) string {
	result := ""
	if len(theCert.Subject.CommonName) > 0 {
		result += ", CN=" + theCert.Subject.CommonName
	}
	for l := 0; l < len(theCert.Subject.Locality); l++ {
		result += ", L=" + theCert.Subject.Locality[l]
	}
	for p := 0; p < len(theCert.Subject.Province); p++ {
		result += ", ST=" + theCert.Subject.Province[p]
	}
	for o := 0; o < len(theCert.Subject.Organization); o++ {
		result += ", O=" + theCert.Subject.Organization[o]
	}
	for ou := 0; ou < len(theCert.Subject.OrganizationalUnit); ou++ {
		result += ", OU=" + theCert.Subject.OrganizationalUnit[ou]
	}
	for c := 0; c < len(theCert.Subject.Country); c++ {
		result += ", C=" + theCert.Subject.Country[c]
	}
	for street := 0; street < len(theCert.Subject.StreetAddress); street++ {
		result += ", STREET=" + theCert.Subject.StreetAddress[street]
	}
	if len(result) > 0 {
		result = result[2:len(result)]
	}
	return result
}
