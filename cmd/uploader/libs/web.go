package libs

import (
	"crypto/tls"
	"decipher.com/oduploader/config"
	"net/http"
	//"time"
	//"bytes"
	"crypto/x509/pkix"
	"fmt"
	//"log"
	"strconv"
)

// Uploader is the state that the server references
type Uploader struct {
	Environment *config.Environment
	TLS         *tls.Config
}

// CreateUploader constructs an uploader object
func CreateUploader(env *config.Environment, tls *tls.Config) *Uploader {
	return &Uploader{
		Environment: env,
		TLS:         tls,
	}
}

// CreateUploadServer generates an http server with the Uploader as handler
func (u Uploader) CreateUploadServer() *http.Server {
	addr :=
		u.Environment.TCPBind +
			":" +
			strconv.Itoa(u.Environment.TCPPort)

	return &http.Server{
		Addr:      addr,
		Handler:   u,
		TLSConfig: u.TLS,
		/*
			ReadTimeout:    10000 * time.Second, //This breaks big downloads
			WriteTimeout:   10000 * time.Second,
			MaxHeaderBytes: 1 << 20, //This prevents clients from DOS'ing us
		*/
	}
}

// ServeHTTP is the muxer for this http server
func (u Uploader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello: %s", getIdentity(r))
}

func getIdentity(r *http.Request) string {
	//TODO: otherwise, pull from headers
	if r.TLS != nil {
		if r.TLS.PeerCertificates != nil {
			if len(r.TLS.PeerCertificates) > 0 {
				return getDNFromCert(r.TLS.PeerCertificates[0].Subject)
			}
		}
	}
	return "anonymous"
}

// idenfity the user the user - start with DN, then go through headers
func getDNFromCert(name pkix.Name) string {
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
			case dnPart[j].Type.String() == "2.5.4.6":
				pPart = "C"
			case dnPart[j].Type.String() == "2.5.4.10":
				pPart = "O"
			case dnPart[j].Type.String() == "2.5.4.11":
				pPart = "OU"
			case dnPart[j].Type.String() == "2.5.4.3":
				pPart = "CN"
			}
			dnArray = dnArray + fmt.Sprintf("%s=%v", pPart, dnPart[j].Value)
		}
	}
	return dnArray
}
