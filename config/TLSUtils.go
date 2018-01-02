package config

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"os"
	"strings"

	"go.uber.org/zap"
)

// buildServerTLSConfig prepares a tls.Config object for this application to
// listen for connecting clients.
func buildServerTLSConfig(caPath, certPath, keyPath string, clientCert bool, ciphers []string, minVersion string) tls.Config {
	// Client Certificate pool
	// The set of root certificate authorities that the sever will use to verify
	// client certificates
	clientCAsCertPool := buildCertPoolFromPath(caPath, "for server")

	// Server public and private certificate
	serverCert := buildx509Identity(certPath, keyPath)

	clientAuthType := tls.NoClientCert
	if clientCert {
		clientAuthType = tls.RequireAndVerifyClientCert
	}

	preferServerCiphers := false
	cipherSuites := buildCipherSuites(ciphers)
	if len(cipherSuites) > 0 {
		preferServerCiphers = true
	}

	// Prefer TLS 1.2. Allow 1.1 or 1.0
	var minimumVersion uint16
	minimumVersion = tls.VersionTLS12
	if minVersion == "1.1" {
		minimumVersion = tls.VersionTLS11
	}
	if minVersion == "1.0" {
		minimumVersion = tls.VersionTLS10
	}
	switch minimumVersion {
	case tls.VersionTLS10:
		logger.Info("tls minversion set", zap.String("ver", "1.0"))
	case tls.VersionTLS11:
		logger.Info("tls minversion set", zap.String("ver", "1.1"))
	case tls.VersionTLS12:
		logger.Info("tls minversion set", zap.String("ver", "1.2"))
	}

	return tls.Config{
		Certificates:             serverCert,
		ClientAuth:               clientAuthType,
		ClientCAs:                clientCAsCertPool,
		CipherSuites:             cipherSuites,
		PreferServerCipherSuites: preferServerCiphers,
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
			clogger := logger.With(zap.String("suite", CipherSuiteNames[i]))
			if v > 0 {
				clogger.Info("enabling cipher suite")
				cipherSuites = append(cipherSuites, v)
			} else {
				clogger.Info("cipher suite not known")
			}
		}
	} else {
		for key, value := range cipherValueConstLookup {
			logger.Info("enabling cipher suite", zap.String("suite", key))
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
		logger.Info(
			"error loading x509 key pair",
			zap.Error(err),
			zap.String("certfile", certFile),
			zap.String("keyfile", keyFile),
		)
	} else {
		theCert = append(theCert, certs)
	}
	return theCert
}

// buildCertPoolFromPath prepares a certificate pool from the passed in file
// path. If the file path is an indivdual file, then a single PEM is placed
// in the pool. If it is a folder, then all files in the folder are added to the pool.
func buildCertPoolFromPath(filePath string, poolName string) *x509.CertPool {
	flogger := logger.With(zap.String("filepath", filePath)).With(zap.String("pool", poolName))
	flogger.Info("preparing certificate pool")
	theCertPool := x509.NewCertPool()

	// Open path indicated in configuration
	pathSpec, err := os.Open(filePath)
	if err != nil {
		flogger.Error("error opening file path", zap.Error(err))
		return theCertPool

	}
	defer pathSpec.Close()

	// Check information about the path specification
	pathSpecInfo, err := pathSpec.Stat()
	if err != nil {
		flogger.Error("error retrieving path specification information", zap.Error(err))
		return theCertPool
	}

	// Handle cases based on the type of path
	switch mode := pathSpecInfo.Mode(); {
	case mode.IsDir():
		// The path is a directory, read all the files
		files, err := ioutil.ReadDir(filePath)
		if err != nil {
			flogger.Error("reading directory", zap.Error(err))
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
	plogger := logger.With(zap.String("pem", PEMfile))
	plogger.Info("adding pem file")
	pem, err := ioutil.ReadFile(PEMfile)
	if err != nil {
		plogger.Error("error reading pem file", zap.Error(err))
		return
	}
	if ok := certPool.AppendCertsFromPEM(pem); !ok {
		plogger.Error("failed to append the PEM to the pool")
		return
	}
}

// GetDistinguishedName returns the common formatted distinguished name built up
// from the sets of attributes on the certificate subject.
// TODO: Callers will eventually make user of user_dn header value from NGINX
func GetDistinguishedName(theCert *x509.Certificate) string {
	result := ""
	if len(theCert.Subject.CommonName) > 0 {
		result += ",cn=" + theCert.Subject.CommonName
	}
	for l := len(theCert.Subject.Locality); l > 0; l-- {
		result += ",l=" + theCert.Subject.Locality[l-1]
	}
	for p := len(theCert.Subject.Province); p > 0; p-- {
		result += ",st=" + theCert.Subject.Province[p-1]
	}
	for ou := len(theCert.Subject.OrganizationalUnit); ou > 0; ou-- {
		result += ",ou=" + theCert.Subject.OrganizationalUnit[ou-1]
	}
	for o := len(theCert.Subject.Organization); o > 0; o-- {
		result += ",o=" + theCert.Subject.Organization[o-1]
	}
	for c := len(theCert.Subject.Country); c > 0; c-- {
		result += ",c=" + theCert.Subject.Country[c-1]
	}
	for street := len(theCert.Subject.StreetAddress); street > 0; street-- {
		result += ",street=" + theCert.Subject.StreetAddress[street-1]
	}
	if len(result) > 0 {
		result = result[1:len(result)]
	}

	return result
}

// GetNormalizedDistinguishedName returns a normalized distinguished name that
// reverses the apache format and comma delimits.
// Logic rewritten to be modeled after https://gitlab.363-283.io/cte/cte-service-framework/blob/develop/core/src/main/scala/gov/ic/cte/server/security/DNHelper.scala
func GetNormalizedDistinguishedName(distinguishedName string) string {
	if len(distinguishedName) == 0 {
		return distinguishedName
	}

	replaced := strings.Replace(distinguishedName, "/", ",", -1)
	splitOut := strings.Split(replaced, ",")
	validCount := getCount(splitOut)
	trimmed := trim(splitOut, validCount)

	if len(trimmed) == 0 {
		return ""
	}

	// Don't have to worry about case since 'trim' toLowers as it trims
	tmp := ""
	if strings.HasPrefix(trimmed[0], "cn") {
		tmp = strings.Join(trimmed, ",")
	} else {
		var rtmp []string
		for r := range trimmed {
			rtmp = append(rtmp, trimmed[len(trimmed)-1-r])
		}
		tmp = strings.Join(rtmp, ",")
	}
	return tmp

}

func trim(v []string, max int) []string {
	if max > 0 {
		var tmp []string
		for _, t := range v {
			if len(strings.TrimSpace(t)) > 0 {
				tmp = append(tmp, strings.ToLower(strings.TrimSpace(t)))
			}
		}
		return tmp
	}
	return v
}
func getCount(v []string) int {
	count := 0
	for _, t := range v {
		if len(strings.TrimSpace(t)) > 0 {
			count++
		}
	}
	return count
}

// GetCommonName returns the CN value part of a passed in distinguished name
func GetCommonName(DistinguishedName string) string {
	if DistinguishedName == "" {
		return ""
	}
	dnParts := strings.Split(DistinguishedName, ",")
	for _, s := range dnParts {
		if strings.Index(strings.ToLower(s), "cn=") == 0 {
			return s[3:len(s)]
		}
	}

	return DistinguishedName
}
