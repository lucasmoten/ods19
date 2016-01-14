package integration

import (
	"crypto/x509"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestReadPythonSourcedCertificate(t *testing.T) {
	// TODO adjust this path and fix this test
	t.Skip("Not yet implemented")
	path := "/Users/cmcfarland/Code/certtool/certs/output/*.asn1"
	files, err := filepath.Glob(path)
	if err != nil {
		t.FailNow()
	}
	_ = files

	pool := x509.NewCertPool()

	for _, item := range files {
		b, err := ioutil.ReadFile(item)
		if err != nil {
			t.Fatal(err)
		}
		cert, err := x509.ParseCertificate(b)
		if err != nil {
			t.Fatal(err)
		}
		pool.AddCert(cert)
	}

}
