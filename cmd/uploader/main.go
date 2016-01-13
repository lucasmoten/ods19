package main

import (
	//"bytes"
	"path/filepath"

	"decipher.com/oduploader/config"
	openssl "github.com/spacemonkeygo/openssl"
	//"crypto/tls"
	//"crypto/x509"
	//"encoding/json"
	"io/ioutil"
	"log"
	//"net/http"
)

func main() {

	//Load client key pair
	ctx, err := openssl.NewCtx()
	if err != nil {
		log.Printf("Unable to create openssl context: %v", ctx)
	}
	err = ctx.LoadVerifyLocations(
		filepath.Join(config.CertsDir, "clients", "client.trust.pem"),
		filepath.Join(config.CertsDir, "clients", "client.trust.pem"),
	)
	if err != nil {
		log.Printf("Unable to load trust: %v", err)
	}

	certBytes, err := ioutil.ReadFile(filepath.Join(config.CertsDir, "clients", "test_1.cert.pem"))
	if err != nil {
		log.Printf("Unable to trust file: %v\n", err)
		return
	}

	cert, err := openssl.LoadCertificateFromPEM(certBytes)
	if err != nil {
		log.Printf("Unable to parse cert:%v", err)
	}
	ctx.UseCertificate(cert)

	keyBytes, err := ioutil.ReadFile(filepath.Join(config.CertsDir, "clients", "test_1.key.pem"))
	if err != nil {
		log.Printf("Unable to key file: %v\n", err)
		return
	}
	privKey, err := openssl.LoadPrivateKeyFromPEM(keyBytes)
	if err != nil {
		log.Printf("Unable to parse private key:%v", nil)
	}
	ctx.UsePrivateKey(privKey)

	//Actually connect
	//hostName := "dockervm" // change this
	hostName := "twl-server-generic2"
	portNum := "7444"
	log.Printf("Connecting to %s\n", hostName)

	conn, err := openssl.Dial("tcp", hostName+":"+portNum, ctx, 0)
	if err != nil {
		log.Printf("Cannot connect: %v", err)
	}
	log.Printf("connected:%v", conn)
	/*
		tlsConfig := &tls.Config{
			RootCAs:      caCertPool,
			ClientCAs:    caCertPool,
			Certificates: []tls.Certificate{cert},
		}
		tlsConfig.BuildNameToCertificate()

		transport := &http.Transport{TLSClientConfig: tlsConfig}
		client := &http.Client{Transport: transport}
		//resp, err := client.Get("https://" + hostName + ":" + portNum + "/capco/allcapcodata?userTokenType=pki_dias")

		var data map[string]json.RawMessage
		jsonStr := []byte(` { "userToken": "C=us, O=U.S. Government, OU=Chimera, OU=DAS, OU=people, CN=test tester01", "userTokenType": "pki_dias", "acm": { "version": "2.1.0", "classif": "U" } } `)
		err = json.Unmarshal(jsonStr, &data)
		if err != nil {
			log.Printf("%v", err)
		}
		url := "https://" + hostName + ":" + portNum + "/acm/checkaccess"
		req, err := http.NewRequest(
			"POST", url,
			bytes.NewBuffer(jsonStr),
		)
		if err != nil {
			log.Printf("unable to make request: %v\n", err)
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("unable to connect: %v\n", err)
			return
		}
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("unable to connect: %v\n", err)
			return
		}
		log.Printf("got:%s\n", string(contents))
	*/
}
