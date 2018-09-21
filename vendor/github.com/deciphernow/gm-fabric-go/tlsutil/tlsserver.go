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
	"fmt"
	"io/ioutil"
)

// BuildServerTLSConfig creates a TLS config for a server.
func BuildServerTLSConfig(caPath, certPath, keyPath string) (*tls.Config, error) {
	cfg := tls.Config{}

	serverCert, err := buildx509Identity(certPath, keyPath)
	if err != nil {
		return &cfg, err
	}
	cfg.Certificates = serverCert
	cfg.ClientAuth = tls.RequireAndVerifyClientCert

	theCertPool := x509.NewCertPool()
	addPEMFileToPool(caPath, theCertPool)
	cfg.ClientCAs = theCertPool

	return &cfg, nil
}

func buildx509Identity(certPath string, keyPath string) ([]tls.Certificate, error) {
	theCert := make([]tls.Certificate, 0, 1)
	certs, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	theCert = append(theCert, certs)
	return theCert, nil
}

// addPEMFileToPool takes a file path representing a certificate in PEM format
// and appends it to the passed in certificate pool. Intended for building up
// a certificate pool of trusted certificate authorities
func addPEMFileToPool(pemFile string, certPool *x509.CertPool) error {
	pem, err := ioutil.ReadFile(pemFile)
	if err != nil {
		return err
	}
	if ok := certPool.AppendCertsFromPEM(pem); !ok {
		return fmt.Errorf("unable to append %s into cert pool", pemFile)
	}
	return nil
}
