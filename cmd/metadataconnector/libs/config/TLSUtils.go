package config

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// buildClientTLSConfig prepares a tls.Config object for this application to use
// when acting as a client connecting to a dependent resource.
func buildClientTLSConfig(CAPath string, ClientCertPath string, ClientKeyPath string, ServerName string, InsecureSkipVerify bool) tls.Config {
	// Root Certificate pool
	// The set of root certificate authorities that this client will use when
	// verifying the server certificate indicated as the identity of the
	// server this config will be used to connect to.
	rootCAsCertPool := buildCertPoolFromPath(CAPath, "for client")

	// Client public and private certificate
	if len(ClientCertPath) == 0 || len(ClientKeyPath) == 0 {
		return tls.Config{
			RootCAs:            rootCAsCertPool,
			ServerName:         ServerName,
			InsecureSkipVerify: InsecureSkipVerify,
		}
	} else {
		clientCert := buildx509Identity(ClientCertPath, ClientKeyPath)

		return tls.Config{
			RootCAs:            rootCAsCertPool,
			Certificates:       clientCert,
			ServerName:         ServerName,
			InsecureSkipVerify: InsecureSkipVerify,
		}
	}
}

// buildServerTLSConfig prepares a tls.Config object for this application to
// listen for connecting clients.
func buildServerTLSConfig(CAPath string, ServerCertPath string, ServerKeyPath string, RequireClientCert bool, CipherSuites []string, MinimumVersion string) tls.Config {
	// Client Certificate pool
	// The set of root certificate authorities that the sever will use to verify
	// client certificates
	clientCAsCertPool := buildCertPoolFromPath(CAPath, "for server")

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
	switch minimumVersion {
	case tls.VersionTLS10:
		log.Println("TLS MinVersion set to 1.0")
	case tls.VersionTLS11:
		log.Println("TLS MinVersion set to 1.1")
	case tls.VersionTLS12:
		log.Println("TLS MinVersion set to 1.2")
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

// buildCipherSuites takes a passed in array of cipher names and returns back
// the mapped cipher id value. If the passed in array is empty, then all ciphers
// known in the map are added.
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
	if len(CipherSuiteNames) > 0 {
		for i := 0; i < len(CipherSuiteNames); i++ {
			v := cipherValueConstLookup[CipherSuiteNames[i]]
			if v > 0 {
				log.Println("Enabling cipher suite: " + CipherSuiteNames[i])
				cipherSuites = append(cipherSuites, v)
			} else {
				log.Println("WARN: Cipher suite `" + CipherSuiteNames[i] + "` declared in configuration is not known to this system.")
			}
		}
	} else {
		log.Println("WARN: CipherSuites not declared in configuration. Adding all known cipher suites.")
		log.Println("WARN: This is inherently less secure as it may be overly permissive by enabling weaker ciphers")
		for key, value := range cipherValueConstLookup {
			log.Println("Enabling cipher suite: " + key)
			cipherSuites = append(cipherSuites, value)
		}
	}
	return cipherSuites
}

// buildx509Identity takes the path of a public and private certificate file in
// PEM format and loads as a standard tls.Certificate in response
func buildx509Identity(certFile string, keyFile string) []tls.Certificate {
	theCert := make([]tls.Certificate, 0, 1)
	certs, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Printf("Error loading x509 Key Pair: %s", err.Error())
	} else {
		theCert = append(theCert, certs)
	}
	return theCert
}

// buildCertPoolFromPath prepares a certificate pool from the passed in file
// path. If the file path is an indivdual file, then a single PEM is placed
// in the pool. If it is a folder, then all files in the folder are read to See
// if they are PEM files, and if so, added to the pool.
func buildCertPoolFromPath(filePath string, poolName string) *x509.CertPool {

	log.Println("Preparing certificate pool " + poolName)
	theCertPool := x509.NewCertPool()

	// Open path indicated in configuration
	pathSpec, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening filepath: %s", err.Error())
		return theCertPool

	}
	defer pathSpec.Close()

	// Check information about the path specification
	pathSpecInfo, err := pathSpec.Stat()
	if err != nil {
		log.Printf("Error retrieving path specification information: %s", err.Error())
		return theCertPool
	}

	// Handle cases based on the type of path
	switch mode := pathSpecInfo.Mode(); {
	case mode.IsDir():
		// The path is a directory, read all the files
		files, err := ioutil.ReadDir(filePath)
		if err != nil {
			log.Printf("Error reading directory: %s", err.Error())
			return theCertPool
		}
		if !strings.HasSuffix(filePath, "/") {
			filePath += "/"
		}
		// With each file
		for f := 0; f < len(files); f++ {
			if !files[f].IsDir() {
				addPEMFileToPool(filePath+files[f].Name(), theCertPool)
			}
		}
	case mode.IsRegular():
		addPEMFileToPool(filePath, theCertPool)
	}

	return theCertPool
}

// addPEMFileToPool takes a file path representing a certificate in PEM format
// and appends it to the passed in certificate pool. Intended for building up
// a certificate pool of trusted certificate authorities
func addPEMFileToPool(PEMfile string, certPool *x509.CertPool) {
	log.Println("Adding PEM file " + PEMfile)
	pem, err := ioutil.ReadFile(PEMfile)
	if err != nil {
		log.Printf("Error reading PEM file: %s", err.Error())
		return
	}
	if ok := certPool.AppendCertsFromPEM(pem); !ok {
		log.Println("Failed to append the PEM to the pool")
		return
	}
}

// GetDistinguishedName returns the common formatted distinguished name built up
// from the sets of attributes on the certificate subject.
// TODO: Callers will eventually make user of user_dn header value from NGINX
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

// GetNormalizedDistinguishedName returns a normalized distinguished name that
// reverses the apache format and comma delimits.
func GetNormalizedDistinguishedName(distinguishedName string) string {
	if strings.Index(distinguishedName, "/") == -1 {
		// assume already in appropriate format
		return distinguishedName
	}

	// Apache format...
	//  in:  /C=US/O=U.S. Government/OU=chimera/OU=DAE/OU=People/CN=test tester10
	// out: CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US
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

// GetCommonName returns the CN value part of a passed in distinguished name
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
