package autopilot

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
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

	cfg "decipher.com/oduploader/config"
	"decipher.com/oduploader/protocol"
	"decipher.com/oduploader/util/testhelpers"
)

//
//WARNING: there should be no: Printf.  It must be Fprintf(ap.Log,...)
//
//

// AutopilotArgs are the things passed in to command line
type AutopilotContext struct {
	Host string
	Url  string
	Root string
	Log  *os.File
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
func (ap AutopilotContext) NewClientTLSConfig(client *ClientIdentity) (*tls.Config, error) {

	// Create the trust
	trustBytes, err := ioutil.ReadFile(client.TrustPem)
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to read %s: %v", client.TrustPem, err)
		return nil, err
	}
	trustCertPool := x509.NewCertPool()
	if trustCertPool.AppendCertsFromPEM(trustBytes) == false {
		fmt.Fprintf(ap.Log, "Error parsing cert: %v", err)
		return nil, err
	}

	//Create certkeypair
	cert, err := tls.LoadX509KeyPair(client.CertPem, client.KeyPem)
	if err != nil {
		fmt.Fprintf(ap.Log, "Error parsing cert: %v", err)
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

func (ap AutopilotContext) getClientIdentity(i int, name string) (*ClientIdentity, error) {
	ci := &ClientIdentity{
		TrustPem: os.ExpandEnv("$GOPATH/src/decipher.com/oduploader/defaultcerts/clients/client.trust.pem"),
		CertPem:  os.ExpandEnv("$GOPATH/src/decipher.com/oduploader/defaultcerts/clients/" + name + ".cert.pem"),
		KeyPem:   os.ExpandEnv("$GOPATH/src/decipher.com/oduploader/defaultcerts/clients/" + name + ".key.pem"),
	}
	cfg, err := ap.NewClientTLSConfig(ci)
	if err != nil {
		fmt.Fprintf(ap.Log, "Cannot get identity: %v", err)
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
			fmt.Fprintf(ap.Log, "Unable to make an upload cache for %s:%v", ci.UploadCache, err)
			return nil, err
		}
	}
	_, err = os.Stat(ci.DownloadCache)
	if os.IsNotExist(err) {
		err = os.Mkdir(ci.DownloadCache, 0700)
		if err != nil {
			fmt.Fprintf(ap.Log, "Unable to make a download cache for %s:%v", name, err)
			return nil, err
		}
	}
	return ci, nil
}

var clients []*ClientIdentity

func (ap AutopilotContext) populateClients(population int) {
	clients = make([]*ClientIdentity, population)
	for i := 0; i < len(clients); i++ {
		client, err := ap.getClientIdentity(i, "test_"+strconv.Itoa(i))
		clients[i] = client
		if err != nil {
			fmt.Fprintf(ap.Log, "Could not create client %d: %v", i, err)
		} else {
			//log.Printf("Creating client %d", i)
		}
	}
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

func (ap AutopilotContext) generateUploadRequest(name string, fqName string, url string, async bool) (*http.Request, error) {
	f, err := os.Open(fqName)
	defer f.Close()
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to open %s: %v", fqName, err)
		return nil, err
	}
	//Create a multipart mime request
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	um := protocol.CreateObjectRequest{
		TypeName: "File",
		RawAcm:   testhelpers.ValidACMUnclassified,
	}
	umStr, err := json.MarshalIndent(um, "", "  ")
	if err != nil {
		fmt.Fprintf(ap.Log, "Cannot marshal object:%v", err)
	}
	//Hmm... had to rewrite part of std Go sdk locally to do this
	writePartField(w, "ObjectMetadata", string(umStr), "application/json")
	fw, err := w.CreateFormFile("filestream", name)
	if err != nil {
		fmt.Fprintf(ap.Log, "unable to create form file from %s:%v", fqName, err)
		return nil, err
	}
	if _, err = io.Copy(fw, f); err != nil {
		fmt.Fprintf(ap.Log, "Could not copy file:%v", err)
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
		fmt.Fprintf(ap.Log, "Could not generate request: %v", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	return req, err
}

func (ap AutopilotContext) generateUpdateRequest(changeToken, name string, fqName string, url string, async bool) (*http.Request, error) {
	f, err := os.Open(fqName)
	defer f.Close()
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to open %s: %v", fqName, err)
		return nil, err
	}
	//Create a multipart mime request
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	um := protocol.UpdateStreamRequest{
		ChangeToken: changeToken,
		RawAcm:      testhelpers.ValidACMUnclassified,
	}
	umStr, err := json.MarshalIndent(um, "", "  ")
	if err != nil {
		fmt.Fprintf(ap.Log, "Cannot marshal object:%v", err)
	}
	//Hmm... had to rewrite part of std Go sdk locally to do this
	writePartField(w, "ObjectMetadata", string(umStr), "application/json")
	fw, err := w.CreateFormFile("filestream", name)
	if err != nil {
		fmt.Fprintf(ap.Log, "unable to create form file from %s:%v", fqName, err)
		return nil, err
	}
	if _, err = io.Copy(fw, f); err != nil {
		fmt.Fprintf(ap.Log, "Could not copy file:%v", err)
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
		fmt.Fprintf(ap.Log, "Could not generate request: %v", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	return req, err
}

// Closing the context should flush and write out the file for this trace
func (ap AutopilotContext) Close() {
	ap.Log.Close()
}

//Dump the transport with a label. TODO: with message
func (ap AutopilotContext) DumpTransport(i int) {
	fmt.Fprintf(ap.Log, "# Transport Parameters for User %d\n", i)
	fmt.Fprintf(ap.Log, "```\n")
	fmt.Fprintf(ap.Log, "MinVersion:%v\n", clients[i].Config.MinVersion)
	fmt.Fprintf(ap.Log, "MaxVersion:%v\n", clients[i].Config.MaxVersion)
	fmt.Fprintf(ap.Log, "InsecureSkipVerify:%v\n", clients[i].Config.InsecureSkipVerify)
	fmt.Fprintf(ap.Log, "```\n")
	ap.Log.Sync()
}

//Dump the request with a label.  TODO: with message
func (ap AutopilotContext) dumpRequest(req *http.Request, title string, msg string) {
	reqBytes, err := httputil.DumpRequestOut(req, showFileUpload)
	fmt.Fprintf(ap.Log, "# %s\n", title+" Request\n")
	fmt.Fprintf(ap.Log, "%s\n", msg+".")
	fmt.Fprintf(ap.Log, "```http\n")
	if err != nil {
		fmt.Fprintf(ap.Log, "%v", err)
	} else {
		fmt.Fprintf(ap.Log, "%s", string(reqBytes))
	}
	fmt.Fprintf(ap.Log, "\n```\n")
	ap.Log.Sync()
}

//Dump the response with a label.  TODO: wth message.
func (ap AutopilotContext) dumpResponse(res *http.Response, msg string) {
	reqBytes, err := httputil.DumpResponse(res, showFileUpload)
	fmt.Fprintf(ap.Log, "%s\n", msg)
	fmt.Fprintf(ap.Log, "```http\n")
	if err != nil {
		fmt.Fprintf(ap.Log, "%v", err)
	} else {
		fmt.Fprintf(ap.Log, "%s", string(reqBytes))
	}
	fmt.Fprintf(ap.Log, "\n```\n")
}

func (ap AutopilotContext) DoUpload(i int, async bool, msg string) (link *protocol.Object, res *http.Response, err error) {
	var listing []os.FileInfo
	var req *http.Request

	//log.Printf("%d upload out of %s", i, clients[i].UploadCache)
	//Pick a random file
	listing, err = ioutil.ReadDir(clients[i].UploadCache)
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to list upload directory %s", clients[i].UploadCache)
		return
	}
	//Grab a random item out of the listing (in memory... beware of huge dirs!)
	if len(listing) > 0 {
		r := rand.Intn(len(listing))
		filePicked := listing[r]

		if filePicked.IsDir() == false {
			filePickedName := filePicked.Name()
			fqName := clients[i].UploadCache + "/" + filePickedName
			req, err = ap.generateUploadRequest(
				filePickedName,
				fqName,
				host+cfg.RootURL+"/object",
				async,
			)
			if err != nil {
				fmt.Fprintf(ap.Log, "Could not generate request:%v", err)
				return
			}

			transport := &http.Transport{TLSClientConfig: clients[i].Config}
			client := &http.Client{Transport: transport}

			ap.dumpRequest(req, "Upload", msg)

			res, err = client.Do(req)
			if err != nil {
				fmt.Fprintf(ap.Log, "Error doing client request:%v", err)
				return
			}
			// Check the response
			if res.StatusCode != http.StatusOK {
				fmt.Fprintf(ap.Log, "bad status: %s", res.Status)
				return
			}

			ap.dumpResponse(res, "json of the uploaded object is returned, as soon as EC2 has it. use it to perform further actions on the file.")

			decoder := json.NewDecoder(res.Body)
			err = decoder.Decode(&link)
		}
	}
	return
}

//Get candidate objects that we own, to perform operations on them
func (ap AutopilotContext) GetLinks(i int, olResponse *protocol.ObjectResultset, msg string) (res *http.Response, err error) {
	var req *http.Request
	req, err = http.NewRequest(
		"GET",
		host+cfg.RootURL+"/objects",
		nil,
	)
	if err != nil {
		fmt.Fprintf(ap.Log, "unable to do request for object listing:%v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	if showFileUpload {
		ap.dumpRequest(req, "Listing", msg)
	}

	transport := &http.Transport{TLSClientConfig: clients[i].Config}
	client := &http.Client{Transport: transport}
	res, err = client.Do(req)
	if err != nil {
		fmt.Fprintf(ap.Log, "Error doing listing request:%v", err)
		return
	}
	if showFileUpload {
		ap.dumpResponse(res, "Got a listing of available files")
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		fmt.Fprintf(ap.Log, "bad status: %s", res.Status)
		return
	}
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(olResponse)
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to decode response:%v", err)
		return
	}
	return
}

func (ap AutopilotContext) DownloadLinkByName(
	name string,
	i int,
	msg string,
) (link *protocol.Object, res *http.Response, err error) {
	var olResponse protocol.ObjectResultset
	res, err = ap.GetLinks(i, &olResponse, msg)
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to do download:%v", err)
		return
	}

	for k, v := range olResponse.Objects {
		if name == v.Name {
			link = &olResponse.Objects[k]
			res, err = ap.DoDownloadLink(i, link, msg)
		}
	}
	return
}

func (ap AutopilotContext) DoDownloadLink(i int, link *protocol.Object, msg string) (res *http.Response, err error) {
	var req *http.Request
	req, err = http.NewRequest(
		"GET",
		host+cfg.RootURL+"/object/"+link.ID+"/stream",
		nil,
	)
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to generate request:%v", err)
		return
	}

	if showFileUpload {
		ap.dumpRequest(req, "GetObject", msg)
	}

	//Now download the stream into a file
	transport2 := &http.Transport{TLSClientConfig: clients[i].Config}
	client2 := &http.Client{Transport: transport2}

	res, err = client2.Do(req)
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to do request:%v", err)
		return
	}

	if showFileUpload {
		ap.dumpResponse(res, "Got the raw file.")
	}

	if res == nil {
		fmt.Fprintf(ap.Log, "Null response:%v", err)
		return
	}

	drainFileName := clients[i].DownloadCache + "/" + link.Name
	drainFile, err := os.Create(drainFileName)
	if err != nil {
		fmt.Fprintf(ap.Log, "Cant open %s", drainFileName)
		return
	}
	defer drainFile.Close()
	if res.Body != nil {
		io.Copy(drainFile, res.Body)
	}
	return
}

func (ap AutopilotContext) DoDownload(i int, msg string) (link *protocol.Object, res *http.Response, err error) {
	//Get the links to download
	var olResponse protocol.ObjectResultset
	res, err = ap.GetLinks(i, &olResponse, msg)
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to do download:%v", err)
		return
	}

	//Grab a random item (if any exist) and download it
	if len(olResponse.Objects) > 0 {

		//Download the listing from which to grab a random item
		randomIndex := rand.Intn(len(olResponse.Objects))
		link = &olResponse.Objects[randomIndex]

		res, err = ap.DoDownloadLink(i, link, msg)
	}
	return
}

func (ap AutopilotContext) DoUpdateLink(i int, link *protocol.Object, msg, toAppend string) (res *http.Response, err error) {
	//Assuming that the file has been downloaded.  Modify it by appending data
	fqName := clients[i].DownloadCache + "/" + link.Name
	//Modify the file a little
	f, err := os.OpenFile(fqName, os.O_RDWR|os.O_APPEND, os.ModeAppend)
	if err != nil {
		fmt.Fprintf(ap.Log, "Could not append to file")
	}
	n, err := f.WriteString(toAppend)
	if err != nil {
		fmt.Fprintf(ap.Log, "%d %v", n, err)
	}
	f.Close()

	req, err := ap.generateUpdateRequest(
		link.ChangeToken,
		link.Name,
		fqName,
		host+cfg.RootURL+"/object/"+link.ID+"/stream",
		false,
	)
	if err != nil {
		fmt.Fprintf(ap.Log, "Could not generate request:%v", err)
		return
	}

	ap.dumpRequest(req, "UpdateObject", msg)

	//Now download the stream into a file
	transport2 := &http.Transport{TLSClientConfig: clients[i].Config}
	client2 := &http.Client{Transport: transport2}

	res, err = client2.Do(req)
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to do request:%v", err)
		return
	}

	ap.dumpResponse(res, "The metadata is different after the update")

	drainFileName := clients[i].DownloadCache + "/" + link.Name
	drainFile, err := os.Create(drainFileName)
	if err != nil {
		fmt.Fprintf(ap.Log, "Cant open %s", drainFileName)
		return
	}
	defer drainFile.Close()
	io.Copy(drainFile, res.Body)
	return
}

func (ap AutopilotContext) DoUpdate(i int, msg, toAppend string) (res *http.Response, err error) {
	//Get the links to download
	var olResponse protocol.ObjectResultset
	res, err = ap.GetLinks(i, &olResponse, msg)
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to do download:%v", err)
		return
	}

	//Grab a random item (if any exist) and download it
	if len(olResponse.Objects) > 0 {

		//Download the listing from which to grab a random item
		randomIndex := rand.Intn(len(olResponse.Objects))
		link := &olResponse.Objects[randomIndex]

		res, err = ap.DoUpdateLink(i, link, msg, toAppend)
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

func (ap AutopilotContext) FindShares(i int, msg string) (links *protocol.ObjectResultset, res *http.Response, err error) {
	var req *http.Request
	req, err = http.NewRequest(
		"GET",
		host+cfg.RootURL+"/shares",
		nil,
	)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Fprintf(ap.Log, "Could not generate request:%v", err)
		return
	}

	ap.dumpRequest(req, "ListShares", msg)

	//Now download the stream into a file
	transport2 := &http.Transport{TLSClientConfig: clients[i].Config}
	client2 := &http.Client{Transport: transport2}

	res, err = client2.Do(req)
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to do request:%v", err)
		return
	}
	ap.dumpResponse(res, "ListShares")
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&links)
	return
}

func (ap AutopilotContext) DoUserList(i int, msg string) (users []*protocol.User, res *http.Response, err error) {
	var req *http.Request
	req, err = http.NewRequest(
		"GET",
		host+cfg.RootURL+"/users",
		nil,
	)

	ap.dumpRequest(req, "User Listing", "Get the users.")

	transport2 := &http.Transport{TLSClientConfig: clients[i].Config}
	client2 := &http.Client{Transport: transport2}
	res, err = client2.Do(req)
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to do request:%v", err)
		return
	}
	ap.dumpResponse(res, "All users who have visited the site gave us their identity")
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&users)
	return
}

// Have user i grant link to j
func (ap AutopilotContext) DoShare(i int, link *protocol.Object, j int, msg string) (*http.Response, error) {
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
		host+cfg.RootURL+"/object/"+link.ID+"/share",
		bytes.NewBuffer(jsonStr),
	)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("Unable to generate request:%v", err)
		return nil, err
	}

	if showFileUpload {
		ap.dumpRequest(req, "Share", msg)
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
		ap.dumpResponse(res, "Share")
	}
	return res, nil
}

func (ap AutopilotContext) generatePopulation() {
	//We have 10 test certs (note the test_0 is known as tester10)
	population := 10
	ap.populateClients(population)
}

// Create a new context, in which all output is logged for this trace by file name
func NewAutopilotContext(logHandle *os.File) (ap *AutopilotContext, err error) {
	ap = &AutopilotContext{
		Host: host,
		Url:  cfg.RootURL,
		Root: autopilotRoot,
		Log:  logHandle,
	}
	ap.generatePopulation()
	fmt.Fprintf(ap.Log, "# Global Parameters\n")
	fmt.Fprintf(ap.Log, "```json\n")
	var data []byte
	data, err = json.MarshalIndent(ap, "", "  ")
	if err != nil {
		fmt.Fprintf(ap.Log, "Unable to marshal global args")
	}
	fmt.Fprintf(ap.Log, "%s\n", data)
	fmt.Fprintf(ap.Log, "```\n")

	return ap, err
}

var Population = 10
var isQuickTest = true
var showFileUpload = true
var host = fmt.Sprintf("https://%s:8080", cfg.DockerVM)
var autopilotRoot = "$GOPATH/src/decipher.com/oduploader/autopilot/cache"

//Set this to true to disable output
var Quietly = false

func Init() {
	flag.StringVar(&host, "url", host, "The URL at which to direct uploads/downloads")
	flag.StringVar(&autopilotRoot, "root", autopilotRoot, "The URL at which to direct uploads/downloads")
	flag.Parse()
}
