package legacyssl

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
)

type LoggingFunction func(string, ...interface{})

func completeChain(id *x509.Certificate, trusts []*x509.Certificate) []*x509.Certificate {
	var chain []*x509.Certificate
	current := id

	for {
		// put the current cert in the chain
		chain = append(chain, current)
		subject := GetDNFromCert(current.Subject)
		issuer := GetDNFromCert(current.Issuer)
		if subject == issuer {
			return chain
		}
		// look for a matching subject - brute force
		for i := 0; i < len(trusts); i++ {
			thissubject := GetDNFromCert(trusts[i].Subject)
			if thissubject == issuer {
				current = trusts[i]
				break
			}
		}
	}
}

func findInChain(subject string, trusts []*x509.Certificate, logit bool) *x509.Certificate {
	for i := 0; i < len(trusts); i++ {
		thissubject := GetDNFromCert(trusts[i].Subject)
		if thissubject == subject {
			return trusts[i]
		}
	}
	return nil
}

func parseCertsFromPEM(logf LoggingFunction, pemCerts []byte) []*x509.Certificate {
	retval := make([]*x509.Certificate, 0)
	for len(pemCerts) > 0 {
		var block *pem.Block
		block, pemCerts = pem.Decode(pemCerts)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			logf("error parsing cert: %v", err)
			continue
		}

		retval = append(retval, cert)
	}

	return retval
}

// NewTLSConfigForTest is used by a test to see if we can get rid of openssl
func NewTLSConfigForTest(logf LoggingFunction, trustPath, certPath, keyPath string) (*tls.Config, error) {
	trustBytes, err := ioutil.ReadFile(trustPath)
	if err != nil {
		return nil, fmt.Errorf("Error parsing CA trust %s: %v", trustPath, err)
	}
	trustCertPool := x509.NewCertPool()
	if !trustCertPool.AppendCertsFromPEM(trustBytes) {
		return nil, fmt.Errorf("Error adding CA trust to pool: %v", err)
	}
	x509parsedCerts := parseCertsFromPEM(logf, trustBytes)

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("Error parsing cert: %v", err)
	}
	x509cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		logf("cant parse cert: %v", err)
		return nil, err
	}
	// build and check our cert chain
	if x509cert == nil {
		panic(fmt.Sprintf("cannot complete chain on a nil cert from %v", cert.Certificate[0]))
	}

	certs := []tls.Certificate{cert}

	opts := x509.VerifyOptions{
		Intermediates: trustCertPool,
		Roots:         trustCertPool,
	}

	x509certs := completeChain(x509cert, x509parsedCerts)
	logf("our cert chain:")
	LogCertificateChain(logf, x509certs)

	ourChains, err := x509cert.Verify(opts)
	if err != nil {
		logf("error verifying chain: %v", err)
		return nil, err
	}
	logf("our cert isValid:")
	LogCertificateChains(logf, ourChains)

	cfg := tls.Config{
		Certificates:             certs,
		ClientCAs:                trustCertPool,
		RootCAs:                  trustCertPool,
		InsecureSkipVerify:       false,
		ServerName:               "twl-server-generic2",
		PreferServerCipherSuites: true,
		ClientAuth:               tls.RequireAndVerifyClientCert,
	}
	cfg.BuildNameToCertificate()

	return &cfg, nil
}

func logCertificateExtensions(logf LoggingFunction, ids []pkix.Extension) {
	extensionMap := make(map[string]string)
	extensionMap["2.5.29.14"] = "Subject Key Identifier"
	extensionMap["2.5.29.15"] = "Key Usage"
	extensionMap["2.5.29.19"] = "Basic Constraints"
	extensionMap["2.5.29.35"] = "Authority Key Identifier"
	extensionMap["2.5.29.37"] = "Extended Key Usage"

	for i := 0; i < len(ids); i++ {
		k := ids[i].Id.String()
		b := ids[i].Critical
		v := hex.EncodeToString(ids[i].Value)
		interpretation := ""
		if k == "2.5.29.15" {
			if ids[i].Value[0]&0x01 != 0 {
				interpretation += ",digitalSig"
			}
			if ids[i].Value[0]&0x02 != 0 {
				interpretation += ",nonRepudiation"
			}
			if ids[i].Value[0]&0x04 != 0 {
				interpretation += ",keyEncipherment"
			}
			if ids[i].Value[0]&0x08 != 0 {
				interpretation += ",dataEncipherment"
			}
			if ids[i].Value[0]&0x10 != 0 {
				interpretation += ",keyAgreement"
			}
			if ids[i].Value[0]&0x20 != 0 {
				interpretation += ",keyCertSign"
			}
			if ids[i].Value[0]&0x40 != 0 {
				interpretation += ",cRLSign"
			}
		}
		logf("      %v:%v %v %v %v", extensionMap[k], k, b, v, interpretation)
	}
}

func logCertificateExtensionsAsn1(logf LoggingFunction, ids []asn1.ObjectIdentifier) {
	for i := 0; i < len(ids); i++ {
		logf("      %v", ids[i])
	}
}

func LogCertificateChain(logf LoggingFunction, chain []*x509.Certificate) {
	for i := 0; i < len(chain); i++ {
		logf("  chain:%d", i)
		logf("    Subject: %s", GetDNFromCert(chain[i].Subject))
		logf("    Issuer: %s", GetDNFromCert(chain[i].Issuer))
		logf("    Signature: %s", hex.EncodeToString(chain[i].Signature)[:8])
		logf("    Extensions:")
		logCertificateExtensions(logf, chain[i].Extensions)
		logf("    ExtraExtensions:")
		logCertificateExtensions(logf, chain[i].ExtraExtensions)
		logf("    UnhandledCriticalExtensions:")
		logCertificateExtensionsAsn1(logf, chain[i].UnhandledCriticalExtensions)
	}
}

func LogCertificateChains(logf LoggingFunction, chains [][]*x509.Certificate) {
	for i := 0; i < len(chains); i++ {
		logf("  newChain:")
		LogCertificateChain(logf, chains[i])
	}
}
