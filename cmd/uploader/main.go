package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
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
	hostName := "twl-server-generic2" // change this
	//hostName := "54.236.228.140"
	portNum := "7444"
	log.Printf("Connecting to %s\n", hostName)
	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}
	tlsConfig.BuildNameToCertificate()

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	resp, err := client.Get("https://" + hostName + ":" + portNum + "/stats")
	if err != nil {
		fmt.Printf("unable to connect: %v\n", err)
		return
	}
	contents, err := ioutil.ReadAll(resp.Body)
	fmt.Printf("%s\n", string(contents))
}
