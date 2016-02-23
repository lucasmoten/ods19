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
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"time"
)

// AutopilotArgs are the things passed in to command line
type AutopilotArgs struct {
	Host          string
	PerPopulation int
	SleepTime     int
	QuickTest     bool
}

// ClientIdentity is a user that is going to connect to our service
type ClientIdentity struct {
	TrustPem      string
	CertPem       string
	KeyPem        string
	Config        *tls.Config `json:"-"`
	Name          string
	UploadCache   string
	DownloadCache string
	Index         int
}

var showFileUpload = true

//XXX This ASSUMES that you have an /etc/hosts entry for dockervm
var host = "https://dockervm:8080"
var rootURL = "/service/metadataconnector/1.0"
var autopilotRoot = "$GOPATH/src/decipher.com/oduploader/autopilot"

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

	ci.UploadCache = os.ExpandEnv(autopilotRoot + "/uploadCache" + name)
	ci.DownloadCache = os.ExpandEnv(autopilotRoot + "/downloadCache" + name)
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

//-----BEGIN-rewrite-part-of-Go-SDK-----

//multipart form-data writing doesn't let Content-Type get emitted.
//So this code lets us call w.WritePartField(w,fieldname,value),
//while also setting the Content-Type

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func createFormField(w *multipart.Writer, fieldname, contentType string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"`, escapeQuotes(fieldname)))
	h.Set("Content-Type", contentType)
	return w.CreatePart(h)
}

func writePartField(w *multipart.Writer, fieldname, value, contentType string) error {
	p, err := createFormField(w, fieldname, contentType)
	if err != nil {
		return err
	}
	_, err = p.Write([]byte(value))
	return err
}

//-----END-rewrite-part-of-Go-SDK-----

func generateUploadRequest(name string, fqName string, url string, async bool) (*http.Request, error) {
	f, err := os.Open(fqName)
	defer f.Close()
	if err != nil {
		log.Printf("Unable to open %s: %v", fqName, err)
		return nil, err
	}
	//Create a multipart mime request
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	um := protocol.Object{
		TypeName: "File",
		RawAcm:   `{"version":"2.1.0","classif":"S"}`,
	}
	umStr, err := json.MarshalIndent(um, "", "  ")
	if err != nil {
		log.Printf("Cannot marshal object:%v", err)
	}
	//Hmm... had to rewrite part of std Go sdk locally to do this
	writePartField(w, "CreateObjectRequest", string(umStr), "application/json")
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

func dumpAutopilotParams() {
	fmt.Printf("# Global Parameters\n")
	fmt.Printf("```json\n")
	ap := AutopilotArgs{
		Host:          host,
		PerPopulation: perPopulation,
		SleepTime:     sleepTime,
		QuickTest:     isQuickTest,
	}
	data, err := json.MarshalIndent(ap, "", "  ")
	if err != nil {
		log.Printf("Unable to marshal global args")
	}
	fmt.Printf("%s\n", data)
	fmt.Printf("```\n")
}

//Dump the transport with a label. TODO: with message
func dumpTransport(i int) {
	fmt.Printf("# Transport Parameters for User %d\n", i)
	fmt.Printf("```\n")
	fmt.Printf("MinVersion:%v\n", clients[i].Config.MinVersion)
	fmt.Printf("MaxVersion:%v\n", clients[i].Config.MaxVersion)
	fmt.Printf("InsecureSkipVerify:%v\n", clients[i].Config.InsecureSkipVerify)
	fmt.Printf("```\n")
}

//Dump the request with a label.  TODO: with message
func dumpRequest(req *http.Request, title string, msg string) {
	reqBytes, err := httputil.DumpRequestOut(req, showFileUpload)
	fmt.Printf("# %s\n", title+" Request\n")
	fmt.Printf("%s\n", msg+".")
	fmt.Printf("```http\n")
	if err != nil {
		log.Printf("%v", err)
	} else {
		fmt.Printf("%s", string(reqBytes))
	}
	fmt.Printf("\n```\n")
}

//Dump the response with a label.  TODO: wth message.
func dumpResponse(res *http.Response, msg string) {
	reqBytes, err := httputil.DumpResponse(res, showFileUpload)
	fmt.Printf("%s\n", msg)
	fmt.Printf("```http\n")
	if err != nil {
		log.Printf("%v", err)
	} else {
		fmt.Printf("%s", string(reqBytes))
	}
	fmt.Printf("\n```\n")
}

func doUpload(i int, async bool, msg string) *protocol.Object {
	var link protocol.Object

	//log.Printf("%d upload out of %s", i, clients[i].UploadCache)
	//Pick a random file
	listing, err := ioutil.ReadDir(clients[i].UploadCache)
	if err != nil {
		log.Printf("Unable to list upload directory %s", clients[i].UploadCache)
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
			async,
		)
		if err != nil {
			log.Printf("Could not generate request:%v", err)
			return nil
		}

		transport := &http.Transport{TLSClientConfig: clients[i].Config}
		client := &http.Client{Transport: transport}

		dumpRequest(req, "Upload", msg)

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

		dumpResponse(res, "json of the uploaded object is returned, as soon as EC2 has it. use it to perform further actions on the file.")

		decoder := json.NewDecoder(res.Body)
		err = decoder.Decode(&link)
	}
	return &link
}

//Get candidate objects that we own, to perform operations on them
func getObjectLinkResponse(i int, olResponse *protocol.ObjectResultset, msg string) (err error) {
	req, err := http.NewRequest(
		"GET",
		host+rootURL+"/objects",
		nil,
	)
	if err != nil {
		log.Printf("unable to do request for object listing:%v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if showFileUpload {
		dumpRequest(req, "Listing", msg)
	}

	transport := &http.Transport{TLSClientConfig: clients[i].Config}
	client := &http.Client{Transport: transport}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error doing listing request:%v", err)
		return err
	}
	if showFileUpload {
		dumpResponse(res, "Got a listing of available files")
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

func doDownloadLink(i int, link *protocol.Object, msg string) {
	dlReq, err := http.NewRequest(
		"GET",
		host+rootURL+"/object/"+link.ID+"/stream",
		nil,
	)
	if err != nil {
		log.Printf("Unable to generate request:%v", err)
		return
	}

	if showFileUpload {
		dumpRequest(dlReq, "GetObject", msg)
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
		dumpResponse(dlRes, "Got the raw file.")
	}

	drainFileName := clients[i].DownloadCache + "/" + link.Name
	drainFile, err := os.Create(drainFileName)
	if err != nil {
		log.Printf("Cant open %s", drainFileName)
		return
	}
	defer drainFile.Close()
	io.Copy(drainFile, dlRes.Body)
}

func doDownload(i int, msg string) *protocol.Object {
	//Get the links to download
	var link *protocol.Object
	var olResponse protocol.ObjectResultset
	err := getObjectLinkResponse(i, &olResponse, msg)
	if err != nil {
		log.Printf("Unable to do download:%v", err)
		return link
	}

	//Grab a random item (if any exist) and download it
	if len(olResponse.Objects) > 0 {

		//Download the listing from which to grab a random item
		randomIndex := rand.Intn(len(olResponse.Objects))
		link = &olResponse.Objects[randomIndex]

		doDownloadLink(i, link, msg)
	}
	return link
}

func doUpdateLink(i int, link *protocol.Object, msg, toAppend string) {
	//Assuming that the file has been downloaded.  Modify it.
	fqName := clients[i].DownloadCache + "/" + link.Name
	//Modify the file a little
	f, err := os.OpenFile(fqName, os.O_APPEND, os.ModeAppend)
	if err != nil {
		log.Printf("Could not append to file")
	}
	defer f.Close()
	f.WriteString(toAppend)
	f.Close()

	req, err := generateUploadRequest(
		link.Name,
		fqName,
		host+rootURL+"/object/"+link.ID+"/stream",
		false,
	)
	if err != nil {
		log.Printf("Could not generate request:%v", err)
		return
	}

	dumpRequest(req, "UpdateObject", msg)

	//Now download the stream into a file
	transport2 := &http.Transport{TLSClientConfig: clients[i].Config}
	client2 := &http.Client{Transport: transport2}

	res, err := client2.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		return
	}

	dumpResponse(res, "The metadata is different after the update")

	drainFileName := clients[i].DownloadCache + "/" + link.Name
	drainFile, err := os.Create(drainFileName)
	if err != nil {
		log.Printf("Cant open %s", drainFileName)
		return
	}
	defer drainFile.Close()
	io.Copy(drainFile, res.Body)
}

func doUpdate(i int, msg, toAppend string) {
	//Get the links to download
	var olResponse protocol.ObjectResultset
	err := getObjectLinkResponse(i, &olResponse, msg)
	if err != nil {
		log.Printf("Unable to do download:%v", err)
		return
	}

	//Grab a random item (if any exist) and download it
	if len(olResponse.Objects) > 0 {

		//Download the listing from which to grab a random item
		randomIndex := rand.Intn(len(olResponse.Objects))
		link := &olResponse.Objects[randomIndex]

		doUpdateLink(i, link, msg, toAppend)
	}
}

func doRandomAction(i int) bool {
	doSleep(i)
	r := rand.Intn(100)
	switch {
	case r > 70:
		doUpload(i, false, "")
	case r > 40:
		doDownload(i, "")
	case r > 20:
		doUpdate(i, "", "")
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

func findShares(i int, msg string) {
	req, err := http.NewRequest(
		"GET",
		host+rootURL+"/shares",
		nil,
	)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Could not generate request:%v", err)
		return
	}

	dumpRequest(req, "ListShares", msg)

	//Now download the stream into a file
	transport2 := &http.Transport{TLSClientConfig: clients[i].Config}
	client2 := &http.Client{Transport: transport2}

	res, err := client2.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		return
	}
	dumpResponse(res, "ListShares")
}

func doUserList(i int, msg string) {
	req, err := http.NewRequest(
		"GET",
		host+rootURL+"/users",
		nil,
	)
	dumpRequest(req, "User Listing", "Get the users.")
	transport2 := &http.Transport{TLSClientConfig: clients[i].Config}
	client2 := &http.Client{Transport: transport2}
	res, err := client2.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		return
	}
	dumpResponse(res, "All users who have visited the site gave us their identity")
}

// Have user i grant link to j
func doShare(i int, link *protocol.Object, j int, msg string) {
	//	dnFrom := dnFromInt(i)
	dnTo := dnFromInt(j)

	jsonObj := protocol.ObjectGrant{
		Grantee: dnTo,
		Create:  true,
		Read:    true,
		Update:  true,
		Delete:  true,
	}

	jsonStr, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
	}

	req, err := http.NewRequest(
		"POST",
		host+rootURL+"/object/"+link.ID+"/share",
		bytes.NewBuffer(jsonStr),
	)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Unable to generate request:%v", err)
		return
	}

	if showFileUpload {
		dumpRequest(req, "Share", msg)
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
		dumpResponse(res, "Share")
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
	Capture the output in markdown so that we can see the raw http.
*/
func quickTest() {
	//The global parameters
	dumpAutopilotParams()
	//The two users involved
	dumpTransport(userID)
	dumpTransport(userID2)

	//Upload some random file
	link := doUpload(userID, false, "Uploading a file for Alice")

	//Have userID2 upload a file so that he exists in the database
	doUpload(userID2, true, "Uploading a file for Bob")

	doUserList(userID, "See which users exist as a side-effect of visiting the site with their certificates.")

	//var listing server.ObjectLinkResponse
	//getObjectLinkResponse(userID, &listing)
	//link = &listing.Objects[0]

	if link != nil {
		//Download THAT file
		doDownloadLink(userID, link, "Alice downloads the file")
		//Update THAT file - the copy in the download cache
		doUpdateLink(userID, link, "Alice updates the file", "moreAppendedJunk")
		//Try to re-download it
		doDownloadLink(userID, link, "Alice downloads it again")
		//Share with a different user
		doShare(userID, link, userID2, "Alice shares file to Bob")

		//List this users shares
		findShares(userID2, "Look at the shares that Bob has")
	} else {
		log.Printf("We uploaded a file but got no link back!")
	}
}

func doMainDefault() {
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

func main() {
	doMainDefault()
}
