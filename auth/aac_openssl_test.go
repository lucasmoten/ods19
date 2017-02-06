package auth_test

import (
	"crypto/tls"
	"fmt"
	"path/filepath"
	"testing"

	"decipher.com/object-drive-server/legacyssl"
)

// Note: this test MUST wait for stallForAvailability because it talks to aac directly
//
// Note: with the new certs, we were able to connect without openssl!!!
// Consider getting rid of it once we have stabilized under the new certs.
//
func TestLegacySSLUnnecessary(t *testing.T) {
	if testing.Short() {
		return
	}

	trustPath := filepath.Join("..", "defaultcerts", "client-aac", "trust", "client.trust.pem")
	certPath := filepath.Join("..", "defaultcerts", "client-aac", "id", "client.cert.pem")
	keyPath := filepath.Join("..", "defaultcerts", "client-aac", "id", "client.key.pem")

	// AAC trust, client public & private key
	tlsConfig, err := legacyssl.NewTLSConfigForTest(
		t.Logf,
		trustPath,
		certPath,
		keyPath,
	)
	if err != nil {
		t.Logf("cannot create config: %v", err)
		t.FailNow()
	}
	//to := fmt.Sprintf("%s:%d", "192.168.99.100", 4430)
	to := fmt.Sprintf("%s:%d", "aac", 9093)
	conn, err := tls.Dial("tcp", to, tlsConfig)
	if err == nil {
		cs := conn.ConnectionState()
		t.Logf("connectionState.Version: %v", cs.Version)
		t.Logf("connectionState.NegotiatedProtocol: %v", cs.NegotiatedProtocol)
		t.Logf("connectionState.DidResume: %v", cs.DidResume)
		t.Logf("connectionState.NegotiatedProtocolIsMutual: %v", cs.NegotiatedProtocolIsMutual)
		t.Logf("connectionState.CipherSuite: %v", cs.CipherSuite)
		t.Logf("connectionState.PeerCertificates:")
		legacyssl.LogCertificateChain(t.Logf, cs.PeerCertificates)
		t.Logf("connectionState.VerifiedChains:")
		legacyssl.LogCertificateChains(t.Logf, cs.VerifiedChains)
		t.Logf("connectionState.ServerName: %v", cs.ServerName)
		t.Logf("connectionState.SignedCertificateTimestamps: %v", cs.SignedCertificateTimestamps)
		t.Logf("connectionState.OCSPResponse: %v", cs.OCSPResponse)
		t.Logf("connectionState.TLSUnique: %v", cs.TLSUnique)
		conn.Close()
		t.Logf("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		t.Logf("hey, direct tls connect to AAC worked! start eliminating cgo if you can!")
		t.Logf("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		//t.FailNow()
	} else {
		t.Logf("expected inability to connect to aac without openssl: %v", err)
	}
}
