package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	//Load trust file
	certBytes, err := ioutil.ReadFile("../cryptotest/defaultcerts/clients/client.trust.pem")
	if err != nil {
		log.Printf("Unable to trust file: %v\n", err)
		return
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(certBytes); !ok {
		log.Printf("Unable to add certificate to certificate pool: %v\n", ok)
		return
	}

	//Load client key pair
	cert, err := tls.LoadX509KeyPair(
		"../cryptotest/defaultcerts/clients/test_1.cert.pem",
		"../cryptotest/defaultcerts/clients/test_1.key.pem",
	)
	if err != nil {
		log.Printf("could not parse client cert: %v\n", cert)
		return
	}

	//Actually connect
	//hostName := "dockervm" // change this
	hostName := "twl-server-generic2"
	portNum := "7444"
	log.Printf("Connecting to %s\n", hostName)
	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
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
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("unable to connect: %v\n", err)
		return
	}
	log.Printf("got:%s\n", string(contents))
}
