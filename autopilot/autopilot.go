package autopilot

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
)

// AutopilotArgs are the things passed in to command line
type AutopilotArgs struct {
	Host          string
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



func GetRandomClassification() string {
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
	um := protocol.CreateObjectRequest{
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

func DumpAutopilotParams() {
	fmt.Printf("# Global Parameters\n")
	fmt.Printf("```json\n")
	ap := AutopilotArgs{
		Host:          host,
	}
	data, err := json.MarshalIndent(ap, "", "  ")
	if err != nil {
		log.Printf("Unable to marshal global args")
	}
	fmt.Printf("%s\n", data)
	fmt.Printf("```\n")
}

//Dump the transport with a label. TODO: with message
func DumpTransport(i int) {
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

func DoUpload(i int, async bool, msg string) (link *protocol.Object, res *http.Response, err error) {
    var listing []os.FileInfo
    var req *http.Request
    
	//log.Printf("%d upload out of %s", i, clients[i].UploadCache)
	//Pick a random file
	listing, err = ioutil.ReadDir(clients[i].UploadCache)
	if err != nil {
		log.Printf("Unable to list upload directory %s", clients[i].UploadCache)
		return
	}
	//Grab a random item out of the listing (in memory... beware of huge dirs!)
	if len(listing) > 0 {
		r := rand.Intn(len(listing))
		filePicked := listing[r]

		if filePicked.IsDir() == false {
			filePickedName := filePicked.Name()
			fqName := clients[i].UploadCache + "/" + filePickedName
			req, err = generateUploadRequest(
				filePickedName,
				fqName,
				host+"/service/metadataconnector/1.0/object",
				async,
			)
			if err != nil {
				log.Printf("Could not generate request:%v", err)
				return
			}

			transport := &http.Transport{TLSClientConfig: clients[i].Config}
			client := &http.Client{Transport: transport}

			dumpRequest(req, "Upload", msg)

			res, err = client.Do(req)
			if err != nil {
				log.Printf("Error doing client request:%v", err)
				return
			}
			// Check the response
			if res.StatusCode != http.StatusOK {
				log.Printf("bad status: %s", res.Status)
				return
			}

			dumpResponse(res, "json of the uploaded object is returned, as soon as EC2 has it. use it to perform further actions on the file.")

			decoder := json.NewDecoder(res.Body)
			err = decoder.Decode(&link)
		}
	}
	return
}

//Get candidate objects that we own, to perform operations on them
func GetLinks(i int, olResponse *protocol.ObjectResultset, msg string) (res *http.Response, err error) {
    var req *http.Request
	req, err = http.NewRequest(
		"GET",
		host+rootURL+"/objects",
		nil,
	)
	if err != nil {
		log.Printf("unable to do request for object listing:%v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	if showFileUpload {
		dumpRequest(req, "Listing", msg)
	}

	transport := &http.Transport{TLSClientConfig: clients[i].Config}
	client := &http.Client{Transport: transport}
	res, err = client.Do(req)
	if err != nil {
		log.Printf("Error doing listing request:%v", err)
		return
	}
	if showFileUpload {
		dumpResponse(res, "Got a listing of available files")
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		return
	}
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(olResponse)
	if err != nil {
		log.Printf("Unable to decode response:%v", err)
		return
	}
    return
}

func DownloadLinkByName(
    name string,
    i int, 
    msg string,
) (link *protocol.Object, res *http.Response, err error) {
	var olResponse protocol.ObjectResultset
	res, err = GetLinks(i, &olResponse, msg)
	if err != nil {
		log.Printf("Unable to do download:%v", err)
		return
	}

    for k,v := range olResponse.Objects {
        if name == v.Name {
            link = &olResponse.Objects[k]
    		res, err = DoDownloadLink(i, link, msg)
        }
    }
    return
}

func DoDownloadLink(i int, link *protocol.Object, msg string) (res *http.Response, err error) {
    var req *http.Request
	req, err = http.NewRequest(
		"GET",
		host+rootURL+"/object/"+link.ID+"/stream",
		nil,
	)
	if err != nil {
		log.Printf("Unable to generate request:%v", err)
		return
	}

	if showFileUpload {
		dumpRequest(req, "GetObject", msg)
	}

	//Now download the stream into a file
	transport2 := &http.Transport{TLSClientConfig: clients[i].Config}
	client2 := &http.Client{Transport: transport2}

	res, err = client2.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		return
	}

	if showFileUpload {
		dumpResponse(res, "Got the raw file.")
	}

	drainFileName := clients[i].DownloadCache + "/" + link.Name
	drainFile, err := os.Create(drainFileName)
	if err != nil {
		log.Printf("Cant open %s", drainFileName)
		return
	}
	defer drainFile.Close()
	io.Copy(drainFile, res.Body)
    return
}

func DoDownload(i int, msg string) (link *protocol.Object, res *http.Response, err error) {
	//Get the links to download
	var olResponse protocol.ObjectResultset
	res, err = GetLinks(i, &olResponse, msg)
	if err != nil {
		log.Printf("Unable to do download:%v", err)
		return
	}

	//Grab a random item (if any exist) and download it
	if len(olResponse.Objects) > 0 {

		//Download the listing from which to grab a random item
		randomIndex := rand.Intn(len(olResponse.Objects))
		link = &olResponse.Objects[randomIndex]

		res, err = DoDownloadLink(i, link, msg)
	}
    return
}

func DoUpdateLink(i int, link *protocol.Object, msg, toAppend string) (res *http.Response, err error) {
	//Assuming that the file has been downloaded.  Modify it by appending data
	fqName := clients[i].DownloadCache + "/" + link.Name
	//Modify the file a little
	f, err := os.OpenFile(fqName,os.O_RDWR|os.O_APPEND, os.ModeAppend)
	if err != nil {
		log.Printf("Could not append to file")
	}
	n, err := f.WriteString(toAppend)
    if err != nil {
        log.Printf("%d %v", n, err)
    }
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

	res, err = client2.Do(req)
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
    return
}

func DoUpdate(i int, msg, toAppend string) (res *http.Response, err error) {
	//Get the links to download
	var olResponse protocol.ObjectResultset
	res, err = GetLinks(i, &olResponse, msg)
	if err != nil {
		log.Printf("Unable to do download:%v", err)
		return
	}

	//Grab a random item (if any exist) and download it
	if len(olResponse.Objects) > 0 {

		//Download the listing from which to grab a random item
		randomIndex := rand.Intn(len(olResponse.Objects))
		link := &olResponse.Objects[randomIndex]

		res, err = DoUpdateLink(i, link, msg, toAppend)
	}
    return
}



func dnFromInt(n int) string {
	if n == 0 {
		n = 10
	}
	return fmt.Sprintf(
		"CN=test tester%02d,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US", n,
	)
}

func FindShares(i int, msg string) (links *protocol.ObjectResultset, res *http.Response, err error) {
    var req *http.Request
	req, err = http.NewRequest(
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

	res, err = client2.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		return
	}
	dumpResponse(res, "ListShares")
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&links)
    return
}

func DoUserList(i int, msg string) (users []*protocol.User, res *http.Response, err error) {
    var req *http.Request
	req, err = http.NewRequest(
		"GET",
		host+rootURL+"/users",
		nil,
	)
    
	dumpRequest(req, "User Listing", "Get the users.")
    
	transport2 := &http.Transport{TLSClientConfig: clients[i].Config}
	client2 := &http.Client{Transport: transport2}
	res, err = client2.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		return
	}
	dumpResponse(res, "All users who have visited the site gave us their identity")
    decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&users)
    return
}

// Have user i grant link to j
func DoShare(i int, link *protocol.Object, j int, msg string) (*http.Response, error) {
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
        return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		host+rootURL+"/object/"+link.ID+"/share",
		bytes.NewBuffer(jsonStr),
	)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Unable to generate request:%v", err)
		return nil, err
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
		return nil, err
	}

	if showFileUpload {
		dumpResponse(res, "Share")
	}
    return res, nil
}

func generatePopulation() {
	//We have 10 test certs (note the test_0 is known as tester10)
	population := 10
	populateClients(population)
}


var Population = 10
var isQuickTest = true
var showFileUpload = true
var host = "https://dockervm:8080"
var rootURL = "/service/metadataconnector/1.0"
var autopilotRoot = "$GOPATH/src/decipher.com/oduploader/autopilot/cache"

func Init() {
	flag.StringVar(&host, "url", host, "The URL at which to direct uploads/downloads")
	flag.StringVar(&autopilotRoot, "root", autopilotRoot, "The URL at which to direct uploads/downloads")
	flag.Parse()

	generatePopulation()
}
