package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"decipher.com/oduploader/protocol"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"time"
)

// ClientIdentity is a user that is going to connect to oru service
type ClientIdentity struct {
	TrustPem      string
	CertPem       string
	KeyPem        string
	Config        *tls.Config
	Name          string
	UploadCache   string
	DownloadCache string
	Index         int
}

var showFileUpload = true
var host = "https://dockervm:8080"

// NewClientTLSConfig creates a per-client tls config
func NewClientTLSConfig(client *ClientIdentity) (*tls.Config, error) {

	// Create the trust
	trustBytes, err := ioutil.ReadFile(client.TrustPem)
	if err != nil {
		log.Printf("Unable to read %s: %v", client.TrustPem, err)
		return nil, err
	}
	trustCertPool := x509.NewCertPool()
	if trustCertPool.AppendCertsFromPEM(trustBytes) == false {
		log.Printf("Error parsing cert: %v", err)
		return nil, err
	}

	//Create certkeypair
	cert, err := tls.LoadX509KeyPair(client.CertPem, client.KeyPem)
	if err != nil {
		log.Printf("Error parsing cert: %v", err)
		return nil, err
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
		// Ensure that we only use our "CA" to validate certificates
		ClientCAs: trustCertPool,
		// PFS because we can but this will reject client with RSA certificates
		// CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		// Force it server side
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS10,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}

func getClientIdentity(i int, name string) (*ClientIdentity, error) {
	ci := &ClientIdentity{
		TrustPem: os.ExpandEnv("$GOPATH/src/decipher.com/oduploader/defaultcerts/clients/client.trust.pem"),
		CertPem:  os.ExpandEnv("$GOPATH/src/decipher.com/oduploader/defaultcerts/clients/" + name + ".cert.pem"),
		KeyPem:   os.ExpandEnv("$GOPATH/src/decipher.com/oduploader/defaultcerts/clients/" + name + ".key.pem"),
	}
	cfg, err := NewClientTLSConfig(ci)
	if err != nil {
		log.Printf("Cannot get identity: %v", err)
		return nil, err
	}
	ci.Config = cfg
	ci.Name = name

	//Keep this huge directory out of $GOPATH
	if os.ExpandEnv("$AUTOPILOT_HOME") == "" {
		os.Setenv("$AUTOPILOT_HOME", os.ExpandEnv("$HOME/autopilot"))
		os.Mkdir("~/autopilot", 0700)
	}
	ci.UploadCache = os.ExpandEnv("$HOME/autopilot/uploadCache" + name)
	ci.DownloadCache = os.ExpandEnv("$HOME/autopilot/downloadCache" + name)
	ci.Index = i
	_, err = os.Stat(ci.UploadCache)
	if os.IsNotExist(err) {
		err = os.Mkdir(ci.UploadCache, 0700)
		if err != nil {
			log.Printf("Unable to make an upload cache for %s:%v", ci.UploadCache, err)
			return nil, err
		}
	}
	_, err = os.Stat(ci.DownloadCache)
	if os.IsNotExist(err) {
		err = os.Mkdir(ci.DownloadCache, 0700)
		if err != nil {
			log.Printf("Unable to make a download cache for %s:%v", name, err)
			return nil, err
		}
	}
	return ci, nil
}

var clients []*ClientIdentity

func populateClients(population int) {
	clients = make([]*ClientIdentity, population)
	for i := 0; i < len(clients); i++ {
		client, err := getClientIdentity(i, "test_"+strconv.Itoa(i))
		clients[i] = client
		if err != nil {
			log.Printf("Could not create client %d: %v", i, err)
		} else {
			//log.Printf("Creating client %d", i)
		}
	}
}

func doSleep(i int) {
	zzz := rand.Intn(sleepTime)
	time.Sleep(time.Duration(zzz) * time.Second)
	//log.Printf("%d sleeps for %ds", i, zzz)
}

func getRandomClassification() string {
	r := rand.Intn(4)
	classes := []string{"U", "C", "S", "T"}
	return classes[r]
}

func generateUploadRequest(name string, fqName string, url string) (*http.Request, error) {
	f, err := os.Open(fqName)
	defer f.Close()
	if err != nil {
		log.Printf("Unable to open %s: %v", fqName, err)
		return nil, err
	}
	//Create a multipart mime request
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.WriteField("type", "File")
	w.WriteField("classification", getRandomClassification())
	fw, err := w.CreateFormFile("filestream", name)
	if err != nil {
		log.Printf("unable to create form file from %s:%v", fqName, err)
		return nil, err
	}
	if _, err = io.Copy(fw, f); err != nil {
		log.Printf("Could not copy file:%v", err)
		return nil, err
	}
	w.Close()

	req, err := http.NewRequest(
		"POST",
		url,
		&b,
	)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Could not generate request: %v", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	return req, err
}

func doUpload(i int) *protocol.ObjectLink {
	var link protocol.ObjectLink

	//log.Printf("%d upload out of %s", i, clients[i].UploadCache)
	//Pick a random file
	listing, err := ioutil.ReadDir(clients[i].UploadCache)
	if err != nil {
		log.Printf("Unable to list upload directory %s", clients[i].UploadCache)
		return nil
	}
	if len(listing) == 0 {
		log.Printf("Nothing to upload...")
		return nil
	}
	//Grab a random item out of the listing (in memory... beware of huge dirs!)
	r := rand.Intn(len(listing))
	filePicked := listing[r]

	if filePicked.IsDir() == false {
		filePickedName := filePicked.Name()
		fqName := clients[i].UploadCache + "/" + filePickedName
		req, err := generateUploadRequest(
			filePickedName,
			fqName,
			host+"/service/metadataconnector/1.0/object",
		)
		if err != nil {
			log.Printf("Could not generate request:%v", err)
			return nil
		}

		transport := &http.Transport{TLSClientConfig: clients[i].Config}
		client := &http.Client{Transport: transport}

		reqBytes, err := httputil.DumpRequestOut(req, showFileUpload)
		log.Printf("%v\n%s", err, string(reqBytes))

		res, err := client.Do(req)
		if err != nil {
			log.Printf("Error doing client request:%v", err)
			return nil
		}
		// Check the response
		if res.StatusCode != http.StatusOK {
			log.Printf("bad status: %s", res.Status)
			return nil
		}
		log.Printf("%s uploaded %s", clients[i].Name, fqName)

		resBytes, err := httputil.DumpResponse(res, true)
		log.Printf("%v\n%s", err, string(resBytes))

		decoder := json.NewDecoder(res.Body)
		err = decoder.Decode(&link)
	}
	return &link
}

//Get candidate objects that we own, to perform operations on them
func getObjectLinkResponse(i int, olResponse *protocol.ObjectLinkResponse) (err error) {
	log.Printf("first, we must get a listing of objects to choose from")
	req, err := http.NewRequest(
		"GET",
		host+"/service/metadataconnector/1.0/objects",
		nil,
	)
	if err != nil {
		log.Printf("unable to do request for object listing:%v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if showFileUpload {
		reqBytes, err := httputil.DumpRequestOut(req, true)
		log.Printf("%v\n%s", err, string(reqBytes))
	}

	transport := &http.Transport{TLSClientConfig: clients[i].Config}
	client := &http.Client{Transport: transport}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error doing listing request:%v", err)
		return err
	}
	if showFileUpload {
		resBytes, err := httputil.DumpResponse(res, true)
		log.Printf("%v\n%s", err, string(resBytes))
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		return nil
	}
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(olResponse)
	if err != nil {
		log.Printf("Unable to decode response:%v", err)
		return err
	}
	return nil
}

func doDownloadLink(i int, link *protocol.ObjectLink) {
	dlReq, err := http.NewRequest(
		"GET",
		host+link.URL+"/stream",
		nil,
	)
	if err != nil {
		log.Printf("Unable to generate request:%v", err)
		return
	}

	if showFileUpload {
		reqBytes, err := httputil.DumpRequestOut(dlReq, showFileUpload)
		log.Printf("%v\n%s", err, string(reqBytes))
	}

	//Now download the stream into a file
	transport2 := &http.Transport{TLSClientConfig: clients[i].Config}
	client2 := &http.Client{Transport: transport2}

	dlRes, err := client2.Do(dlReq)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		return
	}

	if showFileUpload {
		resBytes, err := httputil.DumpResponse(dlRes, true)
		log.Printf("%v\n%s", err, string(resBytes))
	}

	drainFileName := clients[i].DownloadCache + "/" + link.Name
	drainFile, err := os.Create(drainFileName)
	if err != nil {
		log.Printf("Cant open %s", drainFileName)
		return
	}
	defer drainFile.Close()
	io.Copy(drainFile, dlRes.Body)
	log.Printf("downloaded %s", link.Name)
}

func doDownload(i int) *protocol.ObjectLink {
	//Get the links to download
	var link *protocol.ObjectLink
	var olResponse protocol.ObjectLinkResponse
	err := getObjectLinkResponse(i, &olResponse)
	if err != nil {
		log.Printf("Unable to do download:%v", err)
		return link
	}

	//Grab a random item (if any exist) and download it
	if len(olResponse.Objects) > 0 {

		//Download the listing from which to grab a random item
		randomIndex := rand.Intn(len(olResponse.Objects))
		link = &olResponse.Objects[randomIndex]

		doDownloadLink(i, link)
	}
	return link
}

func doUpdateLink(i int, link *protocol.ObjectLink) {
	fqName := clients[i].UploadCache + "/" + link.Name
	req, err := generateUploadRequest(
		link.Name,
		fqName,
		host+link.URL+"/stream",
	)
	if err != nil {
		log.Printf("Could not generate request:%v", err)
		return
	}

	reqBytes, err := httputil.DumpRequestOut(req, showFileUpload)
	log.Printf("%v\n%s", err, string(reqBytes))

	//Now download the stream into a file
	transport2 := &http.Transport{TLSClientConfig: clients[i].Config}
	client2 := &http.Client{Transport: transport2}

	res, err := client2.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		return
	}
	resBytes, err := httputil.DumpResponse(res, true)
	log.Printf("%v\n%s", err, string(resBytes))

	drainFileName := clients[i].DownloadCache + "/" + link.Name
	drainFile, err := os.Create(drainFileName)
	if err != nil {
		log.Printf("Cant open %s", drainFileName)
		return
	}
	defer drainFile.Close()
	io.Copy(drainFile, res.Body)
}

func doUpdate(i int) {
	//Get the links to download
	var olResponse protocol.ObjectLinkResponse
	err := getObjectLinkResponse(i, &olResponse)
	if err != nil {
		log.Printf("Unable to do download:%v", err)
		return
	}

	//Grab a random item (if any exist) and download it
	if len(olResponse.Objects) > 0 {

		//Download the listing from which to grab a random item
		randomIndex := rand.Intn(len(olResponse.Objects))
		link := &olResponse.Objects[randomIndex]

		doUpdateLink(i, link)
	}
}

func doRandomAction(i int) bool {
	doSleep(i)
	r := rand.Intn(100)
	switch {
	case r > 70:
		doUpload(i)
	case r > 40:
		doDownload(i)
	case r > 20:
		doUpdate(i)
	case r > 10:
		return false
	}
	return true
}

func doClient(i int, clientExited chan int) {
	//log.Printf("running client %d", i)
	for {
		if doRandomAction(i) == false {
			break
		}
	}
	clientExited <- i
}

func dnFromInt(n int) string {
	if n == 0 {
		n = 10
	}
	return fmt.Sprintf(
		"CN=test tester%02d,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US", n,
	)
}

// Have user i grant link to j
func doGrant(i int, link *protocol.ObjectLink, j int) {
	//	dnFrom := dnFromInt(i)
	dnTo := dnFromInt(j)

	jsonObj := protocol.ObjectGrant{
		Grantee: dnTo,
		Create:  true,
		Read:    true,
		Update:  true,
		Delete:  true,
	}

	jsonStr, err := json.Marshal(jsonObj)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
	}

	req, err := http.NewRequest(
		"POST",
		host+link.URL+"/grant",
		bytes.NewBuffer(jsonStr),
	)
	if err != nil {
		log.Printf("Unable to generate request:%v", err)
		return
	}

	if showFileUpload {
		reqBytes, err := httputil.DumpRequestOut(req, showFileUpload)
		log.Printf("%v\n%s", err, string(reqBytes))
	}

	//Now download the stream into a file
	transport2 := &http.Transport{TLSClientConfig: clients[i].Config}
	client2 := &http.Client{Transport: transport2}

	res, err := client2.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		return
	}

	if showFileUpload {
		resBytes, err := httputil.DumpResponse(res, true)
		log.Printf("%v\n%s", err, string(resBytes))
	}

}

var population = 10
var perPopulation = 20
var sleepTime = 120
var isQuickTest = true

func generatePopulation() {
	//We have 10 test certs (note the test_0 is known as tester10)
	population := 10
	populateClients(population)
}

func bigTest() {
	clientExited := make(chan int)
	N := 20
	//Launch all clients Nx
	for n := 0; n < N; n++ {
		for i := 0; i < population; i++ {
			go doClient(i, clientExited)
		}
	}

	//Wait for them to all exit
	stillRunning := population * N
	for {
		log.Printf("Waiting on %d more", stillRunning)
		i := <-clientExited
		log.Printf("Client %d exited", i)
		stillRunning--
		if stillRunning <= 0 {
			break
		}
	}
}

var userID = 0
var userID2 = 1

/*
  Do a simple sequence to see that it actually works.
	Capture the output so that we can see the raw http.
*/
func quickTest() {

	//Upload some random file
	link := doUpload(userID)

	//Have userID2 upload a file so that he exists in the database
	doUpload(userID2)

	log.Printf("")
	//var listing server.ObjectLinkResponse
	//getObjectLinkResponse(userID, &listing)
	//link = &listing.Objects[0]

	if link != nil {
		//Download THAT file
		doDownloadLink(userID, link)
		log.Printf("")
		//Update THAT file
		doUpdateLink(userID, link)
		log.Printf("")
		//Try to re-download it
		doDownloadLink(userID, link)
		log.Printf("")

		doGrant(userID, link, userID2)
	} else {
		log.Printf("We uploaded a file but got no link back!")
	}
}

func main() {
	flag.StringVar(&host, "url", "https://dockervm:8080", "The URL at which to direct uploads/downloads")
	flag.IntVar(&perPopulation, "perPopulation", 20, "number of uploads per user")
	flag.IntVar(&sleepTime, "sleepTime", 120, "number of seconds to sleep when we decide to sleep")
	flag.BoolVar(&isQuickTest, "quickTest", true, "just run a simple up/down test")
	flag.Parse()

	generatePopulation()

	if false {
		bigTest()
	} else {
		quickTest()
	}
}
